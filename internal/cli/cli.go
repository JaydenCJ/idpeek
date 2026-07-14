// Package cli wires the idpeek subcommands (decode, time, convert,
// version) to the pure decoding packages. Run is fully in-process — args
// in, exit code out — so the whole CLI is unit-testable without spawning
// a binary.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/JaydenCJ/idpeek/internal/convert"
	"github.com/JaydenCJ/idpeek/internal/detect"
	"github.com/JaydenCJ/idpeek/internal/render"
	"github.com/JaydenCJ/idpeek/internal/snowflake"
	"github.com/JaydenCJ/idpeek/internal/version"
)

// Exit codes.
const (
	ExitOK    = 0
	ExitError = 1 // at least one input failed to decode/convert
	ExitUsage = 2 // bad flags or arguments
)

const usageText = `idpeek — decode UUIDs, ULIDs, KSUIDs, and Snowflakes

Usage:
  idpeek [decode] [flags] <id>... | -
  idpeek time     [flags] <id>... | -
  idpeek convert  --to <target> [flags] <id>... | -
  idpeek version

Subcommands:
  decode    full inspection: kind, version, timestamp, fields (default)
  time      print only the embedded creation time, one line per ID
  convert   re-express an ID: --to uuid|ulid|hex|uuidv6|uuidv1
  version   print the idpeek version

Flags (decode/time/convert):
  --kind uuid|ulid|ksuid|snowflake   force the format, skip auto-detection
  --epoch <name|ms>                  Snowflake epoch (default twitter)
  --format text|json                 decode output style (json = one object/line)
  --unix-ms                          time: print Unix milliseconds instead
  --to uuid|ulid|hex|uuidv6|uuidv1   convert: target representation (required)

Pass - (or pipe with no <id> args) to read IDs from stdin, one per line.
Exit codes: 0 ok, 1 decode/convert failure, 2 usage error.
`

// Run executes idpeek with the given arguments and streams. It returns the
// process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return ExitUsage
	}
	switch args[0] {
	case "version", "--version", "-V":
		fmt.Fprintf(stdout, "idpeek %s\n", version.Version)
		return ExitOK
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usageText)
		return ExitOK
	case "decode", "time", "convert":
		// explicit subcommand
	default:
		// Default subcommand: `idpeek <id>` == `idpeek decode <id>`.
		args = append([]string{"decode"}, args...)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "decode":
		return runDecode(rest, stdin, stdout, stderr)
	case "time":
		return runTime(rest, stdin, stdout, stderr)
	case "convert":
		return runConvert(rest, stdin, stdout, stderr)
	}
	fmt.Fprint(stderr, usageText)
	return ExitUsage
}

// commonFlags holds the flags shared by decode/time/convert.
type commonFlags struct {
	kind  string
	epoch string
}

func newFlagSet(name string, stderr io.Writer, cf *commonFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {} // printing is handled by parseFlags
	fs.StringVar(&cf.kind, "kind", "", "force format: uuid|ulid|ksuid|snowflake")
	fs.StringVar(&cf.epoch, "epoch", snowflake.DefaultEpoch,
		"Snowflake epoch: "+snowflake.EpochNames()+", or Unix-ms offset")
	return fs
}

// parseFlags maps flag parsing onto the CLI's exit-code semantics: an
// explicit -h/--help prints the full usage to stdout and exits 0 (asking
// for help is not a usage error), while any real parse error — already
// reported by the flag package on stderr — gets a pointer to --help and
// exits 2. ok is true only when parsing succeeded and the command should
// proceed.
func parseFlags(fs *flag.FlagSet, args []string, stdout, stderr io.Writer) (code int, ok bool) {
	err := fs.Parse(args)
	if err == nil {
		return ExitOK, true
	}
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, usageText)
		return ExitOK, false
	}
	fmt.Fprintln(stderr, `idpeek: run "idpeek --help" for usage`)
	return ExitUsage, false
}

// validate rejects bad flag values up front so they exit 2, not 1.
func (cf *commonFlags) validate() error {
	switch cf.kind {
	case "", detect.KindUUID, detect.KindULID, detect.KindKSUID, detect.KindSnowflake:
	default:
		return fmt.Errorf("invalid --kind %q (want uuid|ulid|ksuid|snowflake)", cf.kind)
	}
	if _, _, err := snowflake.ResolveEpoch(cf.epoch); err != nil {
		return fmt.Errorf("invalid --epoch: %v", err)
	}
	return nil
}

