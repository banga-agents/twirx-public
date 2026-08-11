# E2 replay runtime stress evidence

Status: **IMPLEMENTATION PASS / PUBLIC DEPLOYMENT NOT PERFORMED**

Implementation commit: `e8783e6a653e8ff7a6fabe36254e821784de8200`

Evidence date: 2026-08-11

## Outcome

The replay-only Lab completed 100 mixed HTTP invocations at its admitted
concurrency ceiling of 8. All 100 returned valid typed results; no request was
rate-limited, returned an unexpected status, failed transport, or failed
result validation. The workload exercised all five E2 operations across the
controlled fixture, TWIRX publisher artifact, and recorded World Bank API
representation.

The run produced five distinct content-addressed results. The stress client
retrieved every result view, provenance view, and ZIP proof bundle, then:

- required complete native, semantic, mapping, transformation, and digest
  bindings on every returned field;
- rejected unsafe response headers, redirects, ambiguous JSON, malformed ZIP
  entries, unsafe names, missing manifests, and substituted artifacts;
- rehashed each artifact against the canonical manifest;
- checked the detached result and bundle identifiers;
- decoded and identity-checked each canonical result;
- verified all five on-disk publications in the primary implementation and
  the independent restricted verifier.

The deterministic controlled result remains:

```text
sha256:8bafb410dee23a3e6a5011f81b46531083d82c9395e0b2e9d134b702433ee972
```

## Measured replay run

Measured host: Linux/amd64, AMD Ryzen 7 6800U, Go
`go1.26.5-X:nodwarf5`.

| Measure | Result |
| --- | ---: |
| Invocations | 100 |
| Explicitly simulated clients | 100 |
| Concurrency | 8 |
| Successful typed results | 100 |
| HTTP 429 / unexpected HTTP / invalid / transport failure | 0 / 0 / 0 / 0 |
| Invocation phase wall time | 368,814 us |
| Measured invocation throughput | 271.139 requests/s |
| Full run including proof download and rehash | 383,332 us |
| Mean / p50 / p95 / p99 / maximum response latency | 29.012 / 6.296 / 67.534 / 68.007 / 68.008 ms |
| Typed-response bytes | 261,000 total; 2,610 mean |
| Distinct result/provenance/bundle sets verified | 5 / 5 / 5 |

The resource sampler captured six observations during the short invocation
phase:

| Sampled process measure | Maximum observed |
| --- | ---: |
| Resident set size | 20,324 KiB |
| Threads | 16 |
| File descriptors | 15 |
| Final sampled user + system CPU ticks | 14 |

These are sampled values, not kernel-enforced peaks. The run is intentionally
short because it consumes the exact initial admitted per-origin bursts; it
does not reset or widen production limits to manufacture a larger number.

## Protective-overload run

A new process received 50 requests at concurrency 8 from one client. Its
configured per-client burst admitted 20 and rejected 30 with HTTP 429 and
`Retry-After`:

```text
requests  concurrency  successes  rate_limited  average_seconds  p95_seconds  average_response_bytes
50        8            20         30            0.010793         0.062898     1117
```

The timing and size columns combine admitted and rejected responses and are
not execution-latency measurements.

## Public-service boundary

The deployable HTTP Lab is now unconditionally replay-only:

- omitted mode becomes `replay`;
- `mode: fresh` fails closed before quota or execution admission;
- status and discovery report `execution_mode: replay_only` and
  `fresh_origin_access: false`;
- origin views report `fresh_enabled: false` on this surface;
- generated public OpenAPI binds mode to the constant `replay`;
- the browser selector contains replay only.

Explicit fresh observation remains a local CLI workflow. It cannot be enabled
through an HTTP service flag or configuration field. Future public fresh
access must use the separately admitted egress worker and target-host network
controls.

The diagnostic client accepts only a literal-loopback base URL or the exact
`https://lab.twirx.org` host, refuses redirects and credentials, and permits
simulated `X-Forwarded-For` clients only against literal loopback. No arbitrary
target can be used as a stress destination.

## Exact commands executed

The complete sequence was rerun on the implementation commit:

```bash
make clean
make build
make test
make demo
make demo-e2
make demo-e3
make demo-e3-worker
make stress-e2
go test -race ./...
go vet ./...
```

Supplemental evidence:

```bash
go test -count=1 -json ./...
go test -coverprofile=/tmp/twirx-runtime-stress.cover ./...
go tool cover -func=/tmp/twirx-runtime-stress.cover

gcc -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-stress-gcc \
  verifier/c/e2_main.c verifier/c/e2.c \
  verifier/c/observation.c verifier/c/sha256.c
clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-stress-clang \
  verifier/c/e2_main.c verifier/c/e2.c \
  verifier/c/observation.c verifier/c/sha256.c

make generate-e2
git diff --exit-code -- generated/e2
git diff --check
shellcheck scripts/stress-e2.sh scripts/load-e2.sh
node --check lab/static/app.js
./scripts/check-docs.sh
systemd-analyze security --offline=yes lab/deploy/twirx-lab.service
```

## Complete local results

- 294 named tests passed;
- 14 fuzz targets passed;
- aggregate statement coverage: 66.1%;
- race detector and vet passed;
- strict GCC 16.1.1 and Clang 22.1.8 builds passed;
- Clang ASan and UBSan verification passed;
- observation vectors: 2 valid accepted, 14 invalid rejected, corrupted
  evidence rejected;
- E2 shared canonical result and bundle conformance passed;
- both restricted-verifier libFuzzer targets completed 5,000 runs without a
  crash;
- all E1, E2, E3.1, and literal-loopback Observatory demonstrations passed;
- E2 generated artifacts reproduced with aggregate digest
  `f4a70b94939ff95d0f777dab1a8875956e7079d6b2c1eaddf26de9278c391daf`;
- the Lab systemd candidate received offline exposure score `1.6 OK`;
- no public origin was contacted by the stress run;
- no dependency, arbitrary-URL path, browser, model, or write action was
  added; E1/E2 extraction and canonical-result behavior did not change.

## Unresolved risks and limits

1. This is a same-host literal-loopback replay measurement. It excludes public
   DNS, Caddy, TLS, VPS contention, WAN clients, and service restarts.
2. The corpus is five operations over two recorded public-origin
   representations and one controlled fixture. It does not demonstrate 500
   working origins or cross-origin semantic discovery.
3. Fixed replay bodies cannot measure live-provider latency, drift, upstream
   failure, DNS behavior, robots changes, or fresh evidence retention.
4. The 100-request run consumes the admitted initial origin bursts using
   simulated clients. It is a bounded maximum-burst test, not a soak test or a
   distributed capacity forecast.
5. Resource values are periodic `/proc` samples, not kernel peak accounting;
   CPU ticks are not normalized utilization.
6. In-memory rate limits still reset on restart and are not distributed.
7. The Lab has not been deployed at `lab.twirx.org`; public-surface and VPS
   service evidence remain absent.
8. The sealed fresh-origin worker remains disabled and is not connected to the
   Lab. Its target-host DNS/firewall boundary remains unresolved.
9. This evidence tests correctness, containment, and cost of the runtime. It
   does not by itself establish that users value the interface or that the
   Genesis-500 program is worth its full review and adapter cost.

## Recommendation

**PASS for founder review and a replay-only staging deployment.** The next
useful experiment is to deploy this exact reviewed release behind Caddy, run
the same bounded client through public TLS, collect VPS resource evidence, and
let real reviewers use the five-operation Lab. Do not enable fresh egress or
claim Atlas-500 operation until the separate admission and host-isolation
requirements pass.
