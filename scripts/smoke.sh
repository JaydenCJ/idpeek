#!/usr/bin/env bash
# End-to-end smoke test for idpeek: builds the binary and asserts on real
# CLI output for every subcommand and every ID format, using published
# reference vectors (RFC 9562, the ULID spec, the KSUID docs, Discord's
# API docs). No network, idempotent, finishes in seconds.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

fail() {
  echo "SMOKE FAIL: $*" >&2
  exit 1
}

BIN="$WORKDIR/idpeek"

echo "1. build"
(cd "$ROOT" && go build -o "$BIN" ./cmd/idpeek) || fail "go build failed"

echo "2. version matches manifest"
"$BIN" --version | grep -qx "idpeek 0.1.0" || fail "--version mismatch"

echo "3. decode a UUIDv7 (RFC 9562 vector)"
OUT="$("$BIN" 017F22E2-79B0-7CC3-98C4-DC0C0C07398F)"
echo "$OUT" | grep -q "kind         uuid" || fail "uuid kind missing"
echo "$OUT" | grep -q "Unix-epoch time-based" || fail "v7 version name missing"
echo "$OUT" | grep -q "2022-02-22T19:22:22.000Z" || fail "v7 timestamp wrong"
echo "$OUT" | grep -q "01FWHE4YDGFK1SHH6W1G60EECF" || fail "ULID equivalent missing"

echo "4. decode a ULID (spec example)"
OUT="$("$BIN" 01ARZ3NDEKTSV4RRFFQ69G5FAV)"
echo "$OUT" | grep -q "2016-07-30T23:54:10.259Z" || fail "ULID timestamp wrong"

echo "5. decode a KSUID (reference docs example)"
OUT="$("$BIN" 0ujtsYcgvSTl8PAuAdqWYSMnLOv)"
echo "$OUT" | grep -q "2017-10-10T04:00:47.000Z" || fail "KSUID timestamp wrong"

echo "6. decode a Snowflake with epoch interpretations"
OUT="$("$BIN" 1541815603606036480)"
echo "$OUT" | grep -q "2022-06-28T16:07:40.105Z" || fail "twitter-epoch time wrong"
echo "$OUT" | grep -q "time@unix.*implausible" || fail "implausible flag missing"

echo "7. time subcommand honors --epoch (Discord docs vector)"
T="$("$BIN" time --epoch discord 175928847299117063)"
[ "$T" = "2016-04-30T11:18:25.796Z" ] || fail "discord time = $T"

echo "8. convert roundtrips UUID <-> ULID"
U="$("$BIN" convert --to ulid 017f22e2-79b0-7cc3-98c4-dc0c0c07398f)"
BACK="$("$BIN" convert --to uuid "$U")"
[ "$BACK" = "017f22e2-79b0-7cc3-98c4-dc0c0c07398f" ] || fail "roundtrip = $BACK"

echo "9. convert UUIDv1 -> UUIDv6 matches the RFC vector"
V6="$("$BIN" convert --to uuidv6 c232ab00-9414-11ec-b3c8-9f6bdeced846)"
[ "$V6" = "1ec9414c-232a-6b00-b3c8-9f6bdeced846" ] || fail "v1->v6 = $V6"

echo "10. JSON output is one valid object per line"
printf '01ARZ3NDEKTSV4RRFFQ69G5FAV\n0ujtsYcgvSTl8PAuAdqWYSMnLOv\n' \
  | "$BIN" decode --format json - > "$WORKDIR/out.jsonl"
[ "$(wc -l < "$WORKDIR/out.jsonl")" = "2" ] || fail "want 2 JSONL lines"
grep -q '"kind":"ulid"' "$WORKDIR/out.jsonl" || fail "ulid JSON missing"
grep -q '"unix_ms":1507608047000' "$WORKDIR/out.jsonl" || fail "ksuid unix_ms missing"

echo "11. exit codes: 1 for a bad ID, 2 for a bad flag"
set +e
"$BIN" decode not-an-id >/dev/null 2>&1
[ $? -eq 1 ] || fail "bad ID should exit 1"
"$BIN" decode --format yaml 1 >/dev/null 2>&1
[ $? -eq 2 ] || fail "bad --format should exit 2"
set -e

echo "12. timestamp-less IDs are refused by time"
if "$BIN" time 919108f7-52d1-4320-9bac-f847db4148a8 >/dev/null 2>&1; then
  fail "v4 UUID has no timestamp; time should exit 1"
fi

echo "SMOKE OK"
