// Package convert implements lossless conversions between ID
// representations: ULID <-> UUID (both are 128 bits), UUIDv1 <-> UUIDv6
// (the same fields in sortable vs legacy order), and raw hex for every
// format. Conversions never invent data — anything lossy is refused.
package convert

import (
	"fmt"

	"github.com/JaydenCJ/idpeek/internal/detect"
	"github.com/JaydenCJ/idpeek/internal/ksuid"
	"github.com/JaydenCJ/idpeek/internal/snowflake"
	"github.com/JaydenCJ/idpeek/internal/ulid"
	"github.com/JaydenCJ/idpeek/internal/uuid"
)

// Targets lists the accepted --to values, in help order.
var Targets = []string{"uuid", "ulid", "hex", "uuidv6", "uuidv1"}

// Convert re-expresses input (any supported ID) in the target
// representation. epochSel only matters for Snowflake inputs.
func Convert(input, target, epochSel string) (string, error) {
	kind, err := detect.DetectKind(input)
	if err != nil {
		return "", err
	}
	switch target {
	case "hex":
		return toHex(input, kind, epochSel)
	case "uuid", "ulid":
		b, err := bits128(input, kind)
		if err != nil {
			return "", err
		}
		if target == "uuid" {
			u, _ := uuid.FromBytes(b)
			return u.Canonical(), nil
		}
		u, _ := ulid.FromBytes(b)
		return u.String(), nil
	case "uuidv6":
		return crossGregorian(input, kind, 1, 6)
	case "uuidv1":
		return crossGregorian(input, kind, 6, 1)
	}
	return "", fmt.Errorf("unknown target %q (want uuid|ulid|hex|uuidv6|uuidv1)", target)
}

// bits128 returns the raw 16 bytes of a 128-bit ID, refusing formats of a
// different width so nothing is silently truncated or padded.
func bits128(input, kind string) ([]byte, error) {
	switch kind {
	case detect.KindUUID:
		u, err := uuid.Parse(input)
		if err != nil {
			return nil, err
		}
		return u.Bytes[:], nil
	case detect.KindULID:
		u, err := ulid.Parse(input)
		if err != nil {
			return nil, err
		}
		return u.Bytes[:], nil
	case detect.KindKSUID:
		return nil, fmt.Errorf("%q is a KSUID (160 bits): cannot convert to a 128-bit format losslessly", input)
	case detect.KindSnowflake:
		return nil, fmt.Errorf("%q is a Snowflake (64 bits): cannot convert to a 128-bit format losslessly", input)
	}
	return nil, fmt.Errorf("unsupported kind %q", kind)
}

func toHex(input, kind, epochSel string) (string, error) {
	switch kind {
	case detect.KindUUID:
		u, err := uuid.Parse(input)
		if err != nil {
			return "", err
		}
		return u.Hex(), nil
	case detect.KindULID:
		u, err := ulid.Parse(input)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", u.Bytes[:]), nil
	case detect.KindKSUID:
		k, err := ksuid.Parse(input)
		if err != nil {
			return "", err
		}
		return k.Hex(), nil
	case detect.KindSnowflake:
		id, err := snowflake.Parse(input, epochSel)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%016x", id.Value), nil
	}
	return "", fmt.Errorf("unsupported kind %q", kind)
}

// crossGregorian converts between UUIDv1 and UUIDv6 by reordering the
// 60-bit Gregorian timestamp; clock sequence, node, and variant bits are
// preserved exactly, so converting back yields the original UUID.
func crossGregorian(input, kind string, from, to int) (string, error) {
	if kind != detect.KindUUID {
		return "", fmt.Errorf("%q: uuidv%d conversion needs a UUIDv%d input", input, to, from)
	}
	u, err := uuid.Parse(input)
	if err != nil {
		return "", err
	}
	if !u.IsRFCVariant() || u.Version() != from {
		return "", fmt.Errorf("%q: uuidv%d conversion needs a UUIDv%d input (got version %d)", input, to, from, u.Version())
	}
	ts, _ := u.Timestamp100ns()
	var out [16]byte
	copy(out[8:], u.Bytes[8:]) // clock_seq + node unchanged
	if to == 6 {
		high := uint32(ts >> 28)
		mid := uint16(ts >> 12)
		low := uint16(ts & 0x0FFF)
		out[0], out[1], out[2], out[3] = byte(high>>24), byte(high>>16), byte(high>>8), byte(high)
		out[4], out[5] = byte(mid>>8), byte(mid)
		out[6] = 0x60 | byte(low>>8)
		out[7] = byte(low)
	} else {
		low := uint32(ts)
		mid := uint16(ts >> 32)
		high := uint16(ts >> 48) // 12 bits
		out[0], out[1], out[2], out[3] = byte(low>>24), byte(low>>16), byte(low>>8), byte(low)
		out[4], out[5] = byte(mid>>8), byte(mid)
		out[6] = 0x10 | byte(high>>8)
		out[7] = byte(high)
	}
	res, _ := uuid.FromBytes(out[:])
	return res.Canonical(), nil
}
