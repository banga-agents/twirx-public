#!/usr/bin/env bash
set -euo pipefail

work=var/demo-e2
rm -rf -- "$work"
mkdir -p "$work/results"

echo "1/5 Invoke canonical operation in offline replay mode"
bin/twirx-lab invoke --root . --results "$work/results" --origin controlled-origin-lab --operation fixture.getOffer --mode replay --input product_id=demo-1 | tee "$work/result.json"
bundles=("$work/results"/*)
bundle=${bundles[0]}

echo "2/5 Verify the result and re-extract it in Go"
bin/twirx-lab verify --root . --results "$work/results" --bundle "$bundle"

echo "3/5 Verify the same canonical result and bundle in restricted C"
bin/tw-verify-result-c "$bundle"

echo "4/5 Invoke the real local MCP tool over stdio"
{
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"deterministic-demo","version":"1"}}}'
  printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/initialized"}'
  printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"fixture.getOffer","arguments":{"product_id":"demo-1"}}}'
} | bin/twirx-lab mcp --root . --results "$work/mcp-results" --mode replay | tee "$work/mcp-transcript.jsonl"

echo "5/5 Stop all origin access and replay the admitted bundle"
bin/twirx-lab verify --root . --results "$work/results" --bundle "$bundle"
echo "E2 deterministic agent transcript passed"
