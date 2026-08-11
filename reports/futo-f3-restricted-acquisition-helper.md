# FUTO F3 restricted archive-acquisition helper

**Restricted-helper recommendation:** PASS

**Real archive-acquisition recommendation:** BLOCKED — founder policy
decisions are not yet complete

**Public-corpus recommendation:** FAIL — zero real archive captures have been
admitted

**Implementation commit:**
`d112b4f91814140888034b5e7362459db31aa864`

**Evidence date:** 2026-08-11

## Result

The implementation commit adds `twirx-archive-acquire`, an operator-only
network boundary for one human-approved, sealed Common Crawl work order. It
can contact only `index.commoncrawl.org` and `data.commoncrawl.org`; no caller
can supply a URL, host, collection, range, port, scheme or redirect policy.

The helper stores each bounded index response before parsing it, derives each
exact WARC byte range from an accepted index record, stores that compressed
range before WARC parsing, passes it to the existing network-incapable
importer, and immediately re-verifies the capture spool. The final acquisition
manifest is written last and binds every raw artifact and capture manifest by
SHA-256.

No real request was executed. The test suite uses only literal-loopback test
servers for the generic range transport and injected in-memory retrievers for
the acquisition workflow. The implementation pass therefore admits the
boundary, not an origin, archive capture, packet, delta or public claim.

## Invariants implemented

- All network authority derives from a validated work order containing a
  completed human review and an explicitly permitting policy decision.
- Index and data hosts are fixed constants. The CLI exposes only work-order
  directory, work-order ID and new output directory inputs.
- Every connection retains HTTPS/TLS hostname verification, fresh DNS
  resolution, public-address enforcement, private/reserved address blocking,
  proxy disabling, destination allowlisting and bounded timeouts.
- Redirects are disabled. Exact request URL, final URL, `GET` method, status,
  byte range and `Content-Range` relationships fail closed.
- Index, compressed-range, WARC, decompressed and retained-body budgets remain
  bounded by the sealed order and Genesis maxima.
- Raw index bytes are written before index parsing. Raw compressed WARC bytes
  are written before WARC or archived-HTTP parsing.
- Failed or interrupted acquisition directories have no publication authority
  because `acquisition-manifest.json` is absent.
- Verification rehashes every acquisition artifact and invokes full offline
  spool verification for every capture.
- Archive classification remains `archive_observation`, `historical`,
  `observed_by: common_crawl`, and
  `current_publisher_statement: false`.
- The existing `twirx-archive` importer remains network-incapable. The public
  snapshot runtime is not linked to the acquisition helper.
- Genesis remains read-only. There is no scheduler, arbitrary URL, adapter or
  browser execution, model, payment, authentication, PostgreSQL deployment or
  canon-promotion path.
- No third-party dependency was added; `go.mod` and `go.sum` are unchanged.

## Conformance and adversarial coverage

The implementation adds a valid acquisition-manifest vector and an invalid
host-authority vector under `conformance/archive-acquisition`. Tests cover:

- exact fixed-host URL and range derivation;
- identity encoding for byte-range requests;
- malformed, reversed, oversized and multi-range rejection;
- nil, non-GET, redirected, wrong-host and wrong-status response rejection;
- exact `206`, body length and `Content-Range` reconciliation;
- raw-evidence retention before parser failure;
- final-manifest absence after failure;
- raw artifact tampering;
- command-line attempts to supply URL, host, collection or range authority;
- bounded strict manifest parsing and fuzzing.

The existing archive-import adversarial and fuzz suite continues to cover
duplicate and ambiguous index records, unsafe paths, WARC/gzip/HTTP framing,
provider-digest mismatch, trailing content and decompression/retention bounds.

## Exact commands executed

The following commands passed against the exact implementation tree later
committed as `d112b4f91814140888034b5e7362459db31aa864`:

