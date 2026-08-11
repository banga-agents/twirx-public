#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: test-c-e2.sh BUNDLE_VERIFIER ARTIFACT_VERIFIER LAB_CLI" >&2
  exit 2
fi

bundle_verifier=$1
artifact_verifier=$2
lab_cli=$3
work=var/test-c-e2

rm -rf -- "$work"
mkdir -p "$work/results"
"$lab_cli" invoke --root . --results "$work/results" --origin controlled-origin-lab --operation fixture.getOffer --mode replay --input product_id=demo-1 >"$work/invocation.json"
bundles=("$work/results"/*)
if [[ ${#bundles[@]} -ne 1 ]]; then
  echo "expected one generated E2 bundle" >&2
  exit 1
fi
base=${bundles[0]}

while IFS=$'\t' read -r id target mutation expected; do
  if [[ $id == id ]]; then
    continue
  fi
  case_dir="$work/$id"
  cp -a -- "$base" "$case_dir"
  case "$target" in
    bundle) command=("$bundle_verifier" "$case_dir") ;;
    result) command=("$artifact_verifier" result "$case_dir/result.cbor") ;;
    manifest) command=("$artifact_verifier" manifest "$case_dir/manifest.cbor") ;;
    closure) command=("$artifact_verifier" closure "$case_dir/semantic-closure.cbor") ;;
    *) echo "unknown vector target $target" >&2; exit 1 ;;
  esac
  case "$mutation" in
    none) ;;
    append-zero)
      case "$target" in
        result) printf '\0' >>"$case_dir/result.cbor" ;;
        manifest) printf '\0' >>"$case_dir/manifest.cbor" ;;
        closure) printf '\0' >>"$case_dir/semantic-closure.cbor" ;;
      esac
      ;;
    substitute-body) printf '{}' >"$case_dir/representation.body" ;;
    remove-manifest) rm -- "$case_dir/manifest.cbor" ;;
    remove-adapter) rm -- "$case_dir/adapter.cbor" ;;
    replace-contract-with-manifest)
      LC_ALL=C sed -i 's/contract\.cbor/manifest.cbor/' "$case_dir/manifest.cbor"
      ;;
    invalid-cbor) printf '\377' >"$case_dir/result.cbor" ;;
    symlink-body)
      rm -- "$case_dir/representation.body"
      ln -s /etc/passwd "$case_dir/representation.body"
      ;;
    *) echo "unknown vector mutation $mutation" >&2; exit 1 ;;
  esac
  if "${command[@]}" >"$case_dir.stdout" 2>"$case_dir.stderr"; then
    actual=accept
  else
    actual=reject
  fi
  if [[ $actual != "$expected" ]]; then
    echo "$id: got $actual, expected $expected" >&2
    exit 1
  fi
done < conformance/e2/vectors.tsv

echo "E2 C shared conformance vectors passed"
