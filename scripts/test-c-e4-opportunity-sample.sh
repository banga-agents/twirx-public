#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: test-c-e4-opportunity-sample.sh VERIFIER SAMPLE_ROOT" >&2
  exit 2
fi

verifier=$1
sample_root=$2
manifest=$sample_root/sample-manifest.json

if [[ ! -x $verifier || ! -f $manifest ]]; then
  echo "restricted-C verifier or sample manifest unavailable" >&2
  exit 1
fi

python3 - "$sample_root" <<'PY'
import hashlib
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1]).resolve()
manifest = json.loads((root / "sample-manifest.json").read_text())
assert manifest["format"] == "tw.e4-opportunity-c-verifier-sample/0.1"
assert manifest["packet_samples"] == 63
assert manifest["mapping_samples"] == 63
assert manifest["frame_samples"] == 6
assert len(manifest["entries"]) == 132
for entry in manifest["entries"]:
    path = (root / entry["path"]).resolve()
    assert root in path.parents
    data = path.read_bytes()
    assert "sha256:" + hashlib.sha256(data).hexdigest() == entry["digest"]
PY

packets=0
mappings=0
frames=0
while IFS= read -r artifact; do
  name=${artifact##*/}
  case $name in
    packet-*) kind=packet; packets=$((packets + 1)) ;;
    mapping-claim-*) kind=mapping-claim; mappings=$((mappings + 1)) ;;
    frame-*) kind=frame; frames=$((frames + 1)) ;;
    *) echo "unknown sample artifact: $artifact" >&2; exit 1 ;;
  esac
  "$verifier" "$kind" "$artifact" >/dev/null
done < <(find "$sample_root/artifacts" -type f -name '*.cbor' -print | LC_ALL=C sort)

if [[ $packets -ne 63 || $mappings -ne 63 || $frames -ne 6 ]]; then
  echo "unexpected Opportunity C sample counts: packets=$packets mappings=$mappings frames=$frames" >&2
  exit 1
fi

echo "E4.5 restricted-C sample verification passed: packets=$packets mappings=$mappings frames=$frames total=$((packets + mappings + frames))"
