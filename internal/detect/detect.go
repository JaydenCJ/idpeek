// Package detect classifies an input string as a UUID, ULID, KSUID, or
// Snowflake and builds the full render.Decoded inspection for it. Detection
// is purely structural (length + alphabet), so it is deterministic and
// never guesses between formats: the four shapes are mutually exclusive.
package detect

import (
	"fmt"
	"strings"

	"github.com/JaydenCJ/idpeek/internal/ksuid"
	"github.com/JaydenCJ/idpeek/internal/render"
	"github.com/JaydenCJ/idpeek/internal/snowflake"
	"github.com/JaydenCJ/idpeek/internal/ulid"
	"github.com/JaydenCJ/idpeek/internal/uuid"
)

// Kind names, also accepted by the --kind flag.
const (
	KindUUID      = "uuid"
	KindULID      = "ulid"
	KindKSUID     = "ksuid"
	KindSnowflake = "snowflake"
)

// looksLikeUUID mirrors the shapes uuid.Parse accepts.
func looksLikeUUID(s string) bool {
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	if len(s) >= 9 && strings.EqualFold(s[:9], "urn:uuid:") {
		s = s[9:]
	}
	if len(s) == 36 {
		return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
	}
	if len(s) != 32 {
		return false
	}
	for i := 0; i < 32; i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// DetectKind returns the structural kind of s, or an error listing the
// shapes idpeek understands.
func DetectKind(s string) (string, error) {
	switch {
	case looksLikeUUID(s):
		return KindUUID, nil
	case ulid.LooksLike(s):
		return KindULID, nil
	case ksuid.LooksLike(s):
		return KindKSUID, nil
	case snowflake.LooksLike(s):
		return KindSnowflake, nil
	}
	return "", fmt.Errorf("%q: unrecognized ID (want a UUID [36 chars / 32 hex], ULID [26 base32], KSUID [27 base62], or Snowflake [decimal int64])", s)
}

// Decode auto-detects the kind of input and inspects it. epochSel selects
// the Snowflake epoch (name or Unix-millisecond offset).
func Decode(input, epochSel string) (render.Decoded, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return render.Decoded{}, fmt.Errorf("empty input")
	}
	kind, err := DetectKind(s)
	if err != nil {
		return render.Decoded{}, err
	}
	return DecodeAs(s, kind, epochSel)
}

// DecodeAs inspects input as an explicitly chosen kind, bypassing
// auto-detection (the --kind flag).
func DecodeAs(input, kind, epochSel string) (render.Decoded, error) {
	s := strings.TrimSpace(input)
	switch kind {
	case KindUUID:
		u, err := uuid.Parse(s)
		if err != nil {
			return render.Decoded{}, err
		}
		return inspectUUID(s, u), nil
	case KindULID:
		u, err := ulid.Parse(s)
		if err != nil {
			return render.Decoded{}, err
		}
		return inspectULID(s, u), nil
	case KindKSUID:
		k, err := ksuid.Parse(s)
		if err != nil {
			return render.Decoded{}, err
		}
		return inspectKSUID(s, k), nil
	case KindSnowflake:
		id, err := snowflake.Parse(s, epochSel)
		if err != nil {
			return render.Decoded{}, err
		}
		return inspectSnowflake(s, id), nil
	}
	return render.Decoded{}, fmt.Errorf("unknown kind %q (want uuid|ulid|ksuid|snowflake)", kind)
}

