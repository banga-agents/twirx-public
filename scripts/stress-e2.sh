#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work="$root/var/e2-stress"
results="$work/results"
overload_results="$work/overload-results"
port=${TW_STRESS_PORT:-18093}
base="http://127.0.0.1:$port"

if [[ ! "$port" =~ ^[0-9]+$ ]] || (( port < 1024 || port > 65535 )); then
  echo "TW_STRESS_PORT must be an unprivileged TCP port" >&2
  exit 2
fi
if [[ "$work" != "$root/var/e2-stress" ]]; then
  echo "refusing unexpected work directory" >&2
  exit 2
fi
rm -rf -- "$work"
mkdir -p "$work"

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
    if curl --fail --silent --show-error \
      --header 'X-Forwarded-For: 198.51.100.254' \
      "$base/api/v1/status" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.05
  done
  echo "Lab did not become healthy on literal loopback" >&2
  return 1
}

start_server() {
  local result_dir=$1
  local log=$2
  "$root/bin/twirx-lab" serve \
    --root "$root" \
    --results "$result_dir" \
    --static "$root/lab/static" \
    --listen "127.0.0.1:$port" >"$log" 2>&1 &
  server_pid=$!
  wait_healthy
}

start_server "$results" "$work/service.log"
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

"$root/bin/twirx-stress" \
  --base "$base" \
  --workload "$root/stress/e2-replay-workload.json" \
  --requests 100 \
  --concurrency 8 \
  --simulated-clients 100 \
  --proof-samples 32 \
  --out "$work/report.json" >"$work/report.stdout.json"
cmp "$work/report.json" "$work/report.stdout.json"
stop_server

verified=0
while IFS= read -r -d '' bundle; do
  "$root/bin/twirx-lab" verify --root "$root" --results "$results" --bundle "$bundle" >/dev/null
  "$root/bin/tw-verify-result-c" "$bundle" >/dev/null
  verified=$((verified + 1))
done < <(find "$results" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)
if (( verified != 5 )); then
  echo "expected five distinct verified result bundles, got $verified" >&2
  exit 1
fi

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

start_server "$overload_results" "$work/overload-service.log"
"$root/scripts/load-e2.sh" "$base" 50 8 20 | tee "$work/overload.tsv"
stop_server

printf 'stress_status\tinvocations\tconcurrency\tsimulated_clients\tverified_bundles\n'
printf 'PASS\t100\t8\t100\t%d\n' "$verified"
