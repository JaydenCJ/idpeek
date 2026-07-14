#!/usr/bin/env bash
# Order mixed-format IDs by their embedded creation time — the "which of
# these records came first?" question that comes up in incident triage.
# IDs with no embedded timestamp (e.g. UUIDv4) are reported on stderr and
# skipped; the rest are printed oldest-first as "<time>  <id>".
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IDPEEK="${IDPEEK:-idpeek}"
IDS="${1:-$HERE/ids.txt}"
command -v "$IDPEEK" >/dev/null 2>&1 \
  || { echo "error: '$IDPEEK' not found; build it (go build -o /tmp/idpeek ./cmd/idpeek) and set IDPEEK" >&2; exit 1; }

while IFS= read -r id; do
  [ -n "$id" ] || continue
  if t="$("$IDPEEK" time "$id" 2>/dev/null)"; then
    printf '%s  %s\n' "$t" "$id"
  else
    echo "skipped (no embedded timestamp): $id" >&2
  fi
done < "$IDS" | sort
