# E3.3 read-only Semantic Snapshot evidence

**Engineering recommendation:** PASS for local grant-demonstration development

**Production-state recommendation:** NOT ADMITTED

**Implementation commit:**
`aa16fbb3151964d6c62d90ed3b5596d0b3caba9a`

**Evidence date:** 2026-08-11

## Result

The implementation commit builds a deterministic, immutable Semantic Snapshot
from already-admitted E2 offline replay evidence and serves a bounded exact
query and trace surface without a database, browser, model, origin request or
semantic-state write. It is operational enough to run the grant demonstration
locally and to stress the actual query path.

It is not the full Semantic Highway funding target. The exact snapshot contains
500 Atlas identities, but proof-bearing packets exist for only two public
origins and one controlled fixture. The fixture is excluded from public views,
queries, traces and counters by default. No claim of 500 observed, compiled or
live origins is made.

## Exact snapshot

The evidence run used:

```text
source revision: aa16fbb3151964d6c62d90ed3b5596d0b3caba9a
created_at:       2026-08-11T00:00:00Z
snapshot_id:      sha256:0e1b831cc563fc8377d05d90193dd10cb1c888ec9237ccfbc75ad95f2a802670
manifest sha256:  0e1b831cc563fc8377d05d90193dd10cb1c888ec9237ccfbc75ad95f2a802670
snapshot bytes:   293286
snapshot files:   59
```

The manifest was published last. Opening the snapshot verifies every manifest
entry, canonical object and proof relationship before the runtime admits any
query.

### Actual content

| Measure | Actual |
| --- | ---: |
| Atlas identities carried | 500 |
| Public origins with packets | 2 |
| Controlled fixture origins with packets | 1 |
| E2 operations replayed | 5 |
| Semantic packets | 18 |
| Resolved packets | 18 |
| Unresolved packets | 0 |
| Semantic deltas | 0 |
| Materialized views | 2 |
| Embedded E2 proof artifacts | 50 |

The two public origins are the admitted TWIRX project origin and World Bank
Indicators replay evidence already present in E2. The controlled origin is a
test fixture, not a public-origin achievement. All replay-derived packets are
marked stale and do not become current publisher statements.

### Funding-demo targets not yet achieved

| Measure | Target | Actual |
| --- | ---: | ---: |
| Atlas identities | 500 | 500 |
| Archive/index profiles | 100 | 0 |
| Current observations | 25 | 0 |
| Semantic packets | 25000 | 18 |
| Materialized views | 2 | 2 |

The second materialized view is a source-statement view over the small admitted
corpus; this evidence does not claim two broad cross-origin domain views.

## Implemented invariants

- Snapshot construction is offline, deterministic for explicit inputs and
  refuses to overwrite an existing output directory.
- Canonical packets preserve the native source term, source locator and lexical
  value before any semantic mapping.
- E2 observations, transports, adapters, operation contracts, semantic
  closures, result envelopes and bundle manifests are reverified on import.
- A cryptographically self-consistent but semantically false proof-index entry
  is rejected by semantic reconciliation.
- Missing or mismatched required evidence rejects the complete snapshot.
- Public views and default runtime operations exclude controlled fixtures.
- The query profile is read-only and materialized-state-only. Live refresh,
  ontology expansion, actions, payment, authentication and arbitrary URLs fail
  closed as unsupported.
- The HTTP process accepts only literal IPv4 or IPv6 loopback listen addresses.
- Request size, response size, concurrency and time are bounded. Proof downloads
  are selected from admitted digest/name pairs rather than visitor paths.
- Canonical packet, query, query-result and manifest bytes are independently
  accepted by the restricted-C verifier.
- The runtime reports zero origin-network requests and does not contain a
  semantic-state write endpoint.
- Canonical encoders now enforce the same 4 MiB document ceiling as decoders.
- No third-party runtime dependency was added, and E1/E2/E3.2 behavior was not
  weakened.

## Commands executed

All passing commands below ran from the implementation commit unless a result
is explicitly described as a deferred target-host check.

```bash
gofmt -d cmd/twirx-snapshot/*.go \
  internal/dataplane/common.go \
  internal/dataplane/dataplane_test.go \
  internal/dataplane/packet.go \
  internal/dataplane/query.go \
  internal/snapshotartifact/*.go \
  internal/snapshotbuild/*.go \
  internal/snapshotruntime/*.go

go vet ./...

GOMAXPROCS=2 go test -race ./...

GOMAXPROCS=2 make test

make bin/twirx-snapshot bin/tw-verify-data-plane-c

TW_SNAPSHOT_STRESS_REQUESTS=5000 \
TW_SNAPSHOT_STRESS_CONCURRENCY=8 \
make stress-semantic-snapshot

GOMAXPROCS=2 go test -run='^$' \
  -bench='^BenchmarkMaterializedQuery$' \
  -benchmem -count=5 ./internal/snapshotruntime

git diff --check
```

The full `make test` result includes:

