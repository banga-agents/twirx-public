#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: test-c-e4-worldstate-release.sh VERIFIER" >&2
  exit 2
fi

verifier=$1
root=generated/e4/releases/world-bank-e2-matrix/cas/sha256

if [[ ! -d $root ]]; then
  echo "E4.2 canonical artifact CAS is unavailable" >&2
  exit 1
fi

total=0
packets=0
mappings=0
frames=0
while IFS= read -r artifact; do
  accepted=0
  accepted_kind=
  for kind in packet mapping-claim frame; do
    if "$verifier" "$kind" "$artifact" >/dev/null 2>&1; then
      accepted=$((accepted + 1))
      accepted_kind=$kind
    fi
  done
  if [[ $accepted -ne 1 ]]; then
    echo "$artifact: accepted as $accepted E4.2 canonical kinds" >&2
    exit 1
  fi
  case $accepted_kind in
    packet) packets=$((packets + 1)) ;;
    mapping-claim) mappings=$((mappings + 1)) ;;
    frame) frames=$((frames + 1)) ;;
  esac
  total=$((total + 1))
done < <(find "$root" -type f -print | LC_ALL=C sort)

if [[ $total -ne 385 || $packets -ne 175 || $mappings -ne 175 || $frames -ne 35 ]]; then
  echo "unexpected E4.2 release counts: total=$total packets=$packets mappings=$mappings frames=$frames" >&2
  exit 1
fi

echo "E4.2 C release verification passed: total=$total packets=$packets mappings=$mappings frames=$frames"
