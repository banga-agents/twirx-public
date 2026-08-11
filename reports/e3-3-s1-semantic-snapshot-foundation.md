# E3.3 S1 Semantic Snapshot foundation evidence

**Recommendation:** PASS for founder review of the S1 engineering candidate;
FAIL for production-state admission, Common Crawl ingestion or public snapshot
deployment

**Evidence date:** 2026-08-11

## Commit binding

```text
admitted E3.2 origin/main  669b2052e7586f2bc05b26ca37a22003a792dba9
S1 implementation tree   2c242ef9e380dd16a0859a650cf2abc1a57994af
```

The review branch was created directly from the admitted `origin/main` merge.
Earlier unpublished local design/implementation commits were combined before
publication so their superseded command transcript could not expose an
unnecessary SSH account or raw host endpoint. No published history was
rewritten.

## Result

S1 now has language-neutral CDDL and prose contracts plus two independent
Genesis implementations for nine canonical object families:

```text
semantic packet
packet batch manifest
semantic delta
semantic query
subscription
query result
materialization manifest
economic event
semantic snapshot manifest
```

All objects use bounded fixed-order deterministic CBOR, empty 0.1 extension
arrays and detached SHA-256 identity. The shared corpus contains 56 vectors:
16 valid documents accepted and 40 malformed or semantically invalid documents
rejected by both Go and restricted C.

The snapshot contract is acyclic and manifest-last. The offline Go verifier
recomputes the detached snapshot ID, validates all manifest bounds and required
roles, and streams every referenced constituent through exact size and SHA-256
checks. On Linux it opens paths component-by-component with `openat`, rejects
symlinks, hard links, devices, FIFOs, sockets and filesystem-boundary changes,
and never follows a caller-supplied artifact path outside the opened root. The
repository remains on its declared Go 1.23 baseline; non-Linux snapshot
directory admission fails closed while canonical codecs still compile.

This is contract and constituent-verification admission only. A snapshot
builder, semantic-index/count reconciliation, query runtime, atomic activation
and off-host restore drill are not implemented by S1.

## Invariants implemented

- Protocol authority remains language-neutral; Go and C are non-normative
  implementations.
- Source-native subject, predicate, locator and lexical value are preserved
  before any semantic mapping.
- Deterministic native typing is allowed without semantic promotion.
- `observed_native`, `provisional_semantic` and `attested_semantic` lanes have
  explicit, fail-closed transition requirements.
- Missing required evidence rejects; optional absence uses explicit `nil` or
  an enumerated unresolved state.
- Packet, result, batch and snapshot cores contain no self-digest.
- Origin, semantic and canon deltas have distinct admissible topologies;
  semantic/canon reinterpretation must preserve source evidence.
- Query execution authority is a bounded typed query, not natural language,
  a URL, SQL, code, a remote MCP server or payment instruction.
- Authority, proof, freshness, mapping status, uncertainty, cost and funding
  remain separate dimensions. Sponsorship cannot alter semantic rank.
- Genesis remains read-only. No browser, model, payment, authentication,
  write action or arbitrary-origin capability was introduced.
- Normal tests and both verifiers require no public network.
- E1, E2 and E3.2 code paths were not refactored or behaviorally weakened.
- Meridian remains a stateless-edge candidate: no PostgreSQL, compiler,
  crawler, corpus or mutable semantic state was installed there.

## Files changed

Design and gate authority:

```text
ARCHITECTURE.md
README.md
THREAT_MODEL.md
decisions/008-semantic-data-plane.md
decisions/009-genesis-data-stack.md
decisions/010-meridian-stateless-snapshot-edge.md
tasks/004-e3-3-semantic-data-plane.md
```

Normative contracts and deployment boundaries:

```text
schemas/cddl/semantic-data-plane.cddl
spec/data-plane/*
deploy/postgresql/*
deploy/snapshot/*
```

Genesis implementations and conformance:

```text
internal/cborlite/cborlite.go
internal/cborlite/cborlite_test.go
internal/dataplane/*
internal/s1vectors/corpus.go
cmd/twirx-s1-vectors/main.go
conformance/e3-s1/vectors.tsv
conformance/README.md
verifier/c/dataplane.c
verifier/c/dataplane.h
verifier/c/dataplane_main.c
verifier/c/fuzz_dataplane.c
scripts/test-c-dataplane.sh
scripts/test-c-dataplane-fuzz.sh
Makefile
```

Evidence:

```text
reports/e3-3-migration-inventory.md
reports/e3-3-vps-capacity-baseline.md
reports/e3-3-semantic-data-plane-design.md
reports/e3-3-s1-snapshot-preimplementation.md
reports/e3-3-s1-semantic-snapshot-foundation.md
```

The S1 implementation commit changes 51 files with 9,968 insertions and 11
deletions against the admitted public base. User/Claude website files,
instruction packs and unrelated untracked reports were not staged or modified.

## Commands and results

Repository-wide validation:

```bash
GOMAXPROCS=2 make test
```

Result: PASS. This ran all Go tests, every existing Go fuzz target plus the new
all-object data-plane fuzz target, the E1/E2/E3.3 sanitized C suites, three
5,000-run C libFuzzer campaigns, all builds, the end-to-end source-statement
test and documentation checks. The S1 C result was:

