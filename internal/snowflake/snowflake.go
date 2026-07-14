// Package snowflake decomposes 64-bit Snowflake IDs (the Twitter layout:
// 41-bit millisecond timestamp, 5-bit datacenter, 5-bit worker, 12-bit
// sequence) and interprets the timestamp against the well-known service
// epochs. Pure functions, no I/O, no wall clock.
package snowflake

import (
	"fmt"
	"sort"
	"strconv"
)

// Epoch is a named Snowflake epoch offset in Unix milliseconds.
type Epoch struct {
	Name   string
	Millis int64
}

// Well-known epochs. "unix" (offset 0) covers Snowflake dialects such as
// Sony's Sonyflake derivatives that count from the Unix epoch directly.
var epochs = map[string]int64{
	"twitter":   1288834974657, // 2010-11-04T01:42:54.657Z
	"discord":   1420070400000, // 2015-01-01T00:00:00Z
	"instagram": 1314220021721, // 2011-08-24T21:07:01.721Z
	"unix":      0,
}

// DefaultEpoch is used when the caller does not pick one explicitly.
const DefaultEpoch = "twitter"

// Plausibility window for an interpretation, in Unix milliseconds:
// [2007-01-01T00:00:00Z, 2040-01-01T00:00:00Z). Fixed constants keep the
// check fully deterministic — no wall clock.
const (
	plausibleMinMS = 1167609600000
	plausibleMaxMS = 2208988800000
)

// ID is a decomposed Snowflake.
type ID struct {
	Value      int64
	EpochName  string // resolved epoch name, or the raw millis for custom
	EpochMS    int64
	Datacenter int64 // bits 21-17
	Worker     int64 // bits 16-12
	Sequence   int64 // bits 11-0
}

// ResolveEpoch turns an epoch selector — a well-known name or a literal
// Unix-millisecond offset — into (name, millis).
func ResolveEpoch(sel string) (string, int64, error) {
	if ms, ok := epochs[sel]; ok {
		return sel, ms, nil
	}
	ms, err := strconv.ParseInt(sel, 10, 64)
	if err != nil || ms < 0 {
		return "", 0, fmt.Errorf("unknown epoch %q (want %s, or a Unix-millisecond offset)", sel, EpochNames())
	}
	return sel, ms, nil
}

// EpochNames returns the well-known epoch names, sorted, joined for help
// and error text (e.g. "discord|instagram|twitter|unix").
func EpochNames() string {
	names := make([]string, 0, len(epochs))
	for n := range epochs {
		names = append(names, n)
	}
	sort.Strings(names)
	out := ""
	for i, n := range names {
		if i > 0 {
			out += "|"
		}
		out += n
	}
	return out
}

// Parse decodes a decimal string as a Snowflake under the given epoch
// selector. Snowflakes are positive int64 values; zero and negatives are
// rejected, as are strings with non-digits or leading '+'.
func Parse(s, epochSel string) (ID, error) {
	if len(s) == 0 || len(s) > 20 {
		return ID{}, fmt.Errorf("%q: not a Snowflake (want 1-19 decimal digits)", s)
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return ID{}, fmt.Errorf("%q: not a Snowflake (non-digit at index %d)", s, i)
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return ID{}, fmt.Errorf("%q: not a Snowflake (overflows int64)", s)
	}
	if v <= 0 {
		return ID{}, fmt.Errorf("%q: not a Snowflake (must be positive)", s)
	}
	name, ms, err := ResolveEpoch(epochSel)
	if err != nil {
		return ID{}, err
	}
	return ID{
		Value:      v,
		EpochName:  name,
		EpochMS:    ms,
		Datacenter: v >> 17 & 0x1F,
		Worker:     v >> 12 & 0x1F,
		Sequence:   v & 0xFFF,
	}, nil
}

// TimestampOffsetMS returns the raw 41-bit field: milliseconds since the
// chosen epoch.
func (id ID) TimestampOffsetMS() int64 { return id.Value >> 22 }

// UnixMilli returns the embedded creation time under the chosen epoch.
func (id ID) UnixMilli() int64 { return id.TimestampOffsetMS() + id.EpochMS }

// MachineID returns the combined 10-bit machine field
// (datacenter<<5 | worker), which is how some dialects read those bits.
func (id ID) MachineID() int64 { return id.Value >> 12 & 0x3FF }

// Interpretation is the embedded time read under one well-known epoch.
type Interpretation struct {
	Epoch     string
	EpochMS   int64
	UnixMilli int64
	Plausible bool // date lands in [2007, 2040)
}

// Interpretations returns the timestamp read under every well-known epoch,
// sorted by epoch name, with a plausibility verdict for each. This is what
// lets `idpeek` say "as a Discord ID this would be 2016-04-30" without
// guessing.
func (id ID) Interpretations() []Interpretation {
	names := make([]string, 0, len(epochs))
	for n := range epochs {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Interpretation, 0, len(names))
	for _, n := range names {
		ms := id.TimestampOffsetMS() + epochs[n]
		out = append(out, Interpretation{
			Epoch:     n,
			EpochMS:   epochs[n],
			UnixMilli: ms,
			Plausible: ms >= plausibleMinMS && ms < plausibleMaxMS,
		})
	}
	return out
}

// LooksLike reports whether s is all decimal digits and fits a positive
// int64. Used by format auto-detection.
func LooksLike(s string) bool {
	if len(s) == 0 || len(s) > 19 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	return err == nil && v > 0
}