func inspectUUID(input string, u uuid.UUID) render.Decoded {
	d := render.Decoded{Input: input, Kind: KindUUID, Canonical: u.Canonical()}
	switch {
	case u.IsNil():
		d.Fields = append(d.Fields, render.Field{Name: "special", Value: "Nil UUID", Detail: "all 128 bits zero, RFC 9562 §5.9"})
	case u.IsMax():
		d.Fields = append(d.Fields, render.Field{Name: "special", Value: "Max UUID", Detail: "all 128 bits one, RFC 9562 §5.10"})
	case !u.IsRFCVariant():
		d.Fields = append(d.Fields, render.Field{Name: "variant", Value: u.Variant()})
		d.Notes = append(d.Notes, "non-RFC variant: the version nibble is not meaningful")
	default:
		v := u.Version()
		d.Fields = append(d.Fields,
			render.Field{Name: "version", Value: fmt.Sprintf("%d", v), Detail: uuid.VersionName(v)},
			render.Field{Name: "variant", Value: u.Variant()},
		)
		inspectUUIDVersion(&d, u, v)
	}
	d.Equivalents = append(d.Equivalents,
		render.Field{Name: "as ULID", Value: mustULID(u.Bytes[:]).String(), Detail: "same 128 bits, Crockford base32"},
		render.Field{Name: "as hex", Value: render.HexBytes(u.Bytes[:])},
	)
	return d
}

func inspectUUIDVersion(d *render.Decoded, u uuid.UUID, v int) {
	switch v {
	case 1, 6:
		if ns, ok := u.UnixNano(); ok {
			d.Time = &render.TimeInfo{
				UnixMilli: ns / 1e6,
				RFC3339:   render.FormatTimeNano(ns),
				Precision: "100ns",
			}
		} else {
			d.Notes = append(d.Notes, "timestamp predates 1970-01-01 and is out of range for v0.1.0")
		}
		ts, _ := u.Timestamp100ns()
		d.Fields = append(d.Fields,
			render.Field{Name: "greg_100ns", Value: fmt.Sprintf("%d", ts), Detail: "100ns intervals since 1582-10-15"},
			render.Field{Name: "clock_seq", Value: fmt.Sprintf("%d", u.ClockSeq())},
			render.Field{Name: "node", Value: render.HexBytes(u.Node()), Detail: nodeDetail(u)},
		)
		if v == 1 {
			d.Notes = append(d.Notes, "v1 is not sortable; convert --to uuidv6 for a DB-friendly ordering")
		}
	case 2:
		domain, name := u.DCEDomain()
		d.Fields = append(d.Fields,
			render.Field{Name: "dce_domain", Value: fmt.Sprintf("%d", domain), Detail: name},
			render.Field{Name: "local_id", Value: fmt.Sprintf("%d", u.DCELocalID()), Detail: "UID/GID stored in place of time_low"},
			render.Field{Name: "node", Value: render.HexBytes(u.Node()), Detail: nodeDetail(u)},
		)
		d.Notes = append(d.Notes, "v2 truncates the timestamp to ~7 min resolution; idpeek does not report it as a time")
	case 3, 5:
		algo := "MD5"
		if v == 5 {
			algo = "SHA-1"
		}
		d.Fields = append(d.Fields, render.Field{Name: "hash", Value: algo, Detail: "digest of namespace UUID + name"})
		d.Notes = append(d.Notes, "name-based: deterministic for the same input, no embedded timestamp")
	case 4:
		d.Fields = append(d.Fields, render.Field{Name: "random_bits", Value: "122"})
		d.Notes = append(d.Notes, "random: no embedded timestamp")
	case 7:
		ms := u.UnixMilliV7()
		d.Time = &render.TimeInfo{UnixMilli: ms, RFC3339: render.FormatTime(ms), Precision: "millisecond"}
		d.Fields = append(d.Fields,
			render.Field{Name: "rand_a", Value: fmt.Sprintf("0x%03x", u.RandA()), Detail: "12 bits, often sub-ms counter"},
			render.Field{Name: "rand_b", Value: fmt.Sprintf("0x%016x", u.RandB()), Detail: "62 bits"},
		)
	case 8:
		d.Notes = append(d.Notes, "v8 layout is vendor-defined; only version/variant bits are standardized")
	default:
		d.Notes = append(d.Notes, fmt.Sprintf("version %d is not defined by RFC 9562", v))
	}
}

func nodeDetail(u uuid.UUID) string {
	if u.NodeIsMulticast() {
		return "multicast bit set: random node ID, not a MAC address"
	}
	return "unicast: may be a real MAC address"
}

