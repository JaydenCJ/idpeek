// Package render defines the format-neutral decoding result (Decoded) and
// renders it as aligned human-readable text or as one JSON object per line
// (JSON Lines), so multiple IDs stream cleanly through pipes.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Field is one named property of a decoded ID. Order is meaningful and is
// preserved in both output formats.
type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Detail string `json:"detail,omitempty"`
}

// TimeInfo is the primary embedded timestamp of an ID.
type TimeInfo struct {
	UnixMilli int64  `json:"unix_ms"`
	RFC3339   string `json:"rfc3339"`
	Precision string `json:"precision"` // "100ns", "millisecond", "second"
}

// Decoded is everything idpeek knows about one input ID.
type Decoded struct {
	Input       string    `json:"input"`
	Kind        string    `json:"kind"` // "uuid", "ulid", "ksuid", "snowflake"
	Canonical   string    `json:"canonical"`
	Time        *TimeInfo `json:"time,omitempty"`
	Fields      []Field   `json:"fields"`
	Equivalents []Field   `json:"equivalents,omitempty"`
	Notes       []string  `json:"notes,omitempty"`
}

// FormatTime renders Unix milliseconds as RFC 3339 UTC with millisecond
// precision (idpeek always reports UTC).
func FormatTime(unixMilli int64) string {
	return time.UnixMilli(unixMilli).UTC().Format("2006-01-02T15:04:05.000Z")
}

// FormatTimeNano renders Unix nanoseconds as RFC 3339 UTC with 100 ns
// precision, matching the resolution of UUID v1/v6 timestamps.
func FormatTimeNano(unixNano int64) string {
	return time.Unix(unixNano/1e9, unixNano%1e9).UTC().Format("2006-01-02T15:04:05.0000000Z")
}

// Text writes one aligned key/value block for d.
func Text(w io.Writer, d Decoded) {
	width := 11
	for _, f := range append(append([]Field{}, d.Fields...), d.Equivalents...) {
		if len(f.Name) > width {
			width = len(f.Name)
		}
	}
	row := func(name, value, detail string) {
		if detail != "" {
			fmt.Fprintf(w, "%-*s  %s  (%s)\n", width, name, value, detail)
		} else {
			fmt.Fprintf(w, "%-*s  %s\n", width, name, value)
		}
	}
	row("input", d.Input, "")
	row("kind", d.Kind, "")
	if d.Canonical != d.Input {
		row("canonical", d.Canonical, "")
	}
	if d.Time != nil {
		row("timestamp", d.Time.RFC3339,
			fmt.Sprintf("unix_ms %d, %s precision", d.Time.UnixMilli, d.Time.Precision))
	}
	for _, f := range d.Fields {
		row(f.Name, f.Value, f.Detail)
	}
	for _, f := range d.Equivalents {
		row(f.Name, f.Value, f.Detail)
	}
	for _, n := range d.Notes {
		fmt.Fprintf(w, "%-*s  %s\n", width, "note", n)
	}
}

// JSON writes d as a single compact JSON object followed by a newline.
func JSON(w io.Writer, d Decoded) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(d)
}

// HexBytes formats a byte slice as lowercase hex with a 0x prefix.
func HexBytes(b []byte) string {
	var sb strings.Builder
	sb.WriteString("0x")
	for _, c := range b {
		fmt.Fprintf(&sb, "%02x", c)
	}
	return sb.String()
}