- all Go package tests;
- 17 one-second Go fuzz targets, including the new packet-segment and query
  request parsers;
- E1 restricted-C conformance: 2 valid accepted, 14 invalid rejected, corrupted
  evidence rejected;
- E2 shared restricted-C conformance;
- E3.3 S1 restricted-C conformance: 56 vectors, 16 valid accepted and 40 invalid
  rejected;
- three restricted-C libFuzzer runs of 5,000 executions each under ASan and
  UBSan;
- the E2 end-to-end proof;
- the Semantic Snapshot integration, including deterministic rebuild, all 18
  packet verifications, canonical query/result verification and zero origin
  requests;
- documentation navigation validation.

## Stress result

The exact snapshot was served on literal loopback and queried through the HTTP
surface. The client verified every canonical result and required the snapshot,
query and result identities to remain stable. The process was stopped, started
again and required to return byte-identical status.

```text
requests:                        5000
concurrency:                     8
successes:                       5000
failures:                        0
runtime origin-network requests: 0
duration:                        846725 microseconds
throughput:                      5905.104963 requests/second
p50:                             941 microseconds
p95:                             1574 microseconds
p99:                             2005 microseconds
maximum RSS:                     20612 KiB
maximum threads:                 19
maximum file descriptors:        14
```

The stable query identity was
`sha256:9ba2b73f4c43134a104cb89c65031b555d3a99dc4247c7f2a0cb22561e6458e7`;
the stable result identity was
`sha256:00a548768a8a275ca26a88e9bb6dd580c9c7654b89bf6cf84d39016d5e39bc71`.

This is local-loopback evidence on an AMD Ryzen 7 6800U host with a very small
18-packet corpus. It is not a VPS capacity claim and must not be generalized to
the target corpus.

## Query benchmark

Five runs of the in-process materialized query benchmark produced:

```text
31677 ns/op   21800 B/op   59 allocs/op
30751 ns/op   21800 B/op   59 allocs/op
30850 ns/op   21800 B/op   59 allocs/op
31077 ns/op   21800 B/op   59 allocs/op
30303 ns/op   21800 B/op   59 allocs/op
```

The median was 30,850 ns/op. This measures only the current exact-match query
over the 18-packet snapshot; it does not measure archive ingestion, ontology
expansion, PostgreSQL, subscription fan-out or network retrieval.

## Toolchain and host

```text
Go:     go1.26.5-X:nodwarf5 linux/amd64
GCC:    16.1.1
Clang:  22.1.8
Kernel: Linux 7.1.3-arch1-1 x86_64
CPU:    AMD Ryzen 7 6800U with Radeon Graphics
```

## Production admission

Nothing was installed, enabled, started, uploaded or deployed outside the local
workspace. No PostgreSQL instance was installed on Meridian. No RAID, DNS,
Object Storage, Storage Box, Caddy, unrelated repository or Meridian service
was touched.

The repository contains a candidate systemd unit and a target-host verification
script. Running the verifier with its default target returned:

```text
FAIL unit_missing=/etc/systemd/system/twirx-snapshot.service
```

Running `systemd-analyze verify` against the repository copy also reported that
`/srv/twirx/current/bin/twirx-snapshot` is absent. Both results are expected on
the development machine and are recorded as evidence that production
activation has not occurred. They are not production verification passes.

## Unresolved risks and limitations

1. The corpus is 18 packets, not 25,000 or 100,000 packets. Scaling behavior is
   unmeasured.
2. Only two public origins contribute packets. There are no Common Crawl
   profiles and no current live observations in this snapshot.
3. There is no prior admitted snapshot, so the delta count is correctly zero
   and no semantic change-stream claim is supported.
4. The query subset has no ontology expansion, subscription, cross-snapshot
   history, economic filtering or refresh planner.
5. The JSON carriage indexes have only the Go implementation. The independent C
   verifier covers the canonical CBOR constituents, not the JSON carriage.
6. Performance evidence is local and uses a small hot in-memory corpus.
7. Object Storage versioning, encrypted Storage Box backup and restore testing
   remain unexecuted.
8. Target-host filesystem ownership, immutable release activation, cgroups,
   Caddy limits, disk reserve, rollback and service coexistence remain
   unverified.
9. The candidate service limits are repository intent until target-host
   enforcement is independently demonstrated.

## Deviations

There was no Common Crawl retrieval, Object Storage upload, backup, PostgreSQL
deployment or public activation. These exclusions follow the founder
infrastructure decision and are not silent omissions. The snapshot uses the
already-admitted E2 replay corpus so the full trusted cycle can be tested before
large ingestion begins.

## Next recommended gate

After founder review of this local candidate, implement an offline,
archive-assisted corpus gate that produces at least 100 policy-linked archive
profiles and a reproducible 25,000-packet snapshot. Add a second admitted
snapshot only when it can support genuine origin and semantic deltas. Keep
production activation as a separate gate requiring target-host capacity,
systemd/cgroup, Caddy, off-host durability and restore evidence.
