#!/usr/bin/env bash
# Decode one reference ID of every supported format, from ids.txt.
# Offline and deterministic: every ID is a published spec/docs vector.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IDPEEK="${IDPEEK:-idpeek}"
command -v "$IDPEEK" >/dev/null 2>&1 \
  || { echo "error: '$IDPEEK' not found; build it (go build -o /tmp/idpeek ./cmd/idpeek) and set IDPEEK" >&2; exit 1; }

"$IDPEEK" decode - < "$HERE/ids.txt"
