#!/usr/bin/env bash
set -euo pipefail

FUZZER="${1:?usage: test-c-fuzz.sh FUZZER}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$ROOT/var/test-c-fuzz"
rm -rf "$WORK"
mkdir -p "$WORK/corpus" "$WORK/artifacts"

python3 - "$ROOT" "$WORK/corpus" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
corpus_dir = pathlib.Path(sys.argv[2])
manifest = json.loads((root / "conformance/observation/vectors.json").read_text(encoding="utf-8"))
for vector in manifest["vectors"]:
    (corpus_dir / f"{vector['id']}.cbor").write_bytes(bytes.fromhex(vector["cbor_hex"]))
PY

if ! "$FUZZER" \
  -runs=5000 \
  -max_len=65536 \
  -artifact_prefix="$WORK/artifacts/" \
  "$WORK/corpus" >"$WORK/fuzzer.log" 2>&1; then
  cat "$WORK/fuzzer.log" >&2
  exit 1
fi

echo "c-fuzz: 5000 libFuzzer runs completed without a crash"
