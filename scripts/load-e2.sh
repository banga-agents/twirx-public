#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
base=${1:-http://127.0.0.1:8090}
requests=${2:-20}
concurrency=${3:-8}
expected_successes=${4:-$requests}
if (( requests < 1 || requests > 60 || concurrency < 1 || concurrency > 32 || expected_successes < 0 || expected_successes > requests )); then
  echo "usage: load-e2.sh [BASE_URL] [REQUESTS 1..60] [CONCURRENCY 1..32] [EXPECTED_SUCCESSES]" >&2
  exit 2
fi

work=$root/var/load-e2
rm -rf -- "$work"
mkdir -p "$work"
payload=$work/invoke.json
results=$work/results.tsv
printf '%s\n' '{"origin_id":"controlled-origin-lab","operation_id":"fixture.getOffer","mode":"replay","input":{"product_id":"demo-1"}}' >"$payload"

export TWIRX_LOAD_BASE=$base
export TWIRX_LOAD_PAYLOAD=$payload
# The single-quoted program is expanded by each inner Bash process, which reads
# the two explicitly exported diagnostic variables.
# shellcheck disable=SC2016
seq 1 "$requests" | xargs -P "$concurrency" -I '{}' bash -c '
  curl --silent --show-error --output /dev/null \
    --header "Content-Type: application/json" \
    --data-binary "@$TWIRX_LOAD_PAYLOAD" \
    --write-out "$1\t%{http_code}\t%{time_total}\t%{size_download}\n" \
    "$TWIRX_LOAD_BASE/api/v1/invoke"
' _ '{}' >"$results"

successes=$(awk -F '\t' '$2 == 200 {count++} END {print count+0}' "$results")
limited=$(awk -F '\t' '$2 == 429 {count++} END {print count+0}' "$results")
unexpected=$(awk -F '\t' '$2 != 200 && $2 != 429 {count++} END {print count+0}' "$results")
if (( successes != expected_successes || limited != requests - expected_successes || unexpected != 0 )); then
  echo "load test had $successes successes, $limited rate limits, and $unexpected unexpected responses" >&2
  sort -n "$results" >&2
  exit 1
fi

average=$(awk -F '\t' '{sum+=$3} END {printf "%.6f", sum/NR}' "$results")
p95_line=$(( (requests * 95 + 99) / 100 ))
p95=$(sort -t $'\t' -k3,3n "$results" | awk -F '\t' -v line="$p95_line" 'NR == line {print $3}')
bytes=$(awk -F '\t' '{sum+=$4} END {printf "%.0f", sum/NR}' "$results")

printf 'requests\tconcurrency\tsuccesses\trate_limited\taverage_seconds\tp95_seconds\taverage_response_bytes\n'
printf '%d\t%d\t%d\t%d\t%s\t%s\t%s\n' "$requests" "$concurrency" "$successes" "$limited" "$average" "$p95" "$bytes"
