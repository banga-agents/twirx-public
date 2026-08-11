#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work="$root/var/e3-semantic-snapshot-stress"
port=${TW_SNAPSHOT_STRESS_PORT:-18095}
requests=${TW_SNAPSHOT_STRESS_REQUESTS:-5000}
concurrency=${TW_SNAPSHOT_STRESS_CONCURRENCY:-8}
scale_fixture_packets=${TW_SNAPSHOT_SCALE_FIXTURE_PACKETS:-0}
query_path=${TW_SNAPSHOT_STRESS_QUERY:-examples/semantic-query-two-origins.json}
include_fixtures=${TW_SNAPSHOT_INCLUDE_FIXTURES:-0}
source_revision=${TW_SNAPSHOT_SOURCE_REVISION:-$(git -C "$root" rev-parse --verify HEAD)}
base="http://127.0.0.1:$port"

for setting in "$port" "$requests" "$concurrency" "$scale_fixture_packets" "$include_fixtures"; do
  if [[ ! "$setting" =~ ^[0-9]+$ ]]; then
    echo "snapshot stress settings must be integers" >&2
    exit 2
  fi
done
if (( port < 1024 || port > 65535 || requests < 1 || requests > 100000 || concurrency < 1 || concurrency > 64 || scale_fixture_packets < 0 || scale_fixture_packets > 100000 || include_fixtures < 0 || include_fixtures > 1 )); then
  echo "snapshot stress settings are outside safe bounds" >&2
  exit 2
