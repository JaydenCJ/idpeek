// Tests for RFC 9562 UUID parsing and field extraction. The version 1/6/7
// vectors come straight from RFC 9562 Appendix A/B, so a regression here
// means disagreeing with the standard itself.
package uuid

import (
	"strings"
	"testing"
)

// RFC 9562 test vectors (all encode 2022-02-22T19:22:22Z, node
// 9f6bdeced846, clock sequence 13256).
const (
	vecV1 = "C232AB00-9414-11EC-B3C8-9F6BDECED846"
	vecV6 = "1EC9414C-232A-6B00-B3C8-9F6BDECED846"
	vecV7 = "017F22E2-79B0-7CC3-98C4-DC0C0C07398F"
	vecV4 = "919108f7-52d1-4320-9bac-f847db4148a8"
	vecV3 = "5df41881-3aed-3515-88a7-2f4a814cf09e"
	vecV5 = "2ed6657d-e927-568b-95e1-2665a8aea6a2"
)

func mustParse(t *testing.T, s string) UUID {
	t.Helper()
	u, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return u
}

func TestParseCanonicalLowercasesOutput(t *testing.T) {
	u := mustParse(t, vecV1)
	if got := u.Canonical(); got != strings.ToLower(vecV1) {
		t.Fatalf("Canonical() = %q, want lowercase form", got)
	}
}

func TestParseCompact32HexForm(t *testing.T) {
	u := mustParse(t, "017f22e279b07cc398c4dc0c0c07398f")
	if u.Canonical() != strings.ToLower(vecV7) {
		t.Fatalf("compact form decoded to %q", u.Canonical())
	}
	if u.Hex() != "017f22e279b07cc398c4dc0c0c07398f" {
		t.Fatalf("Hex() = %q, want 32 lowercase digits back", u.Hex())
	}
}

func TestParseAlternateForms(t *testing.T) {
	// The urn:uuid: prefix and Microsoft braces both appear in real logs.
	if u := mustParse(t, "urn:uuid:"+vecV7); u.Version() != 7 {
		t.Fatalf("urn form version = %d, want 7", u.Version())
	}
	if u := mustParse(t, "{"+vecV4+"}"); u.Version() != 4 {
		t.Fatalf("braced form version = %d, want 4", u.Version())
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	bad := []string{
		"017f22e2-79b0-7cc3-98c4d-c0c0c07398f", // 36 chars, dashes shifted by one
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398g", // 'g' is not hex
		"",
		"abc",
		strings.Repeat("0", 31), // one short of compact
		strings.Repeat("0", 33), // one past compact
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should fail", s)
		}
	}
}