```text
E3.3 S1 C conformance passed: total=56 accepted=16 rejected=40
```

Race and static analysis:

```bash
go test -race ./...
go vet ./...
git diff --check
```

Result: PASS.

Independent compiler builds:

```bash
gcc -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-verify-data-plane-gcc \
  verifier/c/dataplane_main.c verifier/c/dataplane.c
clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-verify-data-plane-clang \
  verifier/c/dataplane_main.c verifier/c/dataplane.c
./scripts/test-c-dataplane.sh /tmp/tw-verify-data-plane-gcc
./scripts/test-c-dataplane.sh /tmp/tw-verify-data-plane-clang
```

Result: PASS under GCC 16.1.1 and Clang 22.1.8; each accepted 16 and
rejected 40. `make test` separately rebuilt the verifier with ASan and UBSan
and completed 5,000 libFuzzer runs without a crash.

Declared-version and fail-closed portability builds:

```bash
GOOS=darwin GOARCH=amd64 go test -c ./internal/dataplane \
  -o /tmp/dataplane-darwin.test
GOOS=windows GOARCH=amd64 go test -c ./internal/dataplane \
  -o /tmp/dataplane-windows.test.exe
```

Result: PASS. Non-Linux snapshot directory verification returns an explicit
unsupported/fail-closed error; in-memory protocol codecs remain portable.

Generated-artifact reproducibility:

```bash
cp conformance/e3-s1/vectors.tsv /tmp/twirx-e3-s1-vectors-final.tsv
make generate-e2 generate-e3 generate-e3-admission generate-e3-s1
cmp /tmp/twirx-e3-s1-vectors-final.tsv conformance/e3-s1/vectors.tsv
git diff --exit-code -- generated/e2 generated/e3
```

Result: PASS. S1 vectors regenerated byte-identically, and prior E2/E3
generated evidence did not change.

Local codec benchmark, informational only:

```bash
go test -run '^$' -bench='Benchmark(Packet|Query)RoundTrip' \
  -benchmem ./internal/dataplane
```

Host: Linux amd64, AMD Ryzen 7 6800U. Result:

```text
BenchmarkPacketRoundTrip-16  152083  7248 ns/op  89.40 MB/s  936 B/op  45 allocs/op
BenchmarkQueryRoundTrip-16   229878  5677 ns/op  67.81 MB/s  808 B/op  37 allocs/op
```

These are local codec microbenchmarks, not public end-to-end, browser-avoidance
or production-capacity claims.

## Transient validation observation

Two `GOMAXPROCS=4 make test` attempts reached the requested one-second fuzz
interval but the existing `internal/atlas` `FuzzRegistryJSON` harness returned
`context deadline exceeded` during shutdown. The same target passed when rerun
alone, and the complete unchanged suite passed with `GOMAXPROCS=2`. No fuzz
target, duration, corpus, assertion or Makefile behavior was weakened. This is
recorded as test-harness concurrency sensitivity rather than hidden.

## Target-host changes

None. The Meridian inspection was read-only. No VPS file, package, service,
database, firewall, DNS, RAID, Object Storage or Storage Box state changed.
No other repository on the host was inspected or touched.

## Unresolved risks and exclusions

- Founder review and S1 admission are still required; this branch is not
  merged or deployed.
- Snapshot constituent verification does not yet parse semantic indexes or
  reconcile manifest origin/concept/mapping/packet/delta counts against their
  artifact contents. That belongs to the builder/importer runtime gate.
- No snapshot builder, read-only query service, smoke-query set, atomic release
  switch or rollback mechanism exists yet.
- The bounded Common Crawl importer is specified but not implemented or run.
  No archive/profile/packet target count is claimed.
- Object Storage policies, versioning and lifecycle have not been configured
  or evidenced.
- No encrypted Storage Box repository or isolated restore drill exists yet.
- Local PostgreSQL S2 is not implemented; authoritative PostgreSQL on Meridian
  remains prohibited.
- Meridian root RAID remains degraded and its swap pressure remains
  unexplained. Systemd 257 also cannot provide the planned project-directory
  quota, so later disk admission needs structural and free-space checks.
- Linux is the only platform with snapshot directory admission in this
  candidate. Other platforms fail closed.
- The 500-origin Atlas, 25,000/100,000 packet targets, two public views and
  historical change stream remain targets, not results.

## Deviations

No authorized implementation scope was omitted. Deliberately deferred work is
the work the founder decision forbids S1 from implying: production PostgreSQL,
live archive retrieval, credential configuration, mutable edge state,
deployment, website changes, browser/model/payment/action capabilities and
public performance claims.

## Next recommended gate

Founder-review and admit S1 first. Then begin S2 on local development
infrastructure only: reversible PostgreSQL migrations, least-privilege roles,
transactional immutable admission and isolated recovery evidence. Meridian
must remain database-free. After S2 admission, S3 can compile admitted local or
archived observations into byte-identical packets; the read-only snapshot
builder/runtime should be admitted only after S3-S6 can reconcile and query the
artifact contents it serves.
