// Tests for the text and JSON renderers: alignment, field order, and the
// exact timestamp formats idpeek promises in its README.
package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sample() Decoded {
	return Decoded{
		Input:     "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Kind:      "ulid",
		Canonical: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Time:      &TimeInfo{UnixMilli: 1469922850259, RFC3339: "2016-07-30T23:54:10.259Z", Precision: "millisecond"},
		Fields:    []Field{{Name: "randomness", Value: "0xd6764c61efb99302bd5b", Detail: "80 bits"}},
		Equivalents: []Field{
			{Name: "as UUID", Value: "01563e3a-b5d3-d676-4c61-efb99302bd5b", Detail: "same 128 bits"},
		},
		Notes: []string{"example note"},
	}
}

func TestTimeFormattersAreUTCFixedWidth(t *testing.T) {
	if got := FormatTime(1469922850259); got != "2016-07-30T23:54:10.259Z" {
		t.Fatalf("FormatTime = %q", got)
	}
	// 1645557742 s + 1234500 ns: seven fractional digits, no rounding.
	if got := FormatTimeNano(1645557742001234500); got != "2022-02-22T19:22:22.0012345Z" {
		t.Fatalf("FormatTimeNano = %q", got)
	}
}

func TestTextIncludesEveryField(t *testing.T) {
	var buf bytes.Buffer
	Text(&buf, sample())
	out := buf.String()
	for _, want := range []string{"input", "kind", "timestamp", "randomness", "as UUID", "example note", "unix_ms 1469922850259"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

func TestTextOmitsCanonicalWhenIdenticalToInput(t *testing.T) {
	var buf bytes.Buffer
	Text(&buf, sample())
	if strings.Contains(buf.String(), "canonical") {
		t.Fatal("canonical row should be omitted when it equals the input")
	}
	d := sample()
	d.Input = strings.ToLower(d.Input)
	buf.Reset()
	Text(&buf, d)
	if !strings.Contains(buf.String(), "canonical") {
		t.Fatal("canonical row should appear when input was not canonical")
	}
}

func TestTextColumnsAreAligned(t *testing.T) {
	var buf bytes.Buffer
	Text(&buf, sample())
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// Every value column must start at the same offset: name width + 2.
	col := -1
	for _, line := range lines {
		idx := strings.Index(line, "  ")
		rest := line[idx:]
		start := idx + len(rest) - len(strings.TrimLeft(rest, " "))
		if col == -1 {
			col = start
		} else if start != col {
			t.Fatalf("misaligned line %q (col %d, want %d)", line, start, col)
		}
	}
}

func TestJSONIsOneCompactLine(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.HasSuffix(out, "\n") {
		t.Fatalf("JSON must be exactly one newline-terminated line: %q", out)
	}
}

func TestJSONRoundtripsAndOmitsEmpty(t *testing.T) {
	var buf bytes.Buffer
	d := sample()
	d.Notes = nil
	d.Equivalents = nil
	if err := JSON(&buf, d); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["kind"] != "ulid" {
		t.Fatalf("kind = %v", m["kind"])
	}
	if _, ok := m["notes"]; ok {
		t.Fatal("empty notes must be omitted")
	}
	tm := m["time"].(map[string]any)
	if tm["unix_ms"].(float64) != 1469922850259 {
		t.Fatalf("unix_ms = %v", tm["unix_ms"])
	}
}

func TestHexBytesFormatting(t *testing.T) {
	if got := HexBytes([]byte{0x00, 0xAB, 0x10}); got != "0x00ab10" {
		t.Fatalf("HexBytes = %q", got)
	}
}
