#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - "$ROOT/docs" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
config = json.loads((root / "docs.json").read_text(encoding="utf-8"))
assert config["$schema"] == "https://mintlify.com/docs.json"

missing = []
def visit(node):
    if isinstance(node, str):
        candidate = root / f"{node}.mdx"
        if not candidate.exists():
            missing.append(str(candidate.relative_to(root)))
    elif isinstance(node, list):
        for item in node:
            visit(item)
    elif isinstance(node, dict):
        for key in ("pages", "groups", "tabs", "anchors"):
            if key in node:
                visit(node[key])

visit(config.get("navigation", {}))
if missing:
    raise SystemExit("missing Mintlify pages: " + ", ".join(missing))
print("docs: configuration parsed and navigation targets exist")
PY
