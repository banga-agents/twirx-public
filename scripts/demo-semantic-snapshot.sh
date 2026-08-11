#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REVISION="$(git -C "$ROOT" rev-parse HEAD)"
CREATED_AT="${TWIRX_SNAPSHOT_CREATED_AT:-2026-08-11T00:00:00Z}"
DEMO_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/twirx-semantic-snapshot.XXXXXX")"
SNAPSHOT="$DEMO_ROOT/snapshot"

"$ROOT/bin/twirx-snapshot" build \
  --root "$ROOT" \
  --out "$SNAPSHOT" \
  --source-revision "$REVISION" \
  --created-at "$CREATED_AT"

"$ROOT/bin/twirx-snapshot" verify --snapshot "$SNAPSHOT"
"$ROOT/bin/twirx-snapshot" query \
  --snapshot "$SNAPSHOT" \
  --file "$ROOT/examples/semantic-query-population.json"
"$ROOT/bin/twirx-snapshot" query \
  --snapshot "$SNAPSHOT" \
  --file "$ROOT/examples/semantic-query-two-origins.json"

echo "Snapshot retained for inspection: $SNAPSHOT"
echo "No network request was made by the builder or runtime."
