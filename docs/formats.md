# ID format reference

Exact bit layouts idpeek decodes, with the sources each claim is pinned
to in the test suite. All multi-byte fields are big-endian; all times are
reported in UTC.

## UUID (RFC 9562)

128 bits, written as `xxxxxxxx-xxxx-Vxxx-Nxxx-xxxxxxxxxxxx` where `V` is
the version nibble and the top bits of `N` are the variant. idpeek also
accepts the 32-hex compact form, the `urn:uuid:` prefix, and `{braces}`.

| Version | Layout (bits, MSB first) | Timestamp |
|---|---|---|
| 1 | time_low 32 · time_mid 16 · ver 4 · time_high 12 · var 2 · clock_seq 14 · node 48 | 60-bit count of 100 ns since 1582-10-15 |
| 2 | local_id 32 replaces time_low; domain replaces clock_seq_low | truncated to ~7 min; not reported as a time |
| 3 / 5 | MD5 / SHA-1 digest of namespace + name | none |
| 4 | 122 random bits | none |
| 6 | time_high 32 · time_mid 16 · ver 4 · time_low 12 · var 2 · clock_seq 14 · node 48 | same clock as v1, field order sortable |
| 7 | unix_ts_ms 48 · ver 4 · rand_a 12 · var 2 · rand_b 62 | Unix milliseconds |
| 8 | vendor-defined | not standardized |

The node field's least-significant bit of the first octet is the
multicast/local bit: RFC 9562 requires it to be set for randomly
generated node IDs, so idpeek uses it to say "random node, not a MAC".

Special values: Nil (all zero) and Max (all one) are reported as such and
carry no timestamp.

## ULID (ulid/spec)

128 bits: `unix_ts_ms 48 · randomness 80`, encoded as 26 characters of
Crockford base32 (`0-9A-Z` minus `I L O U`). Decoding is lenient the way
Crockford intended — lowercase plus `i/l → 1`, `o → 0` — and the output
is always re-canonicalized to uppercase. The first character must be
`0-7`; anything above `7ZZZZZZZZZZZZZZZZZZZZZZZZZ` overflows 128 bits and
is rejected.

Because a ULID is exactly 128 bits, `idpeek convert` maps it to and from
UUID form bit-for-bit (a ULID minted now happens to read as an unusual
UUID version; idpeek says so rather than pretending otherwise).

## KSUID (Segment)

160 bits: `timestamp 32 · payload 128`, where the timestamp counts
seconds since the KSUID epoch `1400000000` (2014-05-13T16:53:20Z),
encoded as exactly 27 base62 characters (`0-9A-Za-z`, case-sensitive).
The maximum value `aWgEPTl1tmebfsQzFP4bxwgy80V` is 2^160-1; larger
strings of valid shape are rejected as overflow.

## Snowflake (Twitter layout)

63 bits in a positive int64: `timestamp 41 · datacenter 5 · worker 5 ·
sequence 12`, where the timestamp counts milliseconds since a
service-chosen epoch. idpeek ships four epochs and accepts any literal
Unix-millisecond offset via `--epoch`:

| Epoch | Offset (Unix ms) | Used by |
|---|---|---|
| `twitter` (default) | 1288834974657 | Twitter/X tweet & user IDs |
| `discord` | 1420070400000 | Discord snowflakes |
| `instagram` | 1314220021721 | Instagram media IDs |
| `unix` | 0 | Sonyflake-style dialects counting from 1970 |

Because the epoch is not recoverable from the ID itself, idpeek always
prints an interpretation table (`time@twitter`, `time@discord`, …) and
marks each reading `plausible` or `implausible` against a fixed
2007-2040 window — deterministically, without consulting the wall clock.
Some dialects read the middle 10 bits as one machine ID; idpeek reports
both the 5/5 split and the combined `machine_id`.

## Detection

The four shapes are mutually exclusive, so auto-detection never guesses:

| Shape | Kind |
|---|---|
| 36 chars with dashes at 8/13/18/23, 32 hex, `urn:uuid:…`, `{…}` | UUID |
| 26 Crockford-base32 chars, first ≤ `7` | ULID |
| 27 base62 chars | KSUID |
| 1-19 decimal digits fitting a positive int64 | Snowflake |

Edge case: a 26-digit numeric string is a valid ULID but overflows int64,
so it decodes as a ULID — the only consistent reading.
