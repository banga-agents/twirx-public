#!/usr/bin/env bash
set -euo pipefail

proof_root="var/e3-worker-proof"
fixture_log="var/e3-worker-fixture.log"
fixture_pid=""

cleanup() {
  if [[ -n "$fixture_pid" ]] && kill -0 "$fixture_pid" 2>/dev/null; then
    kill "$fixture_pid"
    wait "$fixture_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

rm -rf "$proof_root"
mkdir -p var

echo "1/4 Start the explicit loopback fixture origin"
bin/tw-test-origin --addr 127.0.0.1:18081 >"$fixture_log" 2>&1 &
fixture_pid=$!

for _ in $(seq 1 50); do
  if curl --fail --silent --show-error http://127.0.0.1:18081/health >/dev/null; then
    break
  fi
  sleep 0.1
done
curl --fail --silent --show-error http://127.0.0.1:18081/health >/dev/null

echo "2/4 Retrieve only the fixture robots artifact and publish evidence before parsing"
bin/twirx-observer-worker fetch \
  --job conformance/observatory/v1/local-robots-job.json \
  --out "$proof_root"

echo "3/4 Stop the fixture origin"
kill "$fixture_pid"
wait "$fixture_pid"
fixture_pid=""

echo "4/4 Replay the robots decision offline from preserved evidence"
bin/twirx-observer-worker verify --out "$proof_root"

echo "E3 local-fixture retrieval-worker proof passed; no public origin was contacted"
