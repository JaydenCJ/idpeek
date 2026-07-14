// Tests for ULID decoding/encoding against the reference example from the
// ULID specification, plus the Crockford base32 leniency rules.
package ulid

import (
	"strings"
	"testing"
)

// specULID is the canonical example from the ULID spec README.
const specULID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func mustParse(t *testing.T, s string) ULID {
	t.Helper()
	u, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return u
}

func TestSpecExampleFields(t *testing.T) {
	u := mustParse(t, specULID)
	if ms := u.UnixMilli(); ms != 1469922850259 {
		t.Fatalf("timestamp = %d, want 1469922850259 (2016-07-30T23:54:10.259Z)", ms)
	}
	want := "\xd6\x76\x4c\x61\xef\xb9\x93\x02\xbd\x5b"
	if got := u.Randomness(); string(got) != want {
		t.Fatalf("randomness = %x, want %x", got, want)
	}
}

func TestRoundtripStringIsCanonical(t *testing.T) {
	if got := mustParse(t, specULID).String(); got != specULID {
		t.Fatalf("roundtrip = %q, want %q", got, specULID)
	}
}

func TestLowercaseInputAccepted(t *testing.T) {
	u := mustParse(t, strings.ToLower(specULID))
	if u.String() != specULID {
		t.Fatal("lowercase input must canonicalize to uppercase")
	}
}

func TestCrockfordAliasesDecode(t *testing.T) {
	// Crockford base32 treats I/L as 1 and O as 0, so a hand-typed ULID
	// with those characters must still decode to the same value.
	aliased := strings.NewReplacer("1", "L", "0", "O").Replace(specULID)
	if aliased == specULID {
		t.Fatal("test setup: alias replacement changed nothing")
	}
	if got := mustParse(t, aliased).String(); got != specULID {
		t.Fatalf("aliased form decoded to %q, want %q", got, specULID)
	}
}

func TestRejectsInvalidInput(t *testing.T) {
	bad := []string{
		"U" + specULID[1:], // 'U' is excluded from the ULID alphabet
		"",
		specULID[:25],      // one char short
		specULID + "0",     // one char long
		"8" + specULID[1:], // largest valid ULID starts with '7': 129 bits
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should fail", s)
		}
	}
}

func TestExtremesRoundtrip(t *testing.T) {
	max := "7ZZZZZZZZZZZZZZZZZZZZZZZZZ"
	u := mustParse(t, max)
	for _, b := range u.Bytes {
		if b != 0xFF {
			t.Fatalf("max ULID bytes = %x, want all 0xff", u.Bytes)
		}
	}
	if u.String() != max {
		t.Fatalf("max roundtrip = %q", u.String())
	}
	zero := strings.Repeat("0", 26)
	z := mustParse(t, zero)
	if z.Bytes != [16]byte{} || z.String() != zero {
		t.Fatal("zero ULID must be all-zero bytes and roundtrip")
	}
}

func TestFromBytesMatchesParse(t *testing.T) {
	u := mustParse(t, specULID)
	back, err := FromBytes(u.Bytes[:])
	if err != nil || back.String() != specULID {
		t.Fatalf("FromBytes roundtrip: %v", err)
	}
	if _, err := FromBytes(make([]byte, 15)); err == nil {
		t.Fatal("FromBytes should reject 15 bytes")
	}
}

func TestLooksLikeAgreesWithParse(t *testing.T) {
	cases := map[string]bool{
		specULID:                     true,
		strings.ToLower(specULID):    true,
		"7ZZZZZZZZZZZZZZZZZZZZZZZZZ": true,
		"8ZZZZZZZZZZZZZZZZZZZZZZZZZ": false, // overflow
		specULID[:25]:                false, // short
		specULID + "A":               false, // long
		"01ARZ3NDEKTSV4RRFFQ69G5FAU": false, // 'U' invalid
	}
	for s, want := range cases {
		if got := LooksLike(s); got != want {
			t.Errorf("LooksLike(%q) = %v, want %v", s, got, want)
		}
		_, err := Parse(s)
		if (err == nil) != want {
			t.Errorf("Parse(%q) success = %v, want %v", s, err == nil, want)
		}
	}
}

func TestTimestampIs48Bits(t *testing.T) {
	// All-one timestamp, zero randomness: exactly 2^48-1 milliseconds.
	u := mustParse(t, "7ZZZZZZZZZ0000000000000000")
	if ms := u.UnixMilli(); ms != (1<<48)-1 {
		t.Fatalf("timestamp = %d, want 2^48-1", ms)
	}
}
