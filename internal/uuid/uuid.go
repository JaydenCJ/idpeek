// Package uuid parses RFC 9562 (formerly RFC 4122) UUIDs and exposes their
// version, variant, and every embedded field — including the Gregorian
// timestamps of versions 1/6, the Unix timestamp of version 7, and the DCE
// Security fields of version 2. Pure functions, no I/O, no randomness.
package uuid

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// UUID is a parsed 128-bit UUID.
type UUID struct {
	Bytes [16]byte
}

// Variant constants as defined by RFC 9562 §4.1.
const (
	VariantNCS       = "NCS (reserved, backward compatibility)"
	VariantRFC       = "OSF DCE / RFC 9562"
	VariantMicrosoft = "Microsoft (reserved, backward compatibility)"
	VariantFuture    = "reserved for future definition"
)

// gregorianToUnix100ns is the number of 100 ns intervals between the
// Gregorian epoch (1582-10-15T00:00:00Z) and the Unix epoch.
const gregorianToUnix100ns = 122192928000000000

// Parse accepts the canonical 8-4-4-4-12 form, the 32-hex-digit compact
// form, the urn:uuid: prefix, and Microsoft-style braces. Case-insensitive.
func Parse(s string) (UUID, error) {
	orig := s
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	if len(s) >= 9 && strings.EqualFold(s[:9], "urn:uuid:") {
		s = s[9:]
	}
	var hexOnly string
	switch len(s) {
	case 36:
		if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
			return UUID{}, fmt.Errorf("%q: misplaced dashes for a canonical UUID", orig)
		}
		hexOnly = s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	case 32:
		hexOnly = s
	default:
		return UUID{}, fmt.Errorf("%q: not a UUID (want 36 chars with dashes or 32 hex digits)", orig)
	}
	raw, err := hex.DecodeString(strings.ToLower(hexOnly))
	if err != nil {
		return UUID{}, fmt.Errorf("%q: not a UUID (non-hex digit)", orig)
	}
	var u UUID
	copy(u.Bytes[:], raw)
	return u, nil
}

// FromBytes builds a UUID from exactly 16 bytes.
func FromBytes(b []byte) (UUID, error) {
	if len(b) != 16 {
		return UUID{}, fmt.Errorf("uuid needs 16 bytes, got %d", len(b))
	}
	var u UUID
	copy(u.Bytes[:], b)
	return u, nil
}

