#!/usr/bin/env python3
"""Measure the restricted C bundle verifier as a complete process."""

from __future__ import annotations

import json
from pathlib import Path
import resource
import subprocess
import sys
import time


def main() -> int:
    if len(sys.argv) not in {2, 3}:
        print("usage: benchmark-e2-c.py BUNDLE_DIR [RUNS]", file=sys.stderr)
        return 2
    root = Path(__file__).resolve().parent.parent
    verifier = root / "bin" / "tw-verify-result-c"
    bundle = Path(sys.argv[1]).resolve()
    runs = int(sys.argv[2]) if len(sys.argv) == 3 else 100
    if not verifier.is_file() or not bundle.is_dir() or not 1 <= runs <= 10000:
        print("verifier, bundle directory, or run count is invalid", file=sys.stderr)
        return 2
    start = time.perf_counter()
    failures = 0
    for _ in range(runs):
        completed = subprocess.run(
            [str(verifier), str(bundle)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        failures += completed.returncode != 0
    elapsed = time.perf_counter() - start
    result = {
        "runs": runs,
        "failures": failures,
        "total_seconds": elapsed,
        "mean_seconds": elapsed / runs,
        "peak_child_kib": resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss,
        "scope": "Each sample includes operating-system process startup.",
    }
    print(json.dumps(result, indent=2, sort_keys=True))
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
