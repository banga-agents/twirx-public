#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: test-c-e2-fuzz.sh FUZZER LAB_CLI" >&2
  exit 2
fi

fuzzer=$1
lab_cli=$2
work=var/test-c-e2-fuzz
rm -rf -- "$work"
mkdir -p "$work/results" "$work/corpus"
"$lab_cli" invoke --root . --results "$work/results" --origin controlled-origin-lab --operation fixture.getOffer --mode replay --input product_id=demo-1 >/dev/null
bundles=("$work/results"/*)
cp -- "${bundles[0]}/result.cbor" "$work/corpus/result"
cp -- "${bundles[0]}/manifest.cbor" "$work/corpus/manifest"
cp -- "${bundles[0]}/semantic-closure.cbor" "$work/corpus/closure"
"$fuzzer" -runs=5000 "$work/corpus"
