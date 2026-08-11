#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
PORT="${TW_TEST_PORT:-18081}"
WORK="var/test-e2e"
CAS="$WORK/cas"
RUN="$WORK/run"
URL="http://127.0.0.1:${PORT}/product/sku-001"
rm -rf "$WORK"
mkdir -p "$WORK"

"$ROOT/bin/tw-test-origin" --addr "127.0.0.1:${PORT}" >"$WORK/server.log" 2>&1 &
SERVER_PID=$!
cleanup() {
  if [[ "${SERVER_PID:-0}" -gt 0 ]]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

for _ in $(seq 1 50); do
  if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

"$ROOT/bin/tw" observe --url "$URL" --out "$RUN" --cas "$CAS" --allow-loopback >/dev/null
"$ROOT/bin/tw" verify --observation "$RUN/observation.cbor" --cas "$CAS" >/dev/null
"$ROOT/bin/tw-verify-c" "$RUN/observation.cbor" "$CAS" >/dev/null

# Stop the origin before extraction. This proves replay does not use the network.
kill "$SERVER_PID" >/dev/null 2>&1 || true
wait "$SERVER_PID" >/dev/null 2>&1 || true
SERVER_PID=0

"$ROOT/bin/tw" extract \
  --observation "$RUN/observation.cbor" \
  --cas "$CAS" \
  --adapter adapters/testorigin-product/adapter.json \
  --out "$RUN/result.json" >/dev/null

python3 - "$RUN/result.json" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as handle:
    result = json.load(handle)

assert result["format"] == "tw.result/0.1"
assert result["operation_id"] == "commerce:getOffer"
assert result["resource_type"] == "commerce:Offer"
fields = {field["id"]: field for field in result["fields"]}
assert fields["price_amount"]["native"]["lexical_value"] == "19.99"
assert fields["price_amount"]["semantic"]["value"] == {"type": "decimal", "lexical": "19.99"}
assert fields["price_currency"]["native"]["lexical_value"] == "usd"
assert fields["price_currency"]["semantic"]["value"] == {"type": "currency_code", "lexical": "USD"}
for field in result["fields"]:
    provenance = field["provenance"]
    assert provenance["body_digest"].startswith("sha256:")
    assert provenance["observation_hash"].startswith("sha256:")
    assert provenance["adapter_digest"].startswith("sha256:")
    assert provenance["locator"] == field["native"]["locator"]
print("e2e: source statement, semantic view, and field-level provenance verified")
PY
