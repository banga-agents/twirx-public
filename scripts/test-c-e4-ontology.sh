#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: test-c-e4-ontology.sh VERIFIER" >&2
  exit 2
fi

verifier=$1
work=var/test-c-e4-ontology
rm -rf -- "$work"
mkdir -p "$work"

count=0
accepted=0
rejected=0
while IFS=$'\t' read -r name kind expected reason hex; do
  if [[ $name == name ]]; then
    continue
  fi
  vector="$work/$name.cbor"
  printf '%s' "$hex" | xxd -r -p >"$vector"
  if "$verifier" "$kind" "$vector" >"$work/$name.stdout" 2>"$work/$name.stderr"; then
    actual=accept
    accepted=$((accepted + 1))
  else
    actual=reject
    rejected=$((rejected + 1))
  fi
  if [[ $actual != "$expected" ]]; then
    echo "$name: got $actual, expected $expected ($reason)" >&2
    exit 1
  fi
  count=$((count + 1))
done < conformance/e4-ontology/vectors.tsv

if [[ $count -ne 33 || $accepted -ne 7 || $rejected -ne 26 ]]; then
  echo "unexpected E4.0 corpus counts: total=$count accepted=$accepted rejected=$rejected" >&2
  exit 1
fi

echo "E4.0 C conformance passed: total=$count accepted=$accepted rejected=$rejected"
