#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work="$root/var/e3-atlas-500-stress"
port=${TW_ATLAS_STRESS_PORT:-18094}
rounds=${TW_ATLAS_STRESS_ROUNDS:-100}
workers=${TW_ATLAS_STRESS_WORKERS:-32}
base="http://127.0.0.1:$port"

for setting in "$port" "$rounds" "$workers"; do
  if [[ ! "$setting" =~ ^[0-9]+$ ]]; then
    echo "Atlas stress settings must be integers" >&2
    exit 2
  fi
done
if (( port < 1024 || port > 65535 || rounds < 1 || rounds > 1000 || workers < 1 || workers > 128 )); then
  echo "Atlas stress settings are outside their safe bounds" >&2
  exit 2
fi
if [[ "$work" != "$root/var/e3-atlas-500-stress" ]]; then
  echo "refusing unexpected work directory" >&2
  exit 2
fi
rm -rf -- "$work"
mkdir -p "$work"

"$root/bin/twirx-admission" atlas-queue \
  --root "$root" \
  --admissions atlas/admissions \
  --fixtures atlas/fixtures \
  --version 2026-08-11 >"$work/admission-queue.json"
jq -e '
  .counts.selected == 500 and
  .counts.prepared_dossiers == 25 and
  .counts.unprepared_dossiers == 475 and
  .counts.policy_review_state.pending == 500 and
  (.origins | length) == 500
' "$work/admission-queue.json" >/dev/null

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
    if curl --fail --silent --show-error "$base/api/v1/atlas/status" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.05
  done
  echo "Atlas did not become healthy on literal loopback" >&2
  return 1
}

start_server() {
  local log=$1
  "$root/bin/twirx-atlas" serve --root "$root" --listen "127.0.0.1:$port" >"$log" 2>&1 &
  server_pid=$!
  wait_healthy
}

start_server "$work/service.log"
curl --fail --silent --show-error "$base/api/v1/atlas/status" >"$work/status-before.json"

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

"$root/bin/twirx-atlas" stress \
  --root "$root" \
  --base "$base" \
  --at 2026-08-11T00:00:00Z \
  --rounds "$rounds" \
  --workers "$workers" \
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
curl --fail --silent --show-error "$base/api/v1/atlas/status" >"$work/status-after.json"
cmp "$work/status-before.json" "$work/status-after.json"
stop_server

jq -e '
  .corpus.selected_origins == 500 and
  .frontier.decisions == 500 and
  .frontier.jobs == 0 and
  .discovery.unique_origins == 500 and
  .discovery.direct_lookups == 500 and
  .adversarial.rejected == 8 and
  .adversarial.recovery == true and
  .workload.requests == (.workload.rounds * 500) and
  .workload.successes == .workload.requests and
  .workload.failures == 0
' "$work/report.json" >/dev/null

jq --slurpfile queue "$work/admission-queue.json" '{status:"PASS",mode,network_access,origins:.corpus.selected_origins,prepared_dossiers:$queue[0].counts.prepared_dossiers,unprepared_dossiers:$queue[0].counts.unprepared_dossiers,frontier_decisions:.frontier.decisions,frontier_jobs:.frontier.jobs,requests:.workload.requests,workers:.workload.workers,requests_per_second:.performance.requests_per_second,p95_microseconds:.performance.p95_microseconds,response_set_digest:.discovery.response_set_digest}' "$work/report.json"
