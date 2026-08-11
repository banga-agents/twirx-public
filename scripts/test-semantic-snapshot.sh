#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/twirx-snapshot-test.XXXXXX")"
case "$WORK" in
  "${TMPDIR:-/tmp}"/twirx-snapshot-test.*) ;;
  *) echo "refusing unsafe temporary path" >&2; exit 1 ;;
esac
trap 'rm -rf -- "$WORK"' EXIT

SNAPSHOT="$WORK/snapshot"
PACKETS="$WORK/packets"

"$ROOT/bin/twirx-snapshot" build \
  --root "$ROOT" \
  --out "$SNAPSHOT" \
  --source-revision test-suite \
  --created-at 2026-08-11T00:00:00Z >"$WORK/build.json"

"$ROOT/bin/twirx-snapshot" build \
  --root "$ROOT" \
  --out "$WORK/snapshot-second" \
  --source-revision test-suite \
  --created-at 2026-08-11T00:00:00Z >"$WORK/build-second.json"

"$ROOT/bin/twirx-snapshot" verify --snapshot "$SNAPSHOT" >"$WORK/verify.json"
"$ROOT/bin/tw-verify-data-plane-c" snapshot "$SNAPSHOT/manifest.cbor"

python3 - "$SNAPSHOT/packets/segment-000001.json" "$PACKETS" <<'PY'
import base64
import json
import pathlib
import sys

segment = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
destination = pathlib.Path(sys.argv[2])
destination.mkdir(mode=0o750)
assert segment["format"] == "tw.snapshot-packet-segment/0.1"
assert segment["start_sequence"] == 1
assert len(segment["entries"]) == 18
for entry in segment["entries"]:
    digest = entry["digest"].removeprefix("sha256:")
    assert len(digest) == 64
    (destination / f"{digest}.cbor").write_bytes(base64.b64decode(entry["cbor"], validate=True))
PY

count=0
for packet in "$PACKETS"/*.cbor; do
  "$ROOT/bin/tw-verify-data-plane-c" packet "$packet"
  count=$((count + 1))
done
if [[ $count -ne 18 ]]; then
  echo "unexpected compiled packet count: $count" >&2
  exit 1
fi

"$ROOT/bin/twirx-snapshot" query \
  --snapshot "$SNAPSHOT" \
  --file "$ROOT/examples/semantic-query-population.json" >"$WORK/population.json"
"$ROOT/bin/twirx-snapshot" query \
  --snapshot "$SNAPSHOT" \
  --file "$ROOT/examples/semantic-query-two-origins.json" >"$WORK/two-origins.json"

mkdir "$WORK/query-cbor"
python3 - "$WORK/population.json" "$WORK/two-origins.json" "$WORK/query-cbor" <<'PY'
import base64
import json
import pathlib
import sys

destination = pathlib.Path(sys.argv[3])
for name, source in zip(("population", "two-origins"), sys.argv[1:3]):
    document = json.loads(pathlib.Path(source).read_text(encoding="utf-8"))
    (destination / f"{name}.query.cbor").write_bytes(base64.b64decode(document["canonical_query_cbor"], validate=True))
    (destination / f"{name}.result.cbor").write_bytes(base64.b64decode(document["canonical_result_cbor"], validate=True))
PY

for query in "$WORK/query-cbor"/*.query.cbor; do
  "$ROOT/bin/tw-verify-data-plane-c" query "$query"
done
for result in "$WORK/query-cbor"/*.result.cbor; do
  "$ROOT/bin/tw-verify-data-plane-c" query-result "$result"
done

python3 - "$WORK/build.json" "$WORK/build-second.json" "$WORK/population.json" "$WORK/two-origins.json" "$ROOT/conformance/e3-snapshot/expected-baseline.json" <<'PY'
import json
import pathlib
import sys

build, second, population, combined, baseline = [json.loads(pathlib.Path(path).read_text(encoding="utf-8")) for path in sys.argv[1:]]
assert baseline["format"] == "tw.snapshot-demo-baseline/0.1"
assert build["snapshot_id"] == second["snapshot_id"]
assert build["actual"] == baseline["actual"]
assert population["status"] == "resolved" and len(population["rows"]) == baseline["population_query_rows"]
assert combined["status"] == "resolved"
assert sorted({row["origin_id"] for row in combined["rows"]}) == baseline["two_origin_query_origins"]
assert population["plan"]["network_requests"] == baseline["network_requests"]
assert combined["plan"]["network_requests"] == baseline["network_requests"]
PY

echo "Semantic Snapshot integration passed: packets=18 public_origins=2 fixtures_excluded=true network_requests=0"
