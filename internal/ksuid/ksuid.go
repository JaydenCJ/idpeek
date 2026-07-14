// Package ksuid decodes and encodes KSUIDs (K-Sortable Unique IDentifiers,
// as popularized by Segment): 20 bytes — a 32-bit big-endian timestamp of
// seconds since the KSUID epoch (2014-05-13T16:53:20Z) followed by a
// 128-bit random payload — rendered as exactly 27 base62 characters.
// Pure functions, no I/O; base62 math is done on the byte array directly,
// so there is no math/big dependency.
package ksuid

import (
	"encoding/hex"
	"fmt"
)

// Epoch is the KSUID epoch as Unix seconds (1400000000).
const Epoch int64 = 1400000000

// alphabet is base62 in ASCII order: 0-9, A-Z, a-z.
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// EncodedLen is the fixed string length of a KSUID.
const EncodedLen = 27

var decodeMap = func() [256]byte {
	var m [256]byte
	for i := range m {
		m[i] = 0xFF
	}
	for v, c := range []byte(alphabet) {
		m[c] = byte(v)
	}
	return m
}()

// KSUID is a parsed 160-bit KSUID.
type KSUID struct {
	Bytes [20]byte
}

// Parse decodes a 27-character base62 string, rejecting values that
// overflow 160 bits (anything above "aWgEPTl1tmebfsQzFP4bxwgy80V").
func Parse(s string) (KSUID, error) {
	if len(s) != EncodedLen {
		return KSUID{}, fmt.Errorf("%q: not a KSUID (want 27 chars, got %d)", s, len(s))
	}
	var k KSUID
	for i := 0; i < len(s); i++ {
		v := decodeMap[s[i]]
		if v == 0xFF {
			return KSUID{}, fmt.Errorf("%q: not a KSUID (invalid character %q at index %d)", s, s[i], i)
		}
		// k = k*62 + v, big-endian, carrying through all 20 bytes.
		carry := uint16(v)
		for j := 19; j >= 0; j-- {
			carry += uint16(k.Bytes[j]) * 62
			k.Bytes[j] = byte(carry)
			carry >>= 8
		}
		if carry != 0 {
			return KSUID{}, fmt.Errorf("%q: not a KSUID (value overflows 160 bits)", s)
		}
	}
	return k, nil
}

// FromBytes builds a KSUID from exactly 20 bytes.
func FromBytes(b []byte) (KSUID, error) {
	if len(b) != 20 {
		return KSUID{}, fmt.Errorf("ksuid needs 20 bytes, got %d", len(b))
	}
	var k KSUID
	copy(k.Bytes[:], b)
	return k, nil
}

// String encodes the KSUID back to its canonical 27-char base62 form,
// left-padded with '0'.
func (k KSUID) String() string {
	quo := k.Bytes // repeatedly divided in place
	var out [EncodedLen]byte
	for i := EncodedLen - 1; i >= 0; i-- {
		var rem uint16
		for j := 0; j < 20; j++ {
			cur := rem<<8 | uint16(quo[j])
			quo[j] = byte(cur / 62)
			rem = cur % 62
		}
		out[i] = alphabet[rem]
	}
	return string(out[:])
}

// RawTimestamp returns the 32-bit timestamp field (seconds since Epoch).
func (k KSUID) RawTimestamp() uint32 {
	b := k.Bytes
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// Unix returns the embedded creation time as Unix seconds.
func (k KSUID) Unix() int64 { return int64(k.RawTimestamp()) + Epoch }

// Payload returns the 128-bit random payload as 16 bytes.
func (k KSUID) Payload() []byte { return append([]byte(nil), k.Bytes[4:20]...) }

// Hex returns all 20 bytes as 40 lowercase hex digits.
func (k KSUID) Hex() string { return hex.EncodeToString(k.Bytes[:]) }

// LooksLike reports whether s has the shape of a KSUID (27 base62 chars)
// without checking the 160-bit overflow bound. Used by format
// auto-detection; Parse still enforces the bound.
func LooksLike(s string) bool {
	if len(s) != EncodedLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if decodeMap[s[i]] == 0xFF {
			return false
		}
	}
	return true
}
