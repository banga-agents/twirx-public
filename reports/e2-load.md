# E2 load and protective-limit evidence

**Status:** Local replay load passed; configured overload was rejected

**Evidence date:** 2026-08-10

The service ran on loopback on the same AMD Ryzen 7 6800U host as the client.
Every successful request invoked `fixture.getOffer` in offline replay mode,
published the same deterministic result through the idempotent publication
path, and completed post-publication verification. There was no network origin
latency.

Service command:

```bash
bin/twirx-lab serve --root . --results var/e2-load-results \
  --listen 127.0.0.1:8092 --static lab/static
```

Nominal run:

```bash
scripts/load-e2.sh http://127.0.0.1:8092 20 8
```

Result:

```text
requests  concurrency  successes  rate_limited  average_seconds  p95_seconds  average_response_bytes
20        8            20         0             0.025419         0.059545     2679
```

The service was restarted so the per-client token bucket began from a known
state. The protective-overload run used the configured burst of 20 and asked
the harness to require exactly 20 successful requests and 30 HTTP 429
responses:

```bash
bin/twirx-lab serve --root . --results var/e2-overload-results \
  --listen 127.0.0.1:8092 --static lab/static
scripts/load-e2.sh http://127.0.0.1:8092 50 8 20
```

Result:

```text
requests  concurrency  successes  rate_limited  average_seconds  p95_seconds  average_response_bytes
50        8            20         30            0.011270         0.065403     1120
```

The overload timing and response-size columns combine successful and rejected
responses and must not be read as successful invocation latency. Unit and race
tests separately exercise the global concurrency limit, per-origin quota,
bounded client-bucket table, and 16 concurrent attempts to publish one
identical result.

## Limitations

- This is a single-process, single-host loopback smoke/load test, not a
  capacity forecast or distributed stress test.
- Requests used one small replay fixture and one deterministic result ID.
- Caddy, TLS, public network behavior, fresh upstream origins, and VPS resource
  contention are excluded.
- The default rate limit deliberately bounds this diagnostic to a small burst;
  it was not weakened to obtain a larger throughput number.
