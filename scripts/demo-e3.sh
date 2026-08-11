#!/usr/bin/env bash
set -euo pipefail

echo "1/3 Validate the exact offline Genesis-500 selection, policy set, and admitted registry"
bin/twirx-atlas validate --root .

echo "2/3 Generate evidence-derived Atlas metrics"
bin/twirx-atlas metrics --root .

echo "3/3 Produce a deterministic dry-run frontier with network access disabled"
bin/twirx-atlas plan --root . --at 2026-08-10T00:00:00Z

echo "E3.1 offline policy/frontier control demonstration passed; no origin was fetched"