// Canonical returns the lowercase 8-4-4-4-12 representation.
func (u UUID) Canonical() string {
	h := hex.EncodeToString(u.Bytes[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// Hex returns the 32-digit compact representation.
func (u UUID) Hex() string { return hex.EncodeToString(u.Bytes[:]) }

// IsNil reports whether every bit is zero (the Nil UUID).
func (u UUID) IsNil() bool { return u.Bytes == [16]byte{} }

// IsMax reports whether every bit is one (the Max UUID, RFC 9562 §5.10).
func (u UUID) IsMax() bool {
	for _, b := range u.Bytes {
		if b != 0xFF {
			return false
		}
	}
	return true
}

// Variant decodes the variant bits in octet 8.
func (u UUID) Variant() string {
	b := u.Bytes[8]
	switch {
	case b&0x80 == 0:
		return VariantNCS
	case b&0xC0 == 0x80:
		return VariantRFC
	case b&0xE0 == 0xC0:
		return VariantMicrosoft
	default:
		return VariantFuture
	}
}

// IsRFCVariant reports whether the variant bits are 10x (the only variant
// for which the version field is meaningful).
func (u UUID) IsRFCVariant() bool { return u.Bytes[8]&0xC0 == 0x80 }

// Version returns the version nibble (octet 6, high nibble). Only
// meaningful for the RFC variant; check IsRFCVariant first.
func (u UUID) Version() int { return int(u.Bytes[6] >> 4) }

// VersionName returns the RFC 9562 name for a version number.
func VersionName(v int) string {
	switch v {
	case 1:
		return "Gregorian time-based"
	case 2:
		return "DCE Security"
	case 3:
		return "name-based (MD5)"
	case 4:
		return "random"
	case 5:
		return "name-based (SHA-1)"
	case 6:
		return "reordered Gregorian time-based"
	case 7:
		return "Unix-epoch time-based"
	case 8:
		return "custom / vendor-specific"
	default:
		return "unknown"
	}
}

// Timestamp100ns returns the 60-bit count of 100 ns intervals since the
// Gregorian epoch for v1/v6, with ok=false for versions that embed no
// full-resolution Gregorian timestamp.
func (u UUID) Timestamp100ns() (uint64, bool) {
	if !u.IsRFCVariant() {
		return 0, false
	}
	b := u.Bytes
	switch u.Version() {
	case 1:
		low := uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
		mid := uint64(b[4])<<8 | uint64(b[5])
		high := uint64(b[6]&0x0F)<<8 | uint64(b[7])
		return high<<48 | mid<<32 | low, true
	case 6:
		high := uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
		mid := uint64(b[4])<<8 | uint64(b[5])
		low := uint64(b[6]&0x0F)<<8 | uint64(b[7])
		return high<<28 | mid<<12 | low, true
	default:
		return 0, false
	}
}

// UnixNano converts a v1/v6/v7 embedded timestamp to Unix nanoseconds.
// ok=false when the version carries no timestamp (or predates 1970, which
// cannot be represented losslessly here and is out of scope for v0.1.0).
func (u UUID) UnixNano() (int64, bool) {
	if !u.IsRFCVariant() {
		return 0, false
	}
	switch u.Version() {
	case 1, 6:
		ts, ok := u.Timestamp100ns()
		if !ok || ts < gregorianToUnix100ns {
			return 0, false
		}
		return int64(ts-gregorianToUnix100ns) * 100, true
	case 7:
		ms := u.UnixMilliV7()
		return ms * 1_000_000, true
	default:
		return 0, false
	}
}

// UnixMilliV7 returns the 48-bit big-endian Unix millisecond timestamp of a
// version 7 UUID. Callers must ensure Version() == 7.
func (u UUID) UnixMilliV7() int64 {
	b := u.Bytes
	return int64(b[0])<<40 | int64(b[1])<<32 | int64(b[2])<<24 |
		int64(b[3])<<16 | int64(b[4])<<8 | int64(b[5])
}

// ClockSeq returns the 14-bit clock sequence of v1/v6 UUIDs.
func (u UUID) ClockSeq() uint16 {
	return uint16(u.Bytes[8]&0x3F)<<8 | uint16(u.Bytes[9])
}

// Node returns the 48-bit node field of v1/v6 UUIDs as 6 bytes.
func (u UUID) Node() []byte { return append([]byte(nil), u.Bytes[10:16]...) }

// NodeIsMulticast reports whether the multicast/local bit of the node field
// is set — RFC 9562 requires it for randomly generated node IDs, so a set
// bit means "not a real MAC address".
func (u UUID) NodeIsMulticast() bool { return u.Bytes[10]&0x01 == 1 }

// DCEDomain returns the DCE Security domain octet of a v2 UUID with a
// human-readable name (person/group/org per DCE 1.1).
func (u UUID) DCEDomain() (byte, string) {
	d := u.Bytes[9]
	switch d {
	case 0:
		return d, "person (POSIX UID)"
	case 1:
		return d, "group (POSIX GID)"
	case 2:
		return d, "org"
	default:
		return d, "site-defined"
	}
}

// DCELocalID returns the 32-bit local identifier (UID/GID) that a v2 UUID
// stores in place of time_low.
func (u UUID) DCELocalID() uint32 {
	b := u.Bytes
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// RandA returns the 12-bit rand_a field of a v7 UUID.
func (u UUID) RandA() uint16 { return uint16(u.Bytes[6]&0x0F)<<8 | uint16(u.Bytes[7]) }

// RandB returns the 62-bit rand_b field of a v7 UUID.
func (u UUID) RandB() uint64 {
	b := u.Bytes
	v := uint64(b[8]&0x3F)<<56 | uint64(b[9])<<48 | uint64(b[10])<<40 |
		uint64(b[11])<<32 | uint64(b[12])<<24 | uint64(b[13])<<16 |
		uint64(b[14])<<8 | uint64(b[15])
	return v
}