func inspectULID(input string, u ulid.ULID) render.Decoded {
	ms := u.UnixMilli()
	d := render.Decoded{
		Input:     input,
		Kind:      KindULID,
		Canonical: u.String(),
		Time:      &render.TimeInfo{UnixMilli: ms, RFC3339: render.FormatTime(ms), Precision: "millisecond"},
		Fields: []render.Field{
			{Name: "randomness", Value: render.HexBytes(u.Randomness()), Detail: "80 bits"},
		},
	}
	uu := mustUUID(u.Bytes[:])
	uuidDetail := "same 128 bits"
	if uu.IsRFCVariant() && uu.Version() >= 1 && uu.Version() <= 8 {
		uuidDetail = fmt.Sprintf("same 128 bits; reads as UUIDv%d", uu.Version())
	}
	d.Equivalents = append(d.Equivalents,
		render.Field{Name: "as UUID", Value: uu.Canonical(), Detail: uuidDetail},
		render.Field{Name: "as hex", Value: render.HexBytes(u.Bytes[:])},
	)
	return d
}

func inspectKSUID(input string, k ksuid.KSUID) render.Decoded {
	sec := k.Unix()
	return render.Decoded{
		Input:     input,
		Kind:      KindKSUID,
		Canonical: k.String(),
		Time:      &render.TimeInfo{UnixMilli: sec * 1000, RFC3339: render.FormatTime(sec * 1000), Precision: "second"},
		Fields: []render.Field{
			{Name: "raw_ts", Value: fmt.Sprintf("%d", k.RawTimestamp()), Detail: "seconds since KSUID epoch 2014-05-13T16:53:20Z"},
			{Name: "payload", Value: render.HexBytes(k.Payload()), Detail: "128 bits"},
		},
		Equivalents: []render.Field{
			{Name: "as hex", Value: "0x" + k.Hex(), Detail: "all 160 bits"},
		},
	}
}

func inspectSnowflake(input string, id snowflake.ID) render.Decoded {
	ms := id.UnixMilli()
	d := render.Decoded{
		Input:     input,
		Kind:      KindSnowflake,
		Canonical: fmt.Sprintf("%d", id.Value),
		Time:      &render.TimeInfo{UnixMilli: ms, RFC3339: render.FormatTime(ms), Precision: "millisecond"},
		Fields: []render.Field{
			{Name: "epoch", Value: id.EpochName, Detail: fmt.Sprintf("offset %d ms; override with --epoch", id.EpochMS)},
			{Name: "ts_offset", Value: fmt.Sprintf("%d", id.TimestampOffsetMS()), Detail: "41-bit ms since epoch"},
			{Name: "datacenter", Value: fmt.Sprintf("%d", id.Datacenter), Detail: "bits 21-17"},
			{Name: "worker", Value: fmt.Sprintf("%d", id.Worker), Detail: "bits 16-12"},
			{Name: "machine_id", Value: fmt.Sprintf("%d", id.MachineID()), Detail: "combined 10-bit reading"},
			{Name: "sequence", Value: fmt.Sprintf("%d", id.Sequence), Detail: "bits 11-0"},
		},
	}
	for _, in := range id.Interpretations() {
		verdict := "implausible"
		if in.Plausible {
			verdict = "plausible"
		}
		d.Fields = append(d.Fields, render.Field{
			Name:   "time@" + in.Epoch,
			Value:  render.FormatTime(in.UnixMilli),
			Detail: verdict,
		})
	}
	d.Equivalents = append(d.Equivalents,
		render.Field{Name: "as hex", Value: fmt.Sprintf("0x%016x", id.Value)},
	)
	return d
}

// mustULID and mustUUID convert between the 128-bit types; the length is
// correct by construction, so failure is a programmer error.
func mustULID(b []byte) ulid.ULID {
	u, err := ulid.FromBytes(b)
	if err != nil {
		panic(err)
	}
	return u
}

func mustUUID(b []byte) uuid.UUID {
	u, err := uuid.FromBytes(b)
	if err != nil {
		panic(err)
	}
	return u
}
