#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: test-c-dataplane-fuzz.sh FUZZER" >&2
  exit 2
fi

fuzzer=$1
work=var/test-c-dataplane-fuzz
rm -rf -- "$work"
mkdir -p "$work/corpus"

while IFS=$'\t' read -r name kind expected reason hex; do
  if [[ $name == name || $expected != accept ]]; then
    continue
  fi
  printf '%s' "$hex" | xxd -r -p >"$work/corpus/$kind-$name"
done < conformance/e3-s1/vectors.tsv

while IFS=$'\t' read -r name kind expected reason hex; do
  if [[ $name == name || $expected != accept ]]; then
    continue
  fi
  printf '%s' "$hex" | xxd -r -p >"$work/corpus/$kind-$name"
done < conformance/e4-ontology/vectors.tsv

"$fuzzer" -runs=5000 "$work/corpus"