func TestVersionNumbersFromVectors(t *testing.T) {
	for want, s := range map[int]string{
		1: vecV1, 3: vecV3, 4: vecV4, 5: vecV5, 6: vecV6, 7: vecV7,
	} {
		if got := mustParse(t, s).Version(); got != want {
			t.Errorf("Version(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestVariantDecoding(t *testing.T) {
	cases := []struct {
		octet8 byte
		want   string
	}{
		{0x00, VariantNCS},       // 0xx
		{0x7F, VariantNCS},       // 0xx upper edge
		{0x80, VariantRFC},       // 10x
		{0xBF, VariantRFC},       // 10x upper edge
		{0xC0, VariantMicrosoft}, // 110
		{0xE0, VariantFuture},    // 111
	}
	for _, c := range cases {
		var u UUID
		u.Bytes[8] = c.octet8
		if got := u.Variant(); got != c.want {
			t.Errorf("octet8 %#02x: variant %q, want %q", c.octet8, got, c.want)
		}
	}
}

func TestNilAndMaxSpecials(t *testing.T) {
	nilU := mustParse(t, "00000000-0000-0000-0000-000000000000")
	maxU := mustParse(t, "ffffffff-ffff-ffff-ffff-ffffffffffff")
	if !nilU.IsNil() || nilU.IsMax() {
		t.Fatal("nil UUID misclassified")
	}
	if !maxU.IsMax() || maxU.IsNil() {
		t.Fatal("max UUID misclassified")
	}
}

func TestV1TimestampMatchesRFCVector(t *testing.T) {
	u := mustParse(t, vecV1)
	ts, ok := u.Timestamp100ns()
	if !ok || ts != 138648505420000000 {
		t.Fatalf("v1 ts = %d ok=%v, want 138648505420000000", ts, ok)
	}
	ns, ok := u.UnixNano()
	if !ok || ns != 1645557742*1e9 {
		t.Fatalf("v1 unix ns = %d ok=%v, want 2022-02-22T19:22:22Z", ns, ok)
	}
}

func TestV6TimestampEqualsV1Vector(t *testing.T) {
	// RFC 9562 defines the v6 vector as the v1 vector's fields reordered,
	// so both must decode to the same 60-bit timestamp.
	v1ts, _ := mustParse(t, vecV1).Timestamp100ns()
	v6ts, ok := mustParse(t, vecV6).Timestamp100ns()
	if !ok || v6ts != v1ts {
		t.Fatalf("v6 ts = %d, want %d", v6ts, v1ts)
	}
}

func TestV7FieldsMatchRFCVector(t *testing.T) {
	u := mustParse(t, vecV7)
	if ms := u.UnixMilliV7(); ms != 1645557742000 {
		t.Fatalf("v7 unix ms = %d, want 1645557742000", ms)
	}
	if u.RandA() != 0xCC3 {
		t.Fatalf("rand_a = %#x, want 0xcc3", u.RandA())
	}
	if u.RandB() != 0x18C4DC0C0C07398F {
		t.Fatalf("rand_b = %#x, want 0x18c4dc0c0c07398f", u.RandB())
	}
}

func TestClockSeqAndNodeFromVector(t *testing.T) {
	u := mustParse(t, vecV1)
	if u.ClockSeq() != 13256 {
		t.Fatalf("clock_seq = %d, want 13256", u.ClockSeq())
	}
	if got := u.Node(); string(got) != "\x9f\x6b\xde\xce\xd8\x46" {
		t.Fatalf("node = %x", got)
	}
	// 0x9f has bit 0 set: the RFC vector uses a randomized (multicast) node.
	if !u.NodeIsMulticast() {
		t.Fatal("vector node 9f6bdeced846 has the multicast bit set")
	}
	u.Bytes[10] &^= 0x01
	if u.NodeIsMulticast() {
		t.Fatal("multicast bit cleared but still detected")
	}
}

func TestUnixNanoRefusedForNonTimeVersions(t *testing.T) {
	for _, s := range []string{vecV3, vecV4, vecV5} {
		if _, ok := mustParse(t, s).UnixNano(); ok {
			t.Errorf("%q carries no timestamp but UnixNano said ok", s)
		}
	}
}

func TestUnixNanoRefusedForPre1970Timestamp(t *testing.T) {
	// A v1 UUID whose Gregorian timestamp is zero (1582-10-15) predates the
	// Unix epoch; reporting it as a huge negative time would be misleading.
	u := mustParse(t, "00000000-0000-1000-8000-000000000000")
	if _, ok := u.UnixNano(); ok {
		t.Fatal("pre-1970 timestamp should be refused")
	}
}

func TestDCEFieldsForV2(t *testing.T) {
	// Synthetic v2 UUID: local ID 501 (a typical first UID), domain 0.
	u := mustParse(t, "000001f5-abcd-21ec-8000-9f6bdeced846")
	if u.Version() != 2 {
		t.Fatalf("version = %d, want 2", u.Version())
	}
	if id := u.DCELocalID(); id != 501 {
		t.Fatalf("local id = %d, want 501", id)
	}
	d, name := u.DCEDomain()
	if d != 0 || !strings.Contains(name, "person") {
		t.Fatalf("domain = %d %q, want 0 person", d, name)
	}
}

func TestFromBytesRoundtrip(t *testing.T) {
	u := mustParse(t, vecV7)
	back, err := FromBytes(u.Bytes[:])
	if err != nil || back != u {
		t.Fatalf("FromBytes roundtrip failed: %v", err)
	}
	if _, err := FromBytes([]byte{1, 2, 3}); err == nil {
		t.Fatal("FromBytes should reject short input")
	}
}

func TestHexIs32LowercaseDigits(t *testing.T) {
	got := mustParse(t, vecV7).Hex()
	if got != "017f22e279b07cc398c4dc0c0c07398f" {
		t.Fatalf("Hex() = %q", got)
	}
}

func TestVersionNameCoversAllDefinedVersions(t *testing.T) {
	for v := 1; v <= 8; v++ {
		if VersionName(v) == "unknown" {
			t.Errorf("version %d has no name", v)
		}
	}
	if VersionName(0) != "unknown" || VersionName(9) != "unknown" {
		t.Error("undefined versions must report unknown")
	}
}
