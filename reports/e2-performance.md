# E2 performance evidence

**Status:** Reproducible local measurements; no universal performance claim

**Evidence date:** 2026-08-10

These measurements describe one AMD Ryzen 7 6800U host running Linux
7.1.3-arch1-1 on amd64 with 16 logical CPUs. The toolchain was Go
1.26.5-X:nodwarf5, GCC 16.1.1, and Clang 22.1.8. They measure the committed
controlled 156-byte JSON fixture and the E2 candidate implementation. They do
not predict performance for other origins, representations, operations, or
hosts.

## Go implementation measurements

Command:

```bash
go test -run='^$' -bench=. -benchmem -benchtime=200ms -count=5 \
  ./internal/e2format ./internal/observation \
  ./internal/labengine ./internal/mcpstdio
```

The table reports the median of five samples. The range is the observed
minimum and maximum, not a confidence interval.

| Operation | Median | Observed range | Bytes/op | Allocs/op |
|---|---:|---:|---:|---:|
| Canonical result encode | 1,488 ns | 1,476–1,515 ns | 1,568 | 15 |
| Canonical result decode | 1,534 ns | 1,494–2,238 ns | 720 | 30 |
| Observation v1 encode | 459.5 ns | 450.4–468.5 ns | 560 | 6 |
| Observation v1 decode | 512.1 ns | 510.2–525.0 ns | 379 | 10 |
| Five-field extraction and provenance assembly | 8,902 ns | 8,823–9,029 ns | 6,861 | 126 |
| SHA-256 over 64 KiB | 29,523 ns | 29,294–29,707 ns | 0 | 0 |
| Go proof-bundle verification and re-extraction | 223,984 ns | 220,031–226,759 ns | 64,006 | 619 |
| Offline replay invocation and manifest-last publication | 403,680 ns | 391,047–408,587 ns | 121,448 | 1,120 |
| Local MCP lifecycle plus replay tool call | 505,849 ns | 492,149–592,167 ns | 177,864 | 1,992 |

The 64 KiB SHA-256 sample corresponds to about 2.22 GB/s at the median on
this host. MCP measurement includes parsing initialization and tool-call
messages, executing the operation, publishing and verifying the bundle, and
serializing the response. It does not include a network transport.

## Independent C verifier

Command, executed against the deterministic demonstration bundle:

```bash
env -i PATH=/usr/bin:/bin /usr/bin/python3 scripts/benchmark-e2-c.py \
  var/demo-e2/results/8bafb410dee23a3e6a5011f81b46531083d82c9395e0b2e9d134b702433ee972 \
  100
```

All 100 verifications passed. Total elapsed time was 0.084860180 seconds and
mean elapsed time was 0.000848602 seconds per invocation. Peak child resident
memory was 14,692 KiB. Every sample deliberately includes operating-system
process startup, so this is a command-level measure rather than an isolated C
function benchmark.

## Proof size

The deterministic `fixture.getOffer` proof contains five resolved fields. Its
ten files total 5,250 bytes:

| Artifact | Bytes |
|---|---:|
| `input.cbor` | 34 |
| `semantic-closure.cbor` | 125 |
| `representation.body` | 156 |
| `transport.cbor` | 184 |
| `observation.cbor` | 231 |
| `manifest.cbor` | 575 |
| `transcript.json` | 588 |
| `adapter.cbor` | 853 |
| `result.cbor` | 1,099 |
| `contract.cbor` | 1,405 |

The compact JSON field block containing native values, semantic values,
locators, mappings, and transformations is 1,496 bytes, or 299.2 bytes per
field for this operation. The full provenance endpoint representation is
2,226 bytes before its final newline. A semantic-only five-field agent input
derived from the same result is 191 bytes. These are artifact-size
observations, not evidence that provenance has constant per-field cost.

## Exclusions

- No public-internet latency is included in the Go microbenchmarks.
- Fresh-origin fetch, TLS, DNS, decompression, and upstream rate limits are
  origin-dependent and intentionally reported per live invocation instead.
- Storage is the local filesystem; no distributed object store was measured.
- The C verifier is executed as a separate process; it is not linked into the
  trusted Go request path.
