# Contributing to idpeek

Issues, discussions and pull requests are all welcome.

## Getting started

You need Go ≥1.22; nothing else.

```bash
git clone https://github.com/JaydenCJ/idpeek && cd idpeek
go build ./...
go test ./...
bash scripts/smoke.sh
```

`scripts/smoke.sh` builds the binary and asserts on real CLI output for
every subcommand and every ID format, using published reference vectors
(RFC 9562, the ULID spec, the KSUID docs, Discord's API docs); it must
finish by printing `SMOKE OK`.

## Before you open a pull request

1. `gofmt -l .` reports nothing (formatting is enforced).
2. `go vet ./...` passes with no findings.
3. `go test ./...` passes (90 deterministic tests, no network).
4. `bash scripts/smoke.sh` prints `SMOKE OK`.
5. Add tests for behavior changes; keep logic in pure, unit-testable
   modules (the format packages never touch I/O — only `cli` reads
   streams).

## Ground rules

- Keep dependencies at zero: idpeek is standard library only, and staying
  that way is a feature. Adding one needs strong justification in the PR.
- No network calls, ever. No telemetry. Decoding IDs must work on an
  air-gapped machine.
- Determinism first: identical input must produce byte-identical output,
  including field order; nothing may read the wall clock.
- New format claims need receipts: cite the spec or official docs, and
  add a test pinning a published reference vector.
- Code comments and doc comments are written in English.

## Reporting bugs

Include the output of `idpeek version`, the exact ID (or a redacted one
with the same shape), the full command you ran, and — for misdecodes —
what the correct decoding should be and which spec or generator says so.

## Security

Please do not open public issues for security problems; use GitHub's
private vulnerability reporting on this repository instead.
