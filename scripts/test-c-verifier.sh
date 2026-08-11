#!/usr/bin/env bash
set -euo pipefail

VERIFIER="${1:?usage: test-c-verifier.sh VERIFIER}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$ROOT/var/test-c-verifier"
rm -rf "$WORK"
mkdir -p "$WORK"

python3 - "$VERIFIER" "$ROOT" "$WORK" <<'PY'
import hashlib
import json
import pathlib
import subprocess
import sys

verifier = pathlib.Path(sys.argv[1]).resolve()
root = pathlib.Path(sys.argv[2])
work = pathlib.Path(sys.argv[3])
manifest = json.loads((root / "conformance/observation/vectors.json").read_text(encoding="utf-8"))
body = (root / manifest["body_file"]).read_bytes()
actual_digest = hashlib.sha256(body).hexdigest()
if actual_digest != manifest["body_sha256"]:
    raise SystemExit("conformance body digest does not match manifest")

accepted = 0
rejected = 0
valid_case = None
for vector in manifest["vectors"]:
    case = work / vector["id"]
    case.mkdir(parents=True)
    envelope = case / "observation.cbor"
    envelope.write_bytes(bytes.fromhex(vector["cbor_hex"]))
    body_path = case / "cas" / "sha256" / actual_digest[:2] / actual_digest[2:4] / actual_digest
    body_path.parent.mkdir(parents=True)
    body_path.write_bytes(body)
    result = subprocess.run(
        [str(verifier), str(envelope), str(case / "cas")],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )
    should_accept = vector["expected"] == "accept"
    if should_accept and result.returncode != 0:
        raise SystemExit(f"{vector['id']}: expected acceptance\nstdout={result.stdout}\nstderr={result.stderr}")
    if not should_accept and result.returncode == 0:
        raise SystemExit(f"{vector['id']}: expected rejection; invariant={vector['invariant']}")
    if should_accept:
        accepted += 1
        valid_case = (envelope, case / "cas", body_path)
    else:
        rejected += 1

if valid_case is None:
    raise SystemExit("corpus has no accepted vector")
envelope, cas_root, body_path = valid_case
body_path.write_bytes(b"corrupted\n")
result = subprocess.run(
    [str(verifier), str(envelope), str(cas_root)],
    stdout=subprocess.DEVNULL,
    stderr=subprocess.DEVNULL,
    check=False,
)
if result.returncode == 0:
    raise SystemExit("C verifier accepted corrupted CAS evidence")

print(f"c-verifier: {accepted} valid vector accepted; {rejected} invalid vectors rejected; corrupted evidence rejected")
PY
