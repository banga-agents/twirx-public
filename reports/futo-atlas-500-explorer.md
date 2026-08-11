# FUTO Atlas-500 public explorer

**Status:** PASS — implementation and local validation complete; public edge
activation remains separately gated by authoritative DNS and TLS evidence.

**Validation date:** 2026-08-11

## Result

The immutable Semantic Snapshot runtime now exposes every one of the 500 exact
Genesis Atlas identities through bounded read-only endpoints and a searchable
Query Lab view:

```text
GET /api/v1/origins?offset=0&limit=500
GET /api/v1/origins/{origin_id}
```

Each description contains only the manifest-bound identity, canonical HTTPS
origin, canonical host, domain family, selection catalog state and the actual
number of public packets in this snapshot. Selection never implies policy
review, retrieval, observation, compilation, publisher approval, health or
live execution.

Against snapshot
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`,
the endpoint returned:

```text
500 selected identities
3 identities with public packets
497 identities with no public packets in this snapshot
15 public packets total
5 controlled fixtures excluded by default
0 origin calls from the runtime
```

## Invariants implemented

- origin enumeration is derived from the already-verified immutable snapshot
  selection rather than a website copy file;
- packet state is derived only after proof-index reconciliation;
- controlled fixtures never become Atlas public-packet counts;
- unknown origin IDs return `404`;
- unknown query parameters, caller-supplied URLs and invalid pagination fail
  closed;
- all 500 descriptions are deterministic and preserve canonical selection
  order;
- no retrieval, scheduler, browser, model, payment, action, database or state
  mutation path was added;
- the protocol remains language-neutral; the HTTP mapping and Go runtime are
  non-normative implementation surfaces.

## Commands executed

```bash
gofmt -w \
  internal/snapshotruntime/runtime.go \
  internal/snapshotruntime/runtime_test.go \
  cmd/twirx-snapshot/main.go \
  cmd/twirx-snapshot/main_test.go

GOMAXPROCS=2 go test \
  ./internal/snapshotruntime \
  ./cmd/twirx-snapshot

GOMAXPROCS=2 make test

cd web
go run .

node --check snapshotlab/static/app.js

bin/twirx-snapshot serve \
  --snapshot var/futo-public-snapshot-d13c0bf \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5 \
  --listen 127.0.0.1:8093

curl -fsS \
  'http://127.0.0.1:8093/api/v1/origins?limit=500'

curl -fsS \
  'http://127.0.0.1:8093/api/v1/origins/api-worldbank-org'
```

The complete offline suite, Go fuzz smoke targets, GCC/Clang builds, ASan,
UBSan, Go/C conformance, 5,000-run C fuzz targets, end-to-end extraction,
Semantic Snapshot integration and documentation checks passed.

## Funding and roadmap correction

Public funding copy now calls the first USD 3,200 allocation **full-time
maintainer engineering capacity**. It does not present private household,
housing or relocation details as project evidence.

The 90-day plan commits to dossiers and completed human policy outcomes across
all 500 selected origins and a bounded profile attempt for every policy-eligible
origin. Denied, uncertain, constrained and unsuccessful outcomes remain public.
The minimum technical floors are not represented as already achieved and do
not create an artificial processing ceiling.

## Unresolved risks

1. Only three explicitly approved policy scopes currently contribute public
   packets. Expanding the packet-bearing set requires additional exact human
   decisions and bounded work orders.
2. `lab.twirx.org` still has no authoritative DNS record, so no public service
   execution is claimed by this report.
3. The immutable snapshot has not yet passed versioned off-host upload and
   independent encrypted restore evidence.

## Deviations

None. The change makes the complete 500-identity universe operationally
inspectable without expanding E3.2 retrieval authority or claiming 500 live
origins.

## Next recommended gate

Publish the sanitized source repository, activate the already-staged Lab only
after authoritative DNS resolves, and complete off-host snapshot restore
evidence in a new TWIRX-specific storage namespace that does not touch Meridian
or quantlab data.
