# E4.5 Opportunity Utility Universe admission report

Date: 2026-08-12

Branch: `agent/e4-5-opportunity-admission`

Stacked base: `agent/e4-agent-utility-universe` at
`784688c0ecb056e2d0e409241009c40c34126c97`

Implementation evidence commit:
`708ce2ea8e5289fa0afe9afe1660524d635bcf56`

Protected public FUTO release baseline:
`9b077663a8d3f86d75f4ad18369d073129d81a85`

Admission: **PASS WITH CONDITIONS — ADMITTED FOR VERSIONED PUBLICATION;
LARGE RUNTIME NOT DEPLOYED**

The Genesis steward admitted the exact E4.5 candidate for publication through
[PR #22](https://github.com/banga-agents/TWIRX/pull/22), merged at
`71770df441105fa39b22195ed2524e6e349bfe7e`. This admission does not authorize
continuous refresh, broaden the one-shot source policy or admit the 1.73 GB
runtime to the shared public host.

## Outcome

The exact founder-approved E4 Opportunity acquisition completed once, under
the approved manual-only work order, and produced a verified privacy-safe
Opportunity Utility release candidate.

The release contains:

```text
83,087  real source Opportunity records and frames
35      admitted World State frames
83,122  combined queryable frames
1,037,679 real source-derived Opportunity packets
747,783 candidate mapping claims
42      packet and mapping artifact segments
```

The local read-only Atlas Agent can query the immutable release without an
origin call, browser or model. A two-universe investigation returns 31 of 31
proof-linked results across World State and Opportunity. No semantic join is
inferred.

This result does not claim 500 compiled origins. It proves one substantial
real Opportunity source family beside the admitted World State slice. Its
code and evidence are admitted for public publication; the larger snapshot
runtime remains undeployed and does not replace the protected public FUTO
Lab baseline.

## Exact authority and one-shot execution

The Genesis steward approved only the proposal in
`reports/e4-opportunity-policy-decision-proposal.md`. The approval was bound
to this decision:

```text
reviewed_at:       2026-08-12T04:33:44Z
review_state:      completed
decision:          permit_with_constraints
proposal digest:   sha256:ec901ccd32462eddfc685be91fb6ce185c791297f319edbb1b7c29923fe4e4e3
decision digest:   sha256:bc4c59630608353c132811985ed000690905a65946dc4fca7506b58f0046bc09
execution mode:    manual_once
scheduler enabled: false
redirects:         0
```

The only authorized source was:

```text
https://prod-grants-gov-chatbot.s3.amazonaws.com/extracts/GrantsDBExtract20260811v2.zip
```

The one-shot acquisition was consumed. Repeating it, retrieving another
file, following a redirect, changing a field set or enabling a schedule would
require new authority.

## Acquisition evidence

```text
started:             2026-08-12T04:39:27Z
completed:           2026-08-12T04:49:03Z
requests:            75
transferred bytes:   77,910,428
archive digest:      sha256:45c3bc26d2217618d4af30b1278f1702e5ba055fb5366f663ca892077f051f7b
acquisition manifest sha256:413b6a3c66e764d88ba613312bc0a9426fd17e7b7b232a5a1240d5d989131b2f
raw evidence public: false
scheduler enabled:   false
```

Every one-MiB range and its evidence object was stored before it contributed
to the reconstructed archive. The archive, range topology, artifact index,
timestamps and manifest were verified after acquisition.

## Projection and privacy evidence

```text
expanded XML bytes:             319,931,017
XML digest:                     sha256:8f56f6754db9aed92f68879657356453389703def80f852eb542086ca2798a75
private projection bytes:       118,878,174
projection digest:              sha256:4c7346419b86e2872da785b0e5b37b643ec5f5c39fe6f283e00b2f2f8f682d94
records seen:                   83,133
records accepted:               83,087
records rejected:               46
contact fields excluded:        247,382
description fields excluded:    83,130
eligibility fields withheld:    59,189
eligibility lexical values out: 0
private projection public:      false
```

The approved eligibility field was found to contain contact-like free text.
The public compiler therefore emits a proof-linked `withheld` state for a
non-empty value and a proof-linked `not_provided` state for an absent value.
It publishes neither the native eligibility prose nor a typed eligibility
conclusion. This is narrower than the approved projection authority.

The release carries this exact non-endorsement notice:

> This product uses the Grants.gov public data source but is not endorsed or
> certified by the U.S. Department of Health and Human Services.

## Immutable release

```text
format:                       tw.e4-opportunity-release/0.1
release manifest digest:      sha256:cbaaa1cd2b41f698f7b423a516727f5a7907bba56ac6c17136528f40f45d7690
release bytes:                1,729,848,598
Opportunity packets:         1,037,679
candidate mapping claims:    747,783
Opportunity frames:          83,087
World State frames:          35
combined frames:             83,122
artifact segments:           42
combined query segment bytes:273,165,514
trust lane:                   provisional_semantic
mapping status:               candidate
runtime origin calls:         0
runtime browser executions:   0
runtime model authority:      none
```

The manifest is written last. Full Go verification recomputes every artifact
digest, parses every packet, mapping claim and frame, validates privacy
invariants, rejects duplicate or unused artifacts and reconciles every frame
reference. The final full verification took 26.405 seconds on the measured
local host.

The manifest-pinned public runtime opens only the combined immutable segment
and public privacy report. It has no dependency on the private acquisition or
projection directories.

## Restricted-C evidence

The committed independent-verifier sample contains the deterministic first,
middle and last canonical packet and mapping from every artifact segment,
plus three frames from each universe:

```text
packets:        63
mapping claims: 63
frames:          6
total:          132
```

The restricted-C verifier accepted all 132 valid objects. Full-corpus C
verification is not claimed; full-corpus verification was performed by Go.

## Agent utility evidence

### Opportunity query

```text
scenario:          opportunity.source-records-nsf
query digest:      sha256:eb892dd40169ebe0851f1615d9677f28ad8544455ecdbb29dfee4aa4d416c965
results:           20
proof linked:      20/20
frames available:  83,122
network requests:  0
browser executions:0
live source calls: 0
model authority:   none
```

The first returned source record is
`grants-gov:opportunity/241323`, titled “NSF Science, Engineering and
Education for Sustainability Fellows.” Its canonical result digest is
`sha256:0048c289f6b76fae6265845444ade6da946b06bc17e98147a9116ffb0efd9642`.
Eligibility is `withheld`, not inferred.

### Two-universe investigation

```text
scenario:             utility.source-world-and-opportunity
universes:            tw:world-state, tw:opportunity
results:              31
proof-linked results: 31
network requests:     0
browser executions:   0
live source calls:    0
model authority:      none
semantic join inferred:false
```

This demonstrates coordinated retrieval across two admitted universes. It is
not presented as an equivalence, ranking or inferred semantic join.

## Performance evidence

The benchmark admitted the immutable 273,165,514-byte combined segment once,
then ran 1,000 warm executions of the exact typed Opportunity query, trace and
decode path with 20 results per execution:

```text
one-time admission: 4,825,020,671 ns
minimum:            1,498,014 ns
median:             2,410,962 ns (2.411 ms)
p95:                3,107,245 ns (3.107 ms)
maximum:            4,403,369 ns
mean:               2,229,698 ns
```

Scope: one local host, warm in-process exact query, no process startup,
public-network latency or origin retrieval. These figures are not a public
service latency claim.

## Off-host durability

The exact release and committed C sample were archived to a new isolated,
encrypted Borg repository path, without reading or modifying unrelated
Storage Box archives, Object Storage buckets, Meridian services or
repositories.

```text
archive:                    e4-opportunity-cbaaa1cd-20260812
repository fingerprint:     078c738fa808f29f6661d24a8974aaae04ff2b91e9e95e1a5f99e6e72293ac63
files:                      179
original bytes:             1,729,990,541
compressed bytes:           280,196,085
deduplicated bytes:         251,941,970
borg verify-data:           PASS
byte-identical restore:     PASS
release tree hash list:     sha256:e94f8b893685634a51e10ebe79ca67bc37330c1b4370a5c63b22818f9e777317
C sample tree hash list:    sha256:1c737d91d5cac319fdfc9320153e6e7357a742ebbdf35312e7f13ca485b2a609
restored release verifier:  PASS
restored C sample verifier: PASS
restored agent result:      31/31 proof linked
```

The restore was performed into a fresh temporary directory and the validated
temporary copy was removed afterward. No endpoint, passphrase or encryption
key appears in this report or repository.

## Invariants implemented

- Retrieval is bound to one exact HTTPS object, one decision digest, one
  time-limited manual-once work order and zero redirects.
- Private, loopback, link-local, metadata, multicast and reserved
  destinations remain blocked; DNS destinations are revalidated.
- Scheduler, browser, model, authentication, payment and write execution
  remain disabled.
- Evidence is stored before archive reconstruction and parsing; manifests are
  published last.
- Required evidence fails closed. Optional absence becomes an explicit
  proof-linked `not_provided` state.
- Every released native source term, lexical value and locator is preserved
  before candidate semantic mapping, except the explicitly withheld
  eligibility prose.
- Dates without a source timezone remain unresolved. Currency and applicant
  eligibility are not inferred.
- Mapping claims remain candidate-only and frames remain provisional.
- Public runtime paths import no networking, shell, process, plugin, cgo or
  `unsafe` capability.
- Artifact ordering and identities are deterministic; malformed, duplicate,
  trailing, unreferenced and digest-mismatched objects fail closed.
- Genesis remains read-only.

## Tests and exact commands

```bash
make test

go test -race \
  ./internal/opportunitypilot \
  ./internal/universeimport \
  ./internal/artifactsegment \
  ./internal/opportunityrelease \
  ./internal/atlasagent \
  ./internal/universesnapshot \
  ./cmd/twirx-e4-opportunity \
  ./cmd/twirx-e4-agent

go vet ./...
make build
make verify-e4-worldstate

bin/twirx-e4-opportunity verify-release \
  --root . \
  --acquisition var/e4-opportunity/grants-gov-20260811/acquisition \
  --projection var/e4-opportunity/grants-gov-20260811/projection \
  --world-release generated/e4/releases/world-bank-e2-matrix \
  --release var/e4-opportunity/grants-gov-20260811/releases/opportunity-utility

./scripts/test-c-e4-opportunity-sample.sh \
  bin/tw-verify-data-plane-c \
  generated/e4/releases/grants-gov-20260811-c-sample

(cd web && go run .)

git grep -n -I -E \
  'AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9_]{20,}|-----BEGIN (RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----|xox[baprs]-[A-Za-z0-9-]{10,}|AIza[0-9A-Za-z_-]{30,}' \
  HEAD

git diff --check
```

All commands passed at the implementation evidence commit. `make test`
included all normal Go tests, all Go fuzz targets for three seconds each,
Clang ASan/UBSan restricted-C verification, both restricted-C libFuzzer
harnesses for 5,000 runs each, the end-to-end provenance check, Semantic
Snapshot integration and documentation validation. The website generator
built 16 pages and passed its claim, link and third-party-request checks.

Two preliminary one-second fuzz smoke runs ended at orchestration shutdown
with `context deadline exceeded`, each on a different target, without a crash
or saved failing input. The repository default was increased from one to
three seconds so target startup and shutdown have a stable budget. The exact
affected targets and the complete final suite then passed.

The credential-pattern scan returned no credential-pattern match. The
project contact address `rick@twirx.org` is intentionally public. A scan of
the committed restricted-C sample found no email address; the apparent
string match `eligible-applicants@0.1x.mapping` is a mapping identifier, not
an address.

## Files changed

Authority and source policy:

- `atlas/e4-decisions/grants-gov-20260811/`
- `atlas/e4-plans/grants-gov-20260811.json`
- `atlas/e4-acquisitions/grants-gov-20260811-manual-control.json`
- `atlas/e4-sources/grants-gov-api/dossier.json`
- `reports/e4-opportunity-policy-decision-proposal.md`

Acquisition, projection, release and runtime:

- `cmd/twirx-e4-opportunity/`
- `internal/opportunitypilot/`
- `internal/opportunityrelease/`
- `internal/artifactsegment/`
- `internal/universeimport/grants_bulk.go`
- `internal/universeimport/grants_bulk_test.go`
- `internal/universesnapshot/compact.go`
- `internal/universesnapshot/prototype_test.go`

Agent and ontology:

- `cmd/twirx-e4-agent/`
- `internal/atlasagent/`
- `ontology/universes/opportunity.json`
- `generated/e4/ontology/`

Independent verification:

- `generated/e4/releases/grants-gov-20260811-c-sample/`
- `scripts/test-c-e4-opportunity-sample.sh`
- `.gitattributes`
- `Makefile`

Public website, documentation and specifications:

- `web/data/e4-utility-release.json`
- `web/pages/`
- `web/site.json`
- `docs/protocol/opportunity-utility-universe.mdx`
- `docs/start/`
- `docs/index.mdx`
- `docs/docs.json`
- `spec/ontology/`
- `tasks/010-e4-agent-utility-universe-alpha.md`
- `reports/e4-5-opportunity-preimplementation.md`
- `reports/e4-5-opportunity-admission.md`

## Unresolved risks

1. The release covers one large Opportunity source family and the small World
   State slice; it is not broad five-universe coverage or 500 compiled
   origins.
2. Mapping claims are candidates and frames are provisional. The system does
   not claim that a source record is open, currently eligible, relevant,
   equivalent or ranked.
3. The raw archive and private projection contain excluded and potentially
   sensitive free text. They remain private evidence and require continued
   access control and retention review.
4. Full release verification is implemented in Go. Restricted C verifies a
   deterministic 132-object sample, not all 1,785,462 packet and mapping
   objects plus frames.
5. The compact query segment and artifact-segment formats have one principal
   full implementation. More independent format-level verification would
   reduce common-mode risk.
6. The release is 1.73 GB, admission took 4.825 seconds and the query segment
   is 273 MB. The query benchmark is local and does not establish VPS or
   public-network performance.
7. One encrypted off-host Borg copy has a verified restore. The planned
   separate Object Storage copy remains pending; unrelated operator storage
   has not been touched.
8. The code and evidence are admitted for versioned publication, but the
   1.73 GB release has not passed target-host runtime admission and is not
   added to the protected public Lab.
9. The exact one-shot authority is consumed. Refreshing this corpus or
   retrieving a later extract requires a new policy decision.
10. Source structure may change in later extracts. Parser and privacy
    behavior must continue to fail closed and receive adversarial tests.

## Deviations

- The approved projection allowed eligibility text, but real evidence showed
  contact-like material inside that field. The implementation takes the
  safer narrower path: the private projection is not published and public
  eligibility lexical values are withheld.
- `OpportunityNumber` was not a unique record key in the real source.
  `OpportunityID` is used as the source-native record identity while the
  repeated opportunity number remains a source field.
- Real source values contained bounded spaces and empty optional XML tags.
  The parser now preserves bounded source whitespace, accepts empty optional
  tags as absent, rejects empty required fields and continues to reject
  control characters.
- Restricted C verification uses a deterministic cross-segment sample rather
  than the full corpus. The limitation is explicit and no full-C claim is
  made.
- The website and documentation expose the admitted evidence while retaining
  the smaller protected Lab snapshot; they do not claim that the large
  Opportunity runtime is deployed.

## Next recommended gate

A separate deployment gate can copy the immutable public subset to a second
isolated Object Storage location, restore and verify it there, measure the
target read-only runtime under its real resource limits, and expose the
manifest-pinned Opportunity scenarios without granting the VPS access to the
private acquisition or projection.

Further semantic work should review a bounded set of mappings and useful
query lenses before adding ranking, eligibility interpretation or more source
families. None of those capabilities is authorized by this report.