fi
if [[ "$query_path" = /* || "$query_path" == *".."* || ! -f "$root/$query_path" ]]; then
  echo "snapshot stress query must be an existing repository-relative path" >&2
  exit 2
fi
if (( scale_fixture_packets == 0 && include_fixtures != 0 )); then
  echo "fixture-enabled stress requires a non-zero controlled scale corpus" >&2
  exit 2
fi
if [[ ! "$source_revision" =~ ^[0-9a-f]{40}$ ]]; then
  echo "snapshot stress source revision must be an exact lowercase commit SHA" >&2
  exit 2
fi
if [[ "$work" != "$root/var/e3-semantic-snapshot-stress" ]]; then
  echo "refusing unexpected work directory" >&2
  exit 2
fi
rm -rf -- "$work"
mkdir -p "$work"

build_started_ns=$(python3 -c 'import time; print(time.monotonic_ns())')
"$root/bin/twirx-snapshot" build \
  --root "$root" \
  --out "$work/snapshot" \
  --source-revision "$source_revision" \
  --created-at 2026-08-11T00:00:00Z \
  --scale-fixture-packets "$scale_fixture_packets" >"$work/build.json"
build_finished_ns=$(python3 -c 'import time; print(time.monotonic_ns())')
snapshot_bytes=$(du -sb "$work/snapshot" | awk '{print $1}')
snapshot_files=$(find "$work/snapshot" -type f -print | wc -l)
python3 - "$work/build-metrics.json" "$build_started_ns" "$build_finished_ns" "$snapshot_bytes" "$snapshot_files" <<'PY'
import json
import pathlib
import sys

result = {
    "build_duration_microseconds": (int(sys.argv[3]) - int(sys.argv[2])) // 1000,
    "snapshot_bytes": int(sys.argv[4]),
    "snapshot_files": int(sys.argv[5]),
}
pathlib.Path(sys.argv[1]).write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
"$root/bin/twirx-snapshot" verify --snapshot "$work/snapshot" >"$work/verify.json"
"$root/bin/tw-verify-data-plane-c" snapshot "$work/snapshot/manifest.cbor"

if (( scale_fixture_packets > 0 )); then
  "$root/bin/twirx-snapshot" query \
    --snapshot "$work/snapshot" \
    --file "$root/$query_path" >"$work/public-boundary.json"
  "$root/bin/twirx-snapshot" query \
    --snapshot "$work/snapshot" \
    --file "$root/$query_path" \
    --include-fixtures >"$work/fixture-query.json"
  python3 - "$work/public-boundary.json" "$work/fixture-query.json" "$work/snapshot/packets" "$work/c-packet-samples" <<'PY'
import base64
import json
import pathlib
import sys

public = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
fixture = json.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8"))
assert public["status"] == "unresolved" and not public["rows"]
assert fixture["status"] == "resolved" and len(fixture["rows"]) == 1
assert fixture["rows"][0]["origin_id"] == "controlled-scale-corpus-fixture"

packet_dir = pathlib.Path(sys.argv[3])
destination = pathlib.Path(sys.argv[4])
destination.mkdir(mode=0o750)
segments = sorted(packet_dir.glob("segment-*.json"))
assert len(segments) > 1
sample_count = 0
for segment_path in segments:
    segment = json.loads(segment_path.read_text(encoding="utf-8"))
    assert segment["entries"]
    candidates = [segment["entries"][0]]
    if len(segment["entries"]) > 1:
        candidates.append(segment["entries"][-1])
    for entry in candidates:
        digest = entry["digest"].removeprefix("sha256:")
        (destination / f"{digest}.cbor").write_bytes(base64.b64decode(entry["cbor"], validate=True))
        sample_count += 1
assert sample_count >= 4
PY
  c_packet_samples=0
  for packet in "$work/c-packet-samples"/*.cbor; do
    "$root/bin/tw-verify-data-plane-c" packet "$packet"
    c_packet_samples=$((c_packet_samples + 1))
  done
  printf 'restricted_c_packet_samples=%d\n' "$c_packet_samples" >"$work/c-verification.txt"
fi

server_pid=0
sampler_pid=0
stop_server() {
  if (( server_pid > 0 )); then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
    server_pid=0
  fi
  if (( sampler_pid > 0 )); then
    wait "$sampler_pid" >/dev/null 2>&1 || true
    sampler_pid=0
  fi
}
trap stop_server EXIT

wait_healthy() {
  local attempt
  for ((attempt = 0; attempt < 100; attempt++)); do
    if curl --noproxy '*' --fail --silent --show-error "$base/api/v1/status" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.05
  done
  echo "snapshot runtime did not become healthy on literal loopback" >&2
  return 1
}

start_server() {
  local log=$1
  local fixture_arguments=()
  if (( include_fixtures == 1 )); then
    fixture_arguments+=(--include-fixtures)
  fi
  "$root/bin/twirx-snapshot" serve --snapshot "$work/snapshot" --listen "127.0.0.1:$port" "${fixture_arguments[@]}" >"$log" 2>&1 &
  server_pid=$!
  wait_healthy
}

start_server "$work/service.log"
curl --noproxy '*' --fail --silent --show-error "$base/api/v1/status" >"$work/status-before.json"

resources="$work/resources.tsv"
printf 'rss_kib\tthreads\tfds\tcpu_ticks\n' >"$resources"
(
  while kill -0 "$server_pid" >/dev/null 2>&1; do
    rss=$(awk '/^VmRSS:/ {print $2}' "/proc/$server_pid/status" 2>/dev/null || true)
    threads=$(awk '/^Threads:/ {print $2}' "/proc/$server_pid/status" 2>/dev/null || true)
    fds=$(find "/proc/$server_pid/fd" -mindepth 1 -maxdepth 1 -print 2>/dev/null | wc -l)
    ticks=$(awk '{print $14 + $15}' "/proc/$server_pid/stat" 2>/dev/null || true)
    if [[ -n "$rss" && -n "$threads" && -n "$ticks" ]]; then
      printf '%s\t%s\t%s\t%s\n' "$rss" "$threads" "$fds" "$ticks" >>"$resources"
    fi
    sleep 0.05
  done
) &
sampler_pid=$!

"$root/bin/twirx-snapshot" stress \
  --base "$base" \
  --query "$root/$query_path" \
  --requests "$requests" \
  --concurrency "$concurrency" \
  --out "$work/report.json" >"$work/report.stdout.json"
cmp "$work/report.json" "$work/report.stdout.json"
stop_server

awk -F '\t' '
  NR > 1 {
    samples++
    if ($1 > max_rss) max_rss=$1
    if ($2 > max_threads) max_threads=$2
    if ($3 > max_fds) max_fds=$3
    if ($4 > max_ticks) max_ticks=$4
  }
  END {
    if (samples == 0) exit 1
    print "samples\tmax_rss_kib\tmax_threads\tmax_fds\tfinal_cpu_ticks"
    printf "%d\t%d\t%d\t%d\t%d\n", samples, max_rss, max_threads, max_fds, max_ticks
  }
' "$resources" | tee "$work/resource-summary.tsv"

start_server "$work/restarted-service.log"
curl --noproxy '*' --fail --silent --show-error "$base/api/v1/status" >"$work/status-after.json"
cmp "$work/status-before.json" "$work/status-after.json"
stop_server

python3 - "$work/report.json" "$requests" "$concurrency" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
assert report["format"] == "tw.snapshot-stress-report/0.1"
assert report["pass"] is True
assert report["workload"]["requests"] == int(sys.argv[2])
assert report["workload"]["concurrency"] == int(sys.argv[3])
assert report["workload"]["successes"] == int(sys.argv[2])
assert report["workload"]["failures"] == 0
assert report["workload"]["runtime_origin_network_requests"] == 0
assert report["snapshot_id"].startswith("sha256:")
assert report["result_digest"].startswith("sha256:")
PY

python3 - "$work/report.json" "$work/resource-summary.tsv" "$work/build.json" "$work/build-metrics.json" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
resources = pathlib.Path(sys.argv[2]).read_text(encoding="utf-8").splitlines()[1].split("\t")
build = json.loads(pathlib.Path(sys.argv[3]).read_text(encoding="utf-8"))
build_metrics = json.loads(pathlib.Path(sys.argv[4]).read_text(encoding="utf-8"))
summary = {
    "status": "PASS",
    "packets": build["actual"]["packets"],
    "public_packets": build["actual"]["public_packets"],
    "fixture_packets": build["actual"]["fixture_packets"],
    "build_duration_microseconds": build_metrics["build_duration_microseconds"],
    "snapshot_bytes": build_metrics["snapshot_bytes"],
    "snapshot_files": build_metrics["snapshot_files"],
    "requests": report["workload"]["requests"],
    "concurrency": report["workload"]["concurrency"],
    "runtime_origin_network_requests": 0,
    "requests_per_second_millionths": report["performance"]["requests_per_second_millionths"],
    "p95_microseconds": report["performance"]["p95_microseconds"],
    "max_rss_kib": int(resources[1]),
    "max_threads": int(resources[2]),
    "max_fds": int(resources[3]),
}
print(json.dumps(summary, sort_keys=True))
PY
