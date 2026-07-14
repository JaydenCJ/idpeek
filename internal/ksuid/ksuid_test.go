// Tests for KSUID base62 decoding/encoding against the reference example
// from Segment's KSUID documentation, plus the 160-bit overflow bound.
package ksuid

import (
	"strings"
	"testing"
)

// docKSUID is the worked example from the reference KSUID README:
// timestamp 107608047, payload B5A1CD34B5F99D1154FB6853345C9735.
const docKSUID = "0ujtsYcgvSTl8PAuAdqWYSMnLOv"

// maxKSUID encodes 2^160-1, the largest value 20 bytes can hold.
const maxKSUID = "aWgEPTl1tmebfsQzFP4bxwgy80V"

func mustParse(t *testing.T, s string) KSUID {
	t.Helper()
	k, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return k
}

func TestDocExampleFields(t *testing.T) {
	k := mustParse(t, docKSUID)
	if ts := k.RawTimestamp(); ts != 107608047 {
		t.Fatalf("raw timestamp = %d, want 107608047", ts)
	}
	// 107608047 + 1400000000 = 2017-10-10T04:00:47Z.
	if u := k.Unix(); u != 1507608047 {
		t.Fatalf("unix = %d, want 1507608047", u)
	}
	want := "b5a1cd34b5f99d1154fb6853345c9735"
	got := ""
	for _, b := range k.Payload() {
		got += string("0123456789abcdef"[b>>4]) + string("0123456789abcdef"[b&0xF])
	}
	if got != want {
		t.Fatalf("payload = %s, want %s", got, want)
	}
}

func TestRoundtripString(t *testing.T) {
	if got := mustParse(t, docKSUID).String(); got != docKSUID {
		t.Fatalf("roundtrip = %q, want %q", got, docKSUID)
	}
}

func TestMaxValueRoundtrip(t *testing.T) {
	k := mustParse(t, maxKSUID)
	for _, b := range k.Bytes {
		if b != 0xFF {
			t.Fatalf("max KSUID bytes = %x, want all 0xff", k.Bytes)
		}
	}
	if k.String() != maxKSUID {
		t.Fatalf("max roundtrip = %q", k.String())
	}
}

func TestZeroValueRoundtripPadsWithZeros(t *testing.T) {
	zero := strings.Repeat("0", EncodedLen)
	k := mustParse(t, zero)
	if k.Bytes != [20]byte{} || k.String() != zero {
		t.Fatal("zero KSUID must be all-zero bytes and left-padded on encode")
	}
}

func TestRejectsOverflowAboveMax(t *testing.T) {
	// One above the max value: still 27 valid base62 chars, but 161 bits.
	over := "aWgEPTl1tmebfsQzFP4bxwgy80W"
	if _, err := Parse(over); err == nil {
		t.Fatal("value above 2^160-1 must be rejected")
	}
	// LooksLike stays shape-only, so detection can still classify it and
	// let Parse produce the precise overflow error.
	if !LooksLike(over) {
		t.Fatal("LooksLike should accept the shape and defer to Parse")
	}
}

func TestRejectsMalformedInput(t *testing.T) {
	bad := []string{"", docKSUID[:26], docKSUID + "0"} // wrong lengths
	for _, c := range []string{"-", "_", " ", "!"} {   // non-base62 chars
		bad = append(bad, c+docKSUID[1:])
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should fail", s)
		}
	}
}

func TestBase62IsCaseSensitive(t *testing.T) {
	// 'a' and 'A' are different digits in base62; folding case would
	// silently decode a different ID.
	k1 := mustParse(t, "aWgEPTl1tmebfsQzFP4bxwgy80V")
	k2 := mustParse(t, "AWgEPTl1tmebfsQzFP4bxwgy80V")
	if k1.Bytes == k2.Bytes {
		t.Fatal("case-folded KSUIDs must decode differently")
	}
}

func TestFromBytesMatchesParse(t *testing.T) {
	k := mustParse(t, docKSUID)
	back, err := FromBytes(k.Bytes[:])
	if err != nil || back.String() != docKSUID {
		t.Fatalf("FromBytes roundtrip: %v", err)
	}
	if _, err := FromBytes(make([]byte, 16)); err == nil {
		t.Fatal("FromBytes should reject 16 bytes")
	}
}

func TestHexCovers20Bytes(t *testing.T) {
	h := mustParse(t, docKSUID).Hex()
	if len(h) != 40 || !strings.HasPrefix(h, "0669f7ef") {
		t.Fatalf("Hex() = %q, want 40 digits starting 0669f7ef", h)
	}
}