```bash
gofmt -w \
  internal/safefetch/safefetch.go \
  internal/safefetch/safefetch_test.go \
  internal/archiveacquire/acquire.go \
  internal/archiveacquire/acquire_test.go \
  cmd/twirx-archive-acquire/main.go \
  cmd/twirx-archive-acquire/main_test.go

go test ./internal/safefetch ./internal/archiveacquire \
  ./cmd/twirx-archive-acquire

go test -race ./internal/safefetch ./internal/archiveacquire \
  ./cmd/twirx-archive-acquire

go vet ./internal/safefetch ./internal/archiveacquire \
  ./cmd/twirx-archive-acquire

GOMAXPROCS=2 make test
GOMAXPROCS=2 go test -race ./...
GOMAXPROCS=2 go vet ./...
git diff --check

if rg -n \
  'http\.(Get|Post|Head|Client|Transport)|net\.Dial|DialContext|ListenAndServe|exec\.Command' \
  cmd/twirx-archive internal/archiveimport; then exit 1; fi

if rg -n 'os/exec|plugin|unsafe|import "C"' \
  cmd/twirx-archive-acquire internal/archiveacquire; then exit 1; fi
```

The complete `make test` result included:

- all Go package tests;
- 22 one-second Go fuzz targets, including the acquisition-manifest target;
- the E1 restricted-C ASan/UBSan suite: two valid vectors accepted, 14 invalid
  vectors rejected, and corrupted evidence rejected;
- shared E2 Go/C conformance;
- 56 E3.3 S1 C vectors: 16 valid accepted and 40 invalid rejected;
- three 5,000-run restricted-C libFuzzer campaigns under ASan and UBSan;
- E2 end-to-end source-statement/provenance replay;
- Semantic Snapshot integration: 18 packets, two public origins, fixtures
  excluded and zero runtime network requests;
- documentation navigation validation.

The race detector and vet passed for every Go package. The working tree's
intended diff passed whitespace validation.

## Toolchain and host

```text
Go:     go1.26.5-X:nodwarf5 linux/amd64
GCC:    16.1.1
Clang:  22.1.8
Kernel: Linux 7.1.3-arch1-1 x86_64
```

## Files changed

- `cmd/twirx-archive-acquire/`: restricted operator CLI and authority-surface
  tests;
- `internal/archiveacquire/`: acquisition runner, manifest-last publication,
  independent reconciliation, adversarial tests and manifest fuzz target;
- `internal/safefetch/`: exact bounded byte-range retrieval using the existing
  DNS, address, TLS, redirect and size policy;
- `conformance/archive-acquisition/`: valid and invalid shared manifest
  fixtures;
- `Makefile`: helper build and fuzz integration;
- `README.md`, `conformance/README.md` and
  `deploy/snapshot/COMMON_CRAWL_IMPORT.md`: component and operator boundaries;
- `tasks/008-futo-restricted-archive-acquisition.md`: task gate and limits.

## Unresolved risks

1. The founder has not yet completed the three explicit FUTO policy decisions.
   No real work order may be generated or executed until those human artifacts
   exist.
2. The exact RFC Editor canonical-host reconciliation is still a proposed
   decision: the existing Atlas record uses `rfc-editor.org`, while useful
   exact archive captures use `www.rfc-editor.org`.
3. No real Common Crawl response has tested the deliberately narrow WARC
   profile. Legitimate archive encodings may require a specified and
   adversarially tested extension.
4. Acquisition metadata and raw evidence are content-bound, but this gate does
   not add a cryptographic human signature to the policy decision.
5. Archive capture spools are not yet compiled into Semantic Packets. No real
   semantic origin delta or cross-origin archive query exists.
6. Object Storage upload/versioning, encrypted Storage Box backup, restore
   proof, public snapshot activation and `lab.twirx.org` remain incomplete.
7. The fresh public repository is locally prepared and scanned but cannot be
   published safely until its final name, tracked website source and founder
   review of the integrated PR are resolved.

## Deviations

No real acquisition was performed because the required human decisions are
absent. No packet compiler, snapshot corpus, Object Storage, Storage Box,
Meridian service, DNS, Caddy, website, Mintlify documentation, funding copy or
unrelated repository was modified. No pull request was merged or deployed.

## Next recommended gate

The Genesis steward should approve or amend the prepared decisions for TWIRX,
World Bank Indicators and the RFC Editor archive pilot. Then generate sealed
work orders from the exact committed decision/evidence digests and execute the
two-period RFC Editor acquisition. Reconcile both capture spools before adding
the smallest deterministic archive-to-packet compiler and genuine origin
delta.
