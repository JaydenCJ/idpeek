# idpeek examples

Two runnable scripts plus a sample ID list, all offline and deterministic
— every ID in `ids.txt` is a published reference vector (RFC 9562, the
ULID spec, the KSUID docs, Discord's API docs, a public tweet ID).

Both scripts run the `idpeek` binary on your `PATH`; point `IDPEEK` at a locally
built binary to use that instead:

```bash
go build -o /tmp/idpeek ./cmd/idpeek
IDPEEK=/tmp/idpeek bash examples/decode-samples.sh
```

## decode-samples.sh

Pipes `ids.txt` — one ID of every supported shape — through
`idpeek decode -`, showing the full inspection block for each: kind,
version, embedded timestamp, machine bits, and cross-format equivalents.

```bash
bash examples/decode-samples.sh
```

## timeline.sh

Answers "which of these records came first?" for a mixed bag of IDs: it
runs `idpeek time` on each line, skips IDs that carry no timestamp (like
UUIDv4) with a note on stderr, and prints the rest oldest-first.

```bash
bash examples/timeline.sh                 # uses ids.txt
bash examples/timeline.sh /path/to/yours  # one ID per line
```

Real captured output:

```text
skipped (no embedded timestamp): 919108f7-52d1-4320-9bac-f847db4148a8
2012-03-03T13:01:20.453Z  175928847299117063
2016-07-30T23:54:10.259Z  01ARZ3NDEKTSV4RRFFQ69G5FAV
2017-10-10T04:00:47.000Z  0ujtsYcgvSTl8PAuAdqWYSMnLOv
2022-02-22T19:22:22.0000000Z  1ec9414c-232a-6b00-b3c8-9f6bdeced846
2022-02-22T19:22:22.0000000Z  c232ab00-9414-11ec-b3c8-9f6bdeced846
2022-02-22T19:22:22.000Z  017f22e2-79b0-7cc3-98c4-dc0c0c07398f
2022-06-28T16:07:40.105Z  1541815603606036480
```

Note the first data line: that Discord ID reads as 2012 under the default
twitter epoch (its true creation time is 2016-04-30T11:18:25.796Z) — pass
`--epoch discord` to `idpeek time` when you know the source service.
