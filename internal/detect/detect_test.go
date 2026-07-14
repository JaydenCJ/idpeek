// Tests for format auto-detection and the inspection glue that turns a
// parsed ID into a render.Decoded. Detection must be unambiguous: the four
// formats have mutually exclusive shapes.
package detect

import (
	"strings"
	"testing"
)

func TestDetectKindByShape(t *testing.T) {
	cases := map[string]string{
		"017F22E2-79B0-7CC3-98C4-DC0C0C07398F":          KindUUID,
		"017f22e279b07cc398c4dc0c0c07398f":              KindUUID,
		"urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f": KindUUID,
		"{919108f7-52d1-4320-9bac-f847db4148a8}":        KindUUID,
		"01ARZ3NDEKTSV4RRFFQ69G5FAV":                    KindULID,
		"0ujtsYcgvSTl8PAuAdqWYSMnLOv":                   KindKSUID,
		"1541815603606036480":                           KindSnowflake,
	}
	for input, want := range cases {
		got, err := DetectKind(input)
		if err != nil || got != want {
			t.Errorf("DetectKind(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestDetectKindRejectsGarbage(t *testing.T) {
	for _, s := range []string{"hello-world", "0", "-42", "zzzz", strings.Repeat("g", 32)} {
		if kind, err := DetectKind(s); err == nil {
			t.Errorf("DetectKind(%q) = %q, want error", s, kind)
		}
	}
}

func TestAllDigit26CharStringIsULIDNotSnowflake(t *testing.T) {
	// 26 digits are valid Crockford base32 but overflow int64, so this
	// must land on ULID — the only consistent reading.
	kind, err := DetectKind("01234567012345670123456701")
	if err != nil || kind != KindULID {
		t.Fatalf("kind = %q, %v; want ulid", kind, err)
	}
}

func TestDecodeTrimsWhitespaceAndRejectsBlank(t *testing.T) {
	d, err := Decode("  01ARZ3NDEKTSV4RRFFQ69G5FAV\n", "twitter")
	if err != nil || d.Kind != KindULID {
		t.Fatalf("whitespace-wrapped input failed: %v", err)
	}
	if _, err := Decode("   ", "twitter"); err == nil {
		t.Fatal("blank input must error")
	}
}

func TestDecodeAsForcesKind(t *testing.T) {
	// A 32-hex string auto-detects as UUID, but --kind can be used to read
	// other shapes explicitly; forcing a mismatched kind must error.
	if _, err := DecodeAs("01ARZ3NDEKTSV4RRFFQ69G5FAV", KindUUID, "twitter"); err == nil {
		t.Fatal("forcing uuid on a ULID string should fail to parse")
	}
	d, err := DecodeAs("175928847299117063", KindSnowflake, "discord")
	if err != nil || d.Time.UnixMilli != 1462015105796 {
		t.Fatalf("forced snowflake decode wrong: %+v, %v", d.Time, err)
	}
	if _, err := DecodeAs("1", "guid", "twitter"); err == nil {
		t.Fatal("unknown kind must error")
	}
}

func TestUUIDv7DecodedFieldsAndEquivalents(t *testing.T) {
	d, err := Decode("017F22E2-79B0-7CC3-98C4-DC0C0C07398F", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	if d.Time == nil || d.Time.UnixMilli != 1645557742000 || d.Time.Precision != "millisecond" {
		t.Fatalf("time = %+v", d.Time)
	}
	if d.Canonical != "017f22e2-79b0-7cc3-98c4-dc0c0c07398f" {
		t.Fatalf("canonical = %q", d.Canonical)
	}
	var asULID string
	for _, f := range d.Equivalents {
		if f.Name == "as ULID" {
			asULID = f.Value
		}
	}
	if asULID != "01FWHE4YDGFK1SHH6W1G60EECF" {
		t.Fatalf("as ULID = %q", asULID)
	}
}

func TestUUIDv1DecodedHas100nsPrecision(t *testing.T) {
	d, err := Decode("C232AB00-9414-11EC-B3C8-9F6BDECED846", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	if d.Time == nil || d.Time.Precision != "100ns" || d.Time.UnixMilli != 1645557742000 {
		t.Fatalf("time = %+v", d.Time)
	}
	if d.Time.RFC3339 != "2022-02-22T19:22:22.0000000Z" {
		t.Fatalf("rfc3339 = %q", d.Time.RFC3339)
	}
}

func TestNilUUIDReportsSpecialNoTime(t *testing.T) {
	d, err := Decode("00000000-0000-0000-0000-000000000000", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	if d.Time != nil {
		t.Fatal("nil UUID must not report a timestamp")
	}
	if len(d.Fields) == 0 || d.Fields[0].Value != "Nil UUID" {
		t.Fatalf("fields = %+v", d.Fields)
	}
}

func TestUUIDv4ReportsNoTimestampNote(t *testing.T) {
	d, err := Decode("919108f7-52d1-4320-9bac-f847db4148a8", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	if d.Time != nil {
		t.Fatal("v4 must not report a timestamp")
	}
	found := false
	for _, n := range d.Notes {
		if strings.Contains(n, "no embedded timestamp") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notes = %v, want a no-timestamp note", d.Notes)
	}
}

func TestULIDDecodedRoundtripsThroughUUIDEquivalent(t *testing.T) {
	d, err := Decode("01ARZ3NDEKTSV4RRFFQ69G5FAV", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	var asUUID string
	for _, f := range d.Equivalents {
		if f.Name == "as UUID" {
			asUUID = f.Value
		}
	}
	if asUUID != "01563e3a-b5d3-d676-4c61-efb99302bd5b" {
		t.Fatalf("as UUID = %q", asUUID)
	}
}

func TestKSUIDDecodedSecondPrecision(t *testing.T) {
	d, err := Decode("0ujtsYcgvSTl8PAuAdqWYSMnLOv", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	if d.Time == nil || d.Time.Precision != "second" || d.Time.UnixMilli != 1507608047000 {
		t.Fatalf("time = %+v", d.Time)
	}
}

func TestSnowflakeDecodedListsAllInterpretations(t *testing.T) {
	d, err := Decode("1541815603606036480", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, f := range d.Fields {
		if strings.HasPrefix(f.Name, "time@") {
			count++
		}
	}
	if count != 4 {
		t.Fatalf("got %d time@ interpretations, want 4", count)
	}
}

func TestSnowflakeEpochSelectionChangesPrimaryTime(t *testing.T) {
	twitter, err := Decode("175928847299117063", "twitter")
	if err != nil {
		t.Fatal(err)
	}
	discord, err := Decode("175928847299117063", "discord")
	if err != nil {
		t.Fatal(err)
	}
	if twitter.Time.UnixMilli == discord.Time.UnixMilli {
		t.Fatal("different epochs must yield different primary timestamps")
	}
	if discord.Time.UnixMilli != 1462015105796 {
		t.Fatalf("discord time = %d", discord.Time.UnixMilli)
	}
}
