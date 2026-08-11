# E3.3 sealed Common Crawl offline-importer evidence

**Offline importer recommendation:** PASS

**Network acquisition recommendation:** NOT ADMITTED

**Archive/public-corpus recommendation:** FAIL — zero approved work orders

**Production-state recommendation:** NOT ADMITTED

**Implementation commit:**
`420d6d4ed4c01b86ea0c5178e2129fdf03740238`

**Evidence date:** 2026-08-11

## Result

The implementation commit adds an offline `twirx-archive` tool that validates
sealed Common Crawl work orders, derives exact official-host index and data
requests, parses bounded index and WARC artifacts, publishes raw evidence
before parsing, and independently re-verifies a final immutable capture spool.

The tool contains no network dial, HTTP client, listener, scheduler, arbitrary
URL entry point, policy approval path, adapter execution, semantic mapping or
canon promotion. It made zero public-network requests during this gate.

This result admits the offline artifact boundary. It does not admit network
acquisition or claim any Common Crawl-derived origin, representation, packet,
delta, view or current publisher statement.

## Implemented invariants

- A work order is accepted only with a canonical lowercase HTTPS origin,
  sorted exact same-origin routes, one or two sealed Common Crawl collection
  IDs, bounded capture/request/byte budgets, completed human policy review, a
  retrieval-permitting policy decision, digest references, a safe approval
  reference, reviewer identity and canonical UTC review time.
- Archive evidence is always classified as `archive_observation`,
  `historical`, `observed_by: common_crawl`, and
  `current_publisher_statement: false`.
- Index and data URLs are derived only from the sealed order and the exact
  `index.commoncrawl.org` and `data.commoncrawl.org` hosts.
- The bounded JSON-lines parser rejects duplicate JSON keys, excess records,
  wrong routes, cross-collection or unsafe WARC paths, non-200 captures,
  malformed timestamps and unsigned ranges, records outside the sealed byte
  budget, duplicate capture identities and ambiguous provider digests.
- Import requires an exact `206` response, exact returned byte count and an
  internally consistent `Content-Range` before publication.
- The work order, selected index record, typed range-response evidence and
  compressed WARC member are written and rehashed before the WARC or archived
  HTTP content enters a parser.
- The WARC parser accepts one bounded gzip member, canonical CRLF headers, a
  response record matching the exact route and timestamp, exact content
  length and terminator, and a bounded non-empty HTTP representation with no
  folded or duplicate security-relevant headers or trailing data.
- Common Crawl's provider SHA-1 payload metadata is checked only as provider
  metadata. Every TWIRX artifact identity uses SHA-256.
- `manifest.json` is written last. Verification rehashes, reparses and exactly
  reconciles the work order, index capture, range evidence, WARC record,
  representation and derived capture metadata.
- Input files must be bounded regular files opened without following a final
  symlink. Existing output directories are never overwritten.
- Parser failures leave a completed raw-evidence manifest but cannot create a
  final admitted-capture manifest.
- E1, E2, E3.2 and E3.3 S1 behavior remains unchanged. No dependency was
  added and `go.mod` remains unchanged.

## Operator surface

The offline CLI exposes four commands:

```text
twirx-archive plan
twirx-archive inspect-index
twirx-archive import
twirx-archive verify
```

`plan` reports `network_requests_made: 0`. Acquisition must be performed by a
separately reviewed, restricted operation; supplying a local index or WARC
file does not create policy authority.

## Atlas admission state

The exact Atlas queue at the implementation commit reports:

```text
selected origins:                  500
prepared dossiers:                  25
unprepared dossiers:               475
completed human catalog reviews:     2
pending human catalog reviews:       23
policy review state pending:        500
policy decision uncertain:          500
approved archive work orders:         0
archive captures imported:            0
```

Accordingly, no real work order was invented for testing and no live or
archive acquisition was attempted. Tests use only a controlled `example.org`
fixture constructed in temporary directories.

## Commands executed

The following checks passed against the implementation commit:

```bash
git rev-parse HEAD
# 420d6d4ed4c01b86ea0c5178e2129fdf03740238

git diff HEAD --check
go vet ./...
GOMAXPROCS=2 go test -race ./...

GOMAXPROCS=2 go test -run='^$' \
  -fuzz='^FuzzWorkOrderJSON$' -fuzztime=5s ./internal/archiveimport
GOMAXPROCS=2 go test -run='^$' \
  -fuzz='^FuzzIndexResponse$' -fuzztime=5s ./internal/archiveimport
GOMAXPROCS=2 go test -run='^$' \
  -fuzz='^FuzzCompressedWARC$' -fuzztime=5s ./internal/archiveimport

make build docs-check
GOMAXPROCS=2 make test

bin/twirx-admission atlas-queue \
  --root . --admissions atlas/admissions
bin/twirx-admission review-queue \
  --root . --admissions atlas/admissions

if rg -n \
  'http\.(Get|Post|Head|Client|Transport)|net\.Dial|DialContext|ListenAndServe|exec\.Command' \
  cmd/twirx-archive internal/archiveimport; then exit 1; fi
```

