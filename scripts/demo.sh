#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RUN_DIR="var/demo"
CAS_DIR="var/cas"
PORT="${TW_DEMO_PORT:-18080}"
URL="http://127.0.0.1:${PORT}/product/sku-001"
rm -rf "$RUN_DIR"
mkdir -p var

"$ROOT/bin/tw-test-origin" --addr "127.0.0.1:${PORT}" >var/test-origin.log 2>&1 &
SERVER_PID=$!
cleanup() {
  kill "$SERVER_PID" >/dev/null 2>&1 || true
  wait "$SERVER_PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for _ in $(seq 1 50); do
  if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

printf '\n== Observe controlled origin ==\n'
"$ROOT/bin/tw" observe --url "$URL" --out "$RUN_DIR" --cas "$CAS_DIR" --allow-loopback

printf '\n== Verify in Go ==\n'
"$ROOT/bin/tw" verify --observation "$RUN_DIR/observation.cbor" --cas "$CAS_DIR"

printf '\n== Verify independently in C ==\n'
"$ROOT/bin/tw-verify-c" "$RUN_DIR/observation.cbor" "$CAS_DIR"

printf '\n== Extract typed result offline ==\n'
"$ROOT/bin/tw" extract \
  --observation "$RUN_DIR/observation.cbor" \
  --cas "$CAS_DIR" \
  --adapter adapters/testorigin-product/adapter.json \
  --out "$RUN_DIR/result.json"

printf '\n== Provenance-bearing result ==\n'
cat "$RUN_DIR/result.json"
