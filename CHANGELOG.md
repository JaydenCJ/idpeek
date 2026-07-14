# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-07-13

### Added

- UUID decoding per RFC 9562: canonical, compact, `urn:uuid:`, and braced
  input forms; versions 1-8 plus Nil/Max; variant bits; Gregorian
  timestamps for v1/v6 at 100 ns resolution; Unix-millisecond timestamps
  for v7; DCE Security domain/local-ID for v2; clock sequence, node field,
  and multicast-bit ("random, not a MAC") analysis.
- ULID decoding and encoding: Crockford base32 with lowercase and i/l/o
  alias leniency, 48-bit millisecond timestamp, 80-bit randomness,
  overflow rejection above `7ZZZ…`.
- KSUID decoding and encoding: 27-char base62 over 160 bits with exact
  overflow bounds, timestamp against the KSUID epoch, 128-bit payload.
- Snowflake decomposition: 41/5/5/12 bit layout, built-in twitter,
  discord, instagram, and unix epochs plus custom Unix-ms offsets, and a
  per-epoch interpretation table with deterministic plausibility verdicts.
- Format auto-detection over mutually exclusive shapes, with `--kind` to
  force a format explicitly.
- `decode` (default), `time` (`--unix-ms`), `convert`, and `version`
  subcommands; text and JSON Lines output; stdin batch input via `-`;
  distinct exit codes for decode failures (1) vs usage errors (2).
- Lossless conversions: ULID <-> UUID (bit-identical 128 bits),
  UUIDv1 <-> UUIDv6 (RFC-vector verified), and raw hex for every format;
  width-mismatched conversions are refused, never truncated.
- Runnable examples (`examples/decode-samples.sh`, `examples/timeline.sh`)
  and a bit-layout reference (`docs/formats.md`).
- 90 deterministic offline tests (unit + in-process CLI) built on
  published reference vectors, and `scripts/smoke.sh`.

[0.1.0]: https://github.com/JaydenCJ/idpeek/releases/tag/v0.1.0
