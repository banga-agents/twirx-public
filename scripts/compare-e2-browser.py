#!/usr/bin/env python3
"""Controlled browser-versus-admitted-adapter measurement outside runtime."""

from __future__ import annotations

import json
from pathlib import Path
import resource
import subprocess
import sys
import tempfile
import time
from urllib.parse import urlparse
from urllib.request import urlopen


def measure(output: Path, errors: Path, command: list[str]) -> dict[str, float | int]:
    start = time.perf_counter()
    with output.open("wb") as stdout, errors.open("wb") as stderr:
        completed = subprocess.run(command, stdout=stdout, stderr=stderr, check=False)
    elapsed = time.perf_counter() - start
    return {
        "exit": completed.returncode,
        "seconds": elapsed,
        "peak_kib": resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss,
    }


def measure_child() -> int:
    metrics = measure(Path(sys.argv[2]), Path(sys.argv[3]), sys.argv[4:])
    print(json.dumps(metrics, sort_keys=True))
    return 0 if metrics["exit"] == 0 else 1


def isolated_measure(output: Path, errors: Path, command: list[str]) -> dict:
    completed = subprocess.run(
        [sys.executable, __file__, "--measure", str(output), str(errors), *command],
        capture_output=True,
        text=True,
        check=True,
    )
    return json.loads(completed.stdout)


def main() -> int:
    root = Path(__file__).resolve().parent.parent
    var = root / "var"
    var.mkdir(exist_ok=True)
    work = Path(tempfile.mkdtemp(prefix="e2-browser-comparison.", dir=var))
    fixture_dir = root / "origins" / "fixtures"
    server = subprocess.Popen(
        [sys.executable, "-m", "http.server", "18082", "--bind", "127.0.0.1"],
        cwd=fixture_dir,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    try:
        for _ in range(50):
            try:
                with urlopen("http://127.0.0.1:18082/controlled-product.json", timeout=0.2) as response:
                    if response.status == 200:
                        break
            except OSError:
                time.sleep(0.05)
        else:
            raise RuntimeError("controlled fixture server did not start")

        profile = work / "chromium-profile"
        profile.mkdir()
        browser = isolated_measure(
            work / "browser-dom.html",
            work / "browser-stderr.txt",
            [
                "chromium", "--headless=new", "--no-sandbox", "--disable-gpu",
                "--disable-background-networking", "--disable-component-update",
                "--disable-default-apps", "--disable-domain-reliability",
                "--disable-extensions", "--disable-sync", "--metrics-recording-only",
                "--no-first-run", "--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1",
                f"--user-data-dir={profile}", f"--log-net-log={work / 'netlog.json'}",
                "--net-log-capture-mode=IncludeSensitive", "--dump-dom",
                "http://127.0.0.1:18082/controlled-product.json",
            ],
        )
        results = work / "typed-results"
        results.mkdir()
        typed = isolated_measure(
            work / "typed-result.json",
            work / "typed-stderr.txt",
            [
                str(root / "bin" / "twirx-lab"), "invoke", "--root", str(root),
                "--results", str(results), "--origin", "controlled-origin-lab",
                "--operation", "fixture.getOffer", "--mode", "replay",
                "--input", "product_id=demo-1",
            ],
        )
        result = json.loads((work / "typed-result.json").read_text())
        agent_input = {
            "operation_id": result["operation_id"],
            "status": result["status"],
            "result": {field["id"]: field["semantic"].get("lexical") for field in result["fields"]},
        }
        agent_bytes = json.dumps(agent_input, separators=(",", ":")).encode()
        (work / "typed-agent-input.json").write_bytes(agent_bytes)

        netlog = json.loads((work / "netlog.json").read_text())
        event_types = netlog["constants"]["logEventTypes"]
        starts = [event for event in netlog["events"] if event["type"] == event_types["URL_REQUEST_START_JOB"] and event.get("params", {}).get("url")]
        reads = [event.get("params", {}).get("byte_count", 0) for event in netlog["events"] if event["type"] == event_types["URL_REQUEST_JOB_FILTERED_BYTES_READ"]]
        hosts = sorted({urlparse(event["params"]["url"]).hostname for event in starts if urlparse(event["params"]["url"]).hostname not in {"127.0.0.1", None}})
        summary = {
            "scope": "One controlled 156-byte JSON fixture on one host; no universal performance claim.",
            "work_directory": str(work.relative_to(root)),
            "browser": {
                **browser,
                "url_jobs": len(starts),
                "blocked_background_hosts": hosts,
                "filtered_response_bytes": sum(reads),
                "agent_input_bytes": (work / "browser-dom.html").stat().st_size,
                "evidence_fields": 0,
            },
            "typed_adapter": {
                **typed,
                "network_requests": 0,
                "agent_input_bytes": len(agent_bytes),
                "full_result_bytes": (work / "typed-result.json").stat().st_size,
                "evidence_fields": len(result["fields"]),
                "result_id": result["result_id"],
            },
        }
        print(json.dumps(summary, indent=2, sort_keys=True))
        return 0
    finally:
        server.terminate()
        try:
            server.wait(timeout=3)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait()


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--measure":
        raise SystemExit(measure_child())
    raise SystemExit(main())
