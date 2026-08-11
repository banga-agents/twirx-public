#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: check-lab-surface.sh BASE_URL" >&2
  exit 2
fi

base=${1%/}
curl --fail --silent --show-error "$base/api/v1/status" >/dev/null
curl --fail --silent --show-error "$base/api/v1/origins" >/dev/null

for path in /.git/config /reports/public-readiness.md /AGENTS.md /var/e2/results /TWIRX_PUBLIC_ALPHA_PACK/; do
  status=$(curl --silent --output /dev/null --write-out '%{http_code}' "$base$path")
  if [[ $status != 404 ]]; then
    echo "$path returned $status, want 404" >&2
    exit 1
  fi
done

status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
  -H 'Content-Type: application/json' \
  --data '{"origin_id":"controlled-origin-lab","operation_id":"fixture.getOffer","mode":"replay","input":{"product_id":"demo-1","url":"http://127.0.0.1/private"}}' \
  "$base/api/v1/invoke")
if [[ $status != 400 ]]; then
  echo "arbitrary URL field returned $status, want 400" >&2
  exit 1
fi

headers=$(curl --silent --show-error --head "$base/")
for required in 'content-security-policy:' 'strict-transport-security:' 'x-content-type-options:' 'x-frame-options:'; do
  if ! grep -qi "^$required" <<<"$headers"; then
    echo "missing security header $required" >&2
    exit 1
  fi
done

echo "Lab public surface checks passed"