The clean complete `make test` run included:

- all Go package tests;
- 21 one-second Go fuzz targets, including the three new archive targets;
- the E1 restricted-C ASan/UBSan suite: two valid vectors accepted, 14 invalid
  vectors rejected, and corrupted evidence rejected;
- shared E2 Go/C conformance;
- 56 E3.3 S1 C vectors: 16 valid accepted and 40 invalid rejected;
- three 5,000-run restricted-C libFuzzer campaigns under ASan and UBSan;
- E2 end-to-end replay;
- the Semantic Snapshot integration: 18 packets, two public origins, fixtures
  excluded and zero runtime network requests;
- documentation navigation validation.

The archive fuzz targets in that complete run executed 6,236 work-order,
4,779 index-response and 7,059 compressed-WARC mutations without a failure.
The separate five-second campaigns also passed.

One earlier full-suite attempt ended in the pre-existing
`internal/adapter/FuzzDecodeManifest` target with Go's
`context deadline exceeded` after its one-second fuzz budget. The exact target
then passed for five seconds (39,957 executions), and the complete suite passed
on a clean rerun. No archive assertion or sanitizer failure occurred. This
timing sensitivity is retained as a test-harness limitation rather than
reported as a clean first-attempt run.

## Toolchain and host

```text
Go:     go1.26.5-X:nodwarf5 linux/amd64
GCC:    16.1.1
Clang:  22.1.8
Kernel: Linux 7.1.3-arch1-1 x86_64
```

## Files changed

- `cmd/twirx-archive/`: offline planner, inspector, importer and verifier CLI;
- `internal/archiveimport/`: work-order, index, range, WARC/HTTP and immutable
  spool implementation plus adversarial and fuzz tests;
- `Makefile`: binary build and three fuzz targets;
- `README.md`: repository map;
- `deploy/snapshot/COMMON_CRAWL_IMPORT.md`: exact operator boundary and usage;
- `tasks/007-e3-3-common-crawl-offline-importer.md`: gate contract.

## Unresolved risks and limitations

1. All 500 selected origins still have `pending + uncertain` policy state.
   There is no approved archive work order and therefore no real archive
   evidence or archive-derived semantic packet.
2. Network acquisition is deliberately absent. A future worker must enforce
   official-host DNS, TLS, redirect, range, concurrency and global byte
   budgets on the target host before any admitted request is made.
3. Work-order policy and decision digests are auditable bindings inside the
   reviewed artifact, but this gate does not generate work orders from the
   canonical admission dossiers or apply a cryptographic human signature.
4. The WARC profile is intentionally narrow and tested with controlled
   records. Real Common Crawl samples may reveal legitimate encodings or
   header forms that require a specified, adversarially tested extension.
5. Archive capture spools are not yet inputs to the Semantic Snapshot
   compiler. This gate proves evidence acquisition and verification, not
   archive-to-packet compilation, cross-period deltas or materialized views.
6. No archive performance claim is made. Machine dossier time, actual archive
   request bytes, retained evidence, compiler throughput and 500-origin
   resource projections require an admitted pilot.
7. Object Storage upload/versioning, Storage Box encrypted backup, restore
   proof, Meridian activation and production service limits remain
   unexecuted. Production PostgreSQL on Meridian remains forbidden.
8. The one-second Go fuzz smoke targets can be sensitive to host scheduling,
   as recorded above, even when longer isolated runs pass.

## Deviations

No real origin or Common Crawl endpoint was contacted because founder-reviewed
policy decisions and work orders do not exist. No packet compiler integration,
network worker, Object Storage, Storage Box, PostgreSQL, Meridian, Caddy, DNS,
firewall, website, report owned by Claude, or unrelated repository was
modified. No merge or deployment was performed.

## Next recommended gate

Complete explicit human policy decisions for a very small archive pilot,
generate the first work orders directly from the canonical admission dossiers,
and review a restricted official-host acquisition worker. After that boundary
passes, import one or two captures for each approved origin, compile their
source-native statements into Semantic Packets, and prove one historical
origin-delta stream without representing archive evidence as current publisher
state.
