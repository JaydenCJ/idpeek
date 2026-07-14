// In-process CLI tests: Run() takes args and streams, so the full command
// surface — subcommands, flags, stdin, exit codes — is exercised without
// spawning a binary or touching the network.
package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JaydenCJ/idpeek/internal/version"
)

// run invokes the CLI and captures stdout, stderr, and the exit code.
func run(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut strings.Builder
	code := Run(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestVersionSubcommandAndFlag(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-V"} {
		out, _, code := run(t, "", arg)
		if code != ExitOK || out != "idpeek "+version.Version+"\n" {
			t.Errorf("%s: out=%q code=%d", arg, out, code)
		}
	}
}

func TestHelpAndNoArgs(t *testing.T) {
	// Asking for help — top-level or per subcommand — is not an error:
	// usage goes to stdout and the exit code is 0.
	for _, args := range [][]string{{"--help"}, {"-h"}, {"decode", "--help"}, {"time", "-h"}, {"convert", "--help"}} {
		out, _, code := run(t, "", args...)
		if code != ExitOK || !strings.Contains(out, "Usage:") {
			t.Errorf("%v: code=%d out=%q", args, code, out)
		}
	}
	_, errOut, code := run(t, "")
	if code != ExitUsage || !strings.Contains(errOut, "Usage:") {
		t.Fatalf("no args: code=%d stderr=%q", code, errOut)
	}
}

func TestBareIDDefaultsToDecode(t *testing.T) {
	out, _, code := run(t, "", "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"kind", "ulid", "2016-07-30T23:54:10.259Z"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDecodeMultipleIDsSeparatedByBlankLine(t *testing.T) {
	out, _, code := run(t, "", "decode",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "175928847299117063")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	blocks := strings.Split(strings.TrimRight(out, "\n"), "\n\n")
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2:\n%s", len(blocks), out)
	}
}

func TestDecodeJSONIsOneObjectPerLine(t *testing.T) {
	out, _, code := run(t, "", "decode", "--format", "json",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "0ujtsYcgvSTl8PAuAdqWYSMnLOv")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d JSON lines, want 2", len(lines))
	}
	kinds := []string{}
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line is not valid JSON: %v\n%s", err, line)
		}
		kinds = append(kinds, m["kind"].(string))
	}
	if kinds[0] != "ulid" || kinds[1] != "ksuid" {
		t.Fatalf("kinds = %v", kinds)
	}
}

func TestStdinDashReadsOneIDPerLine(t *testing.T) {
	// Blank lines and surrounding whitespace must be tolerated.
	stdin := "01ARZ3NDEKTSV4RRFFQ69G5FAV\n\n  175928847299117063  \n"
	out, _, code := run(t, stdin, "time", "-")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), out)
	}
	if lines[0] != "2016-07-30T23:54:10.259Z" {
		t.Fatalf("first line = %q", lines[0])
	}
}

func TestTimeSubcommandRFC3339AndUnixMS(t *testing.T) {
	out, _, code := run(t, "", "time", "017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
	if code != ExitOK || out != "2022-02-22T19:22:22.000Z\n" {
		t.Fatalf("time: code=%d out=%q", code, out)
	}
	out, _, code = run(t, "", "time", "--unix-ms", "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if code != ExitOK || out != "1469922850259\n" {
		t.Fatalf("time --unix-ms: code=%d out=%q", code, out)
	}
}

func TestTimeFailsForTimestamplessID(t *testing.T) {
	out, errOut, code := run(t, "", "time", "919108f7-52d1-4320-9bac-f847db4148a8")
	if code != ExitError || out != "" {
		t.Fatalf("v4 uuid: code=%d out=%q", code, out)
	}
	if !strings.Contains(errOut, "no embedded timestamp") {
		t.Fatalf("stderr = %q", errOut)
	}
}

func TestEpochFlagChangesSnowflakeTime(t *testing.T) {
	out, _, code := run(t, "", "time", "--epoch", "discord", "175928847299117063")
	if code != ExitOK || out != "2016-04-30T11:18:25.796Z\n" {
		t.Fatalf("discord epoch: code=%d out=%q", code, out)
	}
}

func TestKindFlagForcesFormat(t *testing.T) {
	// Forcing a mismatched kind is a decode failure (exit 1), proving the
	// flag actually bypasses auto-detection.
	_, errOut, code := run(t, "", "decode", "--kind", "uuid", "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if code != ExitError || !strings.Contains(errOut, "not a UUID") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestBadFlagValuesAreUsageErrors(t *testing.T) {
	// Bad flag values must exit 2 (usage), never 1 (decode failure), so
	// scripts can tell "you called it wrong" from "the ID was bad".
	cases := [][]string{
		{"decode", "--kind", "guid", "1"},
		{"decode", "--format", "yaml", "1"},
		{"time", "--epoch", "myspace", "175928847299117063"},
		{"decode", "--bogus", "1"},
	}
	for _, args := range cases {
		if _, _, code := run(t, "", args...); code != ExitUsage {
			t.Errorf("%v: code=%d, want 2", args, code)
		}
	}
}

func TestConvertUUIDToULID(t *testing.T) {
	out, _, code := run(t, "", "convert", "--to", "ulid",
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
	if code != ExitOK || out != "01FWHE4YDGFK1SHH6W1G60EECF\n" {
		t.Fatalf("convert: code=%d out=%q", code, out)
	}
}

func TestConvertV1ToV6ViaStdin(t *testing.T) {
	out, _, code := run(t, "c232ab00-9414-11ec-b3c8-9f6bdeced846\n",
		"convert", "--to", "uuidv6", "-")
	if code != ExitOK || out != "1ec9414c-232a-6b00-b3c8-9f6bdeced846\n" {
		t.Fatalf("convert stdin: code=%d out=%q", code, out)
	}
}

func TestConvertWithoutTargetIsUsageError(t *testing.T) {
	_, errOut, code := run(t, "", "convert", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
	if code != ExitUsage || !strings.Contains(errOut, "--to") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestBatchContinuesPastBadInputWithExit1(t *testing.T) {
	// One bad ID among good ones: decode everything it can, report the
	// failure on stderr, and exit 1 — the useful behavior for log triage.
	out, errOut, code := run(t, "", "time",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "not-an-id", "175928847299117063")
	if code != ExitError {
		t.Fatalf("code = %d, want 1", code)
	}
	if lines := strings.Count(out, "\n"); lines != 2 {
		t.Fatalf("good IDs must still print (%d lines):\n%s", lines, out)
	}
	if !strings.Contains(errOut, "unrecognized ID") {
		t.Fatalf("stderr = %q", errOut)
	}
}