func (cf *commonFlags) decode(input string) (render.Decoded, error) {
	if cf.kind != "" {
		return detect.DecodeAs(input, cf.kind, cf.epoch)
	}
	return detect.Decode(input, cf.epoch)
}

// inputs resolves the ID list: positional args, with "-" (or an empty
// list) meaning one ID per non-blank stdin line.
func inputs(args []string, stdin io.Reader) ([]string, error) {
	fromStdin := len(args) == 0
	var out []string
	for _, a := range args {
		if a == "-" {
			fromStdin = true
			continue
		}
		out = append(out, a)
	}
	if fromStdin {
		sc := bufio.NewScanner(stdin)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line != "" {
				out = append(out, line)
			}
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("reading stdin: %v", err)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no IDs given (pass IDs as arguments, or pipe them and use -)")
	}
	return out, nil
}

func runDecode(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cf commonFlags
	var format string
	fs := newFlagSet("decode", stderr, &cf)
	fs.StringVar(&format, "format", "text", "output format: text|json")
	if code, ok := parseFlags(fs, args, stdout, stderr); !ok {
		return code
	}
	if format != "text" && format != "json" {
		fmt.Fprintf(stderr, "idpeek: invalid --format %q (want text|json)\n", format)
		return ExitUsage
	}
	if err := cf.validate(); err != nil {
		fmt.Fprintf(stderr, "idpeek: %v\n", err)
		return ExitUsage
	}
	ids, err := inputs(fs.Args(), stdin)
	if err != nil {
		fmt.Fprintf(stderr, "idpeek: %v\n", err)
		return ExitUsage
	}
	code := ExitOK
	for i, id := range ids {
		d, err := cf.decode(id)
		if err != nil {
			fmt.Fprintf(stderr, "idpeek: %v\n", err)
			code = ExitError
			continue
		}
		if format == "json" {
			if err := render.JSON(stdout, d); err != nil {
				fmt.Fprintf(stderr, "idpeek: %v\n", err)
				return ExitError
			}
			continue
		}
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		render.Text(stdout, d)
	}
	return code
}

func runTime(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cf commonFlags
	var unixMS bool
	fs := newFlagSet("time", stderr, &cf)
	fs.BoolVar(&unixMS, "unix-ms", false, "print Unix milliseconds instead of RFC 3339")
	if code, ok := parseFlags(fs, args, stdout, stderr); !ok {
		return code
	}
	if err := cf.validate(); err != nil {
		fmt.Fprintf(stderr, "idpeek: %v\n", err)
		return ExitUsage
	}
	ids, err := inputs(fs.Args(), stdin)
	if err != nil {
		fmt.Fprintf(stderr, "idpeek: %v\n", err)
		return ExitUsage
	}
	code := ExitOK
	for _, id := range ids {
		d, err := cf.decode(id)
		if err != nil {
			fmt.Fprintf(stderr, "idpeek: %v\n", err)
			code = ExitError
			continue
		}
		if d.Time == nil {
			fmt.Fprintf(stderr, "idpeek: %q (%s) has no embedded timestamp\n", id, d.Kind)
			code = ExitError
			continue
		}
		if unixMS {
			fmt.Fprintf(stdout, "%d\n", d.Time.UnixMilli)
		} else {
			fmt.Fprintf(stdout, "%s\n", d.Time.RFC3339)
		}
	}
	return code
}

func runConvert(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cf commonFlags
	var target string
	fs := newFlagSet("convert", stderr, &cf)
	fs.StringVar(&target, "to", "", "target: "+strings.Join(convert.Targets, "|"))
	if code, ok := parseFlags(fs, args, stdout, stderr); !ok {
		return code
	}
	if err := cf.validate(); err != nil {
		fmt.Fprintf(stderr, "idpeek: %v\n", err)
		return ExitUsage
	}
	ok := false
	for _, t := range convert.Targets {
		if target == t {
			ok = true
			break
		}
	}
	if !ok {
		fmt.Fprintf(stderr, "idpeek: convert needs --to %s\n", strings.Join(convert.Targets, "|"))
		return ExitUsage
	}
	ids, err := inputs(fs.Args(), stdin)
	if err != nil {
		fmt.Fprintf(stderr, "idpeek: %v\n", err)
		return ExitUsage
	}
	code := ExitOK
	for _, id := range ids {
		out, err := convert.Convert(strings.TrimSpace(id), target, cf.epoch)
		if err != nil {
			fmt.Fprintf(stderr, "idpeek: %v\n", err)
			code = ExitError
			continue
		}
		fmt.Fprintln(stdout, out)
	}
	return code
}
