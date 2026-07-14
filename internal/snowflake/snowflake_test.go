// Tests for Snowflake decomposition and epoch interpretation. The Twitter
// vector is a real public tweet ID; the Discord vector is the worked
// example from Discord's API documentation.
package snowflake

import "testing"

func mustParse(t *testing.T, s, epoch string) ID {
	t.Helper()
	id, err := Parse(s, epoch)
	if err != nil {
		t.Fatalf("Parse(%q, %q): %v", s, epoch, err)
	}
	return id
}

func TestTwitterVector(t *testing.T) {
	id := mustParse(t, "1541815603606036480", "twitter")
	if ms := id.UnixMilli(); ms != 1656432460105 {
		t.Fatalf("unix ms = %d, want 1656432460105 (2022-06-28T16:07:40.105Z)", ms)
	}
	if id.Datacenter != 11 || id.Worker != 26 || id.Sequence != 0 {
		t.Fatalf("dc/worker/seq = %d/%d/%d, want 11/26/0", id.Datacenter, id.Worker, id.Sequence)
	}
	if id.MachineID() != 11<<5|26 {
		t.Fatalf("machine id = %d, want %d", id.MachineID(), 11<<5|26)
	}
}

func TestDiscordDocExample(t *testing.T) {
	// From Discord's developer docs: 175928847299117063 was created
	// 2016-04-30T11:18:25.796Z, worker 1, process 0, increment 7.
	id := mustParse(t, "175928847299117063", "discord")
	if ms := id.UnixMilli(); ms != 1462015105796 {
		t.Fatalf("unix ms = %d, want 1462015105796", ms)
	}
	if id.Datacenter != 1 || id.Worker != 0 || id.Sequence != 7 {
		t.Fatalf("fields = %d/%d/%d, want 1/0/7", id.Datacenter, id.Worker, id.Sequence)
	}
}

func TestCustomNumericEpoch(t *testing.T) {
	// An epoch given as a literal Unix-ms offset must behave like a name.
	id := mustParse(t, "4194304", "1000") // ts offset = 1 ms
	if id.UnixMilli() != 1001 {
		t.Fatalf("unix ms = %d, want 1001", id.UnixMilli())
	}
	if id.EpochName != "1000" {
		t.Fatalf("epoch name = %q, want the literal offset", id.EpochName)
	}
}

func TestResolveEpochRejectsUnknownNames(t *testing.T) {
	for _, bad := range []string{"tw", "-5", "12ab", ""} {
		if _, _, err := ResolveEpoch(bad); err == nil {
			t.Errorf("ResolveEpoch(%q) should fail", bad)
		}
	}
}

func TestEpochNamesIsSortedAndComplete(t *testing.T) {
	names := EpochNames()
	if names != "discord|instagram|twitter|unix" {
		t.Fatalf("EpochNames() = %q", names)
	}
}

func TestParseRejectsInvalidInput(t *testing.T) {
	bad := []string{
		"", "12x4", "+123", " 123", "-9", // non-numeric shapes
		"0",                   // zero is not a valid snowflake
		"9223372036854775808", // 2^63 overflows int64
	}
	for _, s := range bad {
		if _, err := Parse(s, "twitter"); err == nil {
			t.Errorf("Parse(%q) should fail", s)
		}
	}
	if _, err := Parse("9223372036854775807", "twitter"); err != nil {
		t.Errorf("int64 max should parse: %v", err)
	}
}

func TestInterpretationsCoverAllEpochsSorted(t *testing.T) {
	ins := mustParse(t, "1541815603606036480", "twitter").Interpretations()
	if len(ins) != 4 {
		t.Fatalf("got %d interpretations, want 4", len(ins))
	}
	order := []string{"discord", "instagram", "twitter", "unix"}
	for i, in := range ins {
		if in.Epoch != order[i] {
			t.Fatalf("interpretation %d = %q, want %q", i, in.Epoch, order[i])
		}
	}
}

func TestInterpretationPlausibility(t *testing.T) {
	// For this 2022 tweet ID, the unix-epoch reading lands in 1981 —
	// exactly the trap the plausibility flag exists to catch.
	ins := mustParse(t, "1541815603606036480", "twitter").Interpretations()
	verdicts := map[string]bool{}
	for _, in := range ins {
		verdicts[in.Epoch] = in.Plausible
	}
	if !verdicts["twitter"] || verdicts["unix"] {
		t.Fatalf("verdicts = %v: twitter must be plausible, unix must not", verdicts)
	}
}

func TestLooksLikeShape(t *testing.T) {
	cases := map[string]bool{
		"1":                    true,
		"1541815603606036480":  true,
		"9223372036854775807":  true,  // int64 max
		"9223372036854775808":  false, // int64 overflow
		"12345678901234567890": false, // 20 digits
		"0":                    false,
		"12a":                  false,
		"":                     false,
	}
	for s, want := range cases {
		if got := LooksLike(s); got != want {
			t.Errorf("LooksLike(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestTimestampOffsetIsRaw41BitField(t *testing.T) {
	id := mustParse(t, "1541815603606036480", "unix")
	if off := id.TimestampOffsetMS(); off != 1541815603606036480>>22 {
		t.Fatalf("offset = %d, want value >> 22", off)
	}
	if id.UnixMilli() != id.TimestampOffsetMS() {
		t.Fatal("unix epoch: offset and unix ms must be identical")
	}
}
