// Package ulid decodes and encodes ULIDs (Universally Unique
// Lexicographically Sortable Identifiers, spec: ulid/spec on GitHub):
// 26 characters of Crockford base32 wrapping a 48-bit Unix-millisecond
// timestamp plus 80 bits of randomness. Pure functions, no I/O.
package ulid

import "fmt"

// Alphabet is Crockford base32: digits plus uppercase letters excluding
// I, L, O, and U to avoid misreads.
const Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// decodeMap maps an ASCII byte to its 5-bit value, with Crockford's lenient
// aliases (i/l -> 1, o -> 0) and case folding. 0xFF marks invalid bytes.
var decodeMap = func() [256]byte {
	var m [256]byte
	for i := range m {
		m[i] = 0xFF
	}
	for v, c := range []byte(Alphabet) {
		m[c] = byte(v)
		m[c|0x20] = byte(v) // lowercase alias
	}
	for _, c := range []byte("iIlL") {
		m[c] = 1
	}
	for _, c := range []byte("oO") {
		m[c] = 0
	}
	return m
}()

// ULID is a parsed 128-bit ULID.
type ULID struct {
	Bytes [16]byte
}

// Parse decodes a 26-character Crockford base32 string. It accepts
// lowercase and the standard i/l/o aliases, and rejects strings whose top
// two bits are set (values above 2^128-1, i.e. first char > '7').
func Parse(s string) (ULID, error) {
	if len(s) != 26 {
		return ULID{}, fmt.Errorf("%q: not a ULID (want 26 chars, got %d)", s, len(s))
	}
	first := decodeMap[s[0]]
	if first == 0xFF {
		return ULID{}, fmt.Errorf("%q: not a ULID (invalid character %q)", s, s[0])
	}
	if first > 7 {
		return ULID{}, fmt.Errorf("%q: not a ULID (first char above '7' overflows 128 bits)", s)
	}
	var hi, lo uint64 // hi = top 64 bits, lo = bottom 64 bits
	for i := 0; i < len(s); i++ {
		v := decodeMap[s[i]]
		if v == 0xFF {
			return ULID{}, fmt.Errorf("%q: not a ULID (invalid character %q at index %d)", s, s[i], i)
		}
		hi = hi<<5 | lo>>59
		lo = lo<<5 | uint64(v)
	}
	var u ULID
	for i := 0; i < 8; i++ {
		u.Bytes[i] = byte(hi >> (56 - 8*i))
		u.Bytes[8+i] = byte(lo >> (56 - 8*i))
	}
	return u, nil
}

// FromBytes builds a ULID from exactly 16 bytes.
func FromBytes(b []byte) (ULID, error) {
	if len(b) != 16 {
		return ULID{}, fmt.Errorf("ulid needs 16 bytes, got %d", len(b))
	}
	var u ULID
	copy(u.Bytes[:], b)
	return u, nil
}

// String encodes the ULID back to its canonical uppercase 26-char form.
func (u ULID) String() string {
	var hi, lo uint64
	for i := 0; i < 8; i++ {
		hi = hi<<8 | uint64(u.Bytes[i])
		lo = lo<<8 | uint64(u.Bytes[8+i])
	}
	var out [26]byte
	for i := 25; i >= 0; i-- {
		out[i] = Alphabet[lo&0x1F]
		lo = lo>>5 | hi<<59
		hi >>= 5
	}
	return string(out[:])
}

// UnixMilli returns the 48-bit embedded Unix-millisecond timestamp.
func (u ULID) UnixMilli() int64 {
	b := u.Bytes
	return int64(b[0])<<40 | int64(b[1])<<32 | int64(b[2])<<24 |
		int64(b[3])<<16 | int64(b[4])<<8 | int64(b[5])
}

// Randomness returns the 80-bit random payload as 10 bytes.
func (u ULID) Randomness() []byte { return append([]byte(nil), u.Bytes[6:16]...) }

// LooksLike reports whether s has the shape of a ULID (26 chars, all in the
// lenient Crockford set, first char <= '7') without allocating an error.
// Used by format auto-detection.
func LooksLike(s string) bool {
	if len(s) != 26 {
		return false
	}
	if v := decodeMap[s[0]]; v == 0xFF || v > 7 {
		return false
	}
	for i := 1; i < len(s); i++ {
		if decodeMap[s[i]] == 0xFF {
			return false
		}
	}
	return true
}
