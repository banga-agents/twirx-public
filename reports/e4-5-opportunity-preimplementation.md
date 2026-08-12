# E4.5 Opportunity admission preimplementation report

Date: 2026-08-12

Branch: `agent/e4-5-opportunity-admission`

Stacked base: `agent/e4-agent-utility-universe`

Implementation commit:
`e14c0875e68fb14d260f4ed893516835e98802a4`

Protected FUTO release baseline:
`9b077663a8d3f86d75f4ad18369d073129d81a85`

Recommendation: **OFFLINE ENGINEERING PASS; HUMAN POLICY DECISION AND REAL
ACQUISITION PENDING**

## Outcome

This subgate implements the fail-closed boundary needed to turn one exact
Grants.gov bulk extract into the first substantial real Opportunity corpus.
It does not claim that acquisition has happened.

Implemented:

- exact, expiring, manual-once source work order;
- explicit kill switch, origin revocation and order revocation;
- exact-host secure retrieval using one-MiB identity-encoded byte ranges;
- evidence-before-archive and manifest-last acquisition;
- full range-to-archive verification;
- bounded ZIP and streaming XML projection;
- mandatory contact, description and attachment exclusions;
- manifest-last public approved-field projection;
- offline Semantic Packet, candidate Mapping Claim and Semantic Frame
  compilation;
- unresolved date-only deadlines and no inferred eligibility or currency;
- controlled two-universe World State and Opportunity investigation;
- parser fuzz targets and adversarial tests.

Not performed:

- no Grants.gov network request;
- no real Grants.gov packet, mapping, frame, query result or corpus count;
- no scheduler, browser, model, authentication, payment or write operation;
- no mapping review, canon promotion, public deployment, merge or production
  state mutation.

## Exact pending authority

The proposal is
`reports/e4-opportunity-policy-decision-proposal.md`. It remains explicitly
pending and grants no retrieval authority. The implemented work-order parser
requires a completed human decision and rejects the pending dossier.

The proposed source scope is exactly:

```text
https://prod-grants-gov-chatbot.s3.amazonaws.com/extracts/GrantsDBExtract20260811v2.zip
```

No filename pattern, later daily object, POST API, redirect, attachment or
caller-supplied URL is accepted.

## Implemented invariants

### Retrieval

- HTTPS and the exact source host, path and filename are fixed constants.
- Network execution requires a valid work-order digest, policy-decision
  digest, UTC validity interval and enabled control artifact.
- Scheduler state is fixed false; redirects are fixed zero; concurrency is
  one; requests, bytes, time and record counts are bounded.
- Safe retrieval revalidates DNS destinations and rejects private, loopback,
  link-local, metadata, multicast and reserved addresses.
- Each range body and evidence object is stored before it contributes to the
  reconstructed archive.
- Acquisition is incomplete without its final manifest.
- Verification recomputes every range, evidence, archive and artifact-index
  digest and reconciles timestamps to the work-order interval.

### Projection and privacy

- Raw ZIP and XML remain private evidence and are not copied to the public
  projection.
- Contact fields, free-form descriptions and unapproved fields never enter
  public projection values.
- The projection directory permits exactly the projection, exclusion report
  and final manifest.
- ZIP traversal, duplicate entries, encryption, excessive expansion,
  unsupported entries and excessive compression ratios fail closed.
- XML directives, DTDs, nested scalars, excessive depth, duplicate
  non-repeatable fields, malformed dates and malformed numeric fields fail
  closed.
- The offline projection source file imports no networking package. The
  semantic importer package also has a repository test prohibiting network,
  process, plugin and `unsafe` imports.

### Source fidelity and semantics

- Every packet preserves its source-native term, lexical value, XML locator
  and original XML representation digest.
- Date-only deadlines retain the native source text and remain unresolved; no
  closing instant or timezone is invented.
- Amounts become typed decimals only when exact source syntax validates; no
  currency is inferred.
- Eligibility codes and text are represented as source statements; no
  applicant-eligibility conclusion is generated.
- All semantic mappings remain candidate-only and carry no review decision.
- Frames remain in the provisional semantic lane.

## Controlled multi-universe proof

The deterministic Atlas Agent can now coordinate the controlled World State
and Opportunity scenarios in one investigation. It runs two exact typed
queries over one immutable two-frame snapshot and returns two independently
proof-linked frames.

```text
evidence class:       test_fixture
current claims made:  false
fixtures public:      false
universes:            2
result frames:        2
proof-linked frames:  2
network requests:     0
browser executions:   0
live source calls:    0
model authority:      none
```

This is not a semantic join and does not claim real cross-origin utility.

## E4.2 off-host restoration

Before starting this subgate, the exact merged E4.2 World State release and
acquisition evidence were archived to the existing dedicated off-host Borg
repository and restored into a new temporary directory.

```text
archive:      e4-world-state-48d89cd4-20260812
fingerprint:  57ff2046afe137d8c53091fb2bb69a865366c424c9e76e24f348e4e60ef853a0
files:        680
original:     677.47 kB
compressed:   452.34 kB
deduplicated: 422.23 kB
hash list:    sha256:107144a0f80003578a31509004de0134b9c0f27d68e2cc46c07e1e73618164cd
```

`borg check --verify-data --show-rc`, file-tree comparison, release-report
comparison and `twirx-e4-worldstate verify-release` all passed on the restored
copy. The private storage endpoint and credential-injection prefix are
intentionally excluded from this public report. No unrelated bucket, backup
path, repository or Meridian service was read or modified.

## Tests and exact commands

```bash
go test ./...
go vet ./...
make build
make docs-check
make demo-e4-investigation
make test-go-fuzz
make test-c
make test-c-fuzz
make test-e2e
make test-snapshot
make verify-e4-worldstate

go test -race \
  ./internal/opportunitypilot \
  ./internal/universeimport \
  ./internal/atlasagent \
  ./cmd/twirx-e4-opportunity \
  ./cmd/twirx-e4-agent

git diff --check
```

All commands passed. The two new Go fuzz targets completed without a failure.
The full restricted-C suite passed under Clang ASan/UBSan, including the 385
canonical E4.2 release objects. Each restricted-C libFuzzer harness completed
5,000 runs without a crash.

## Files changed

Authority, specification and task state:

- `atlas/e4-sources/grants-gov-api/dossier.json`
- `atlas/e4-sources/README.md`
- `reports/e4-opportunity-policy-decision-proposal.md`
- `spec/ontology/OPPORTUNITY_SOURCE_PILOT_0_1.md`
- `spec/ontology/README.md`
- `spec/ontology/VISUAL_ATLAS_AGENT_ALPHA.md`
- `tasks/010-e4-agent-utility-universe-alpha.md`

Retrieval, projection and compilation:

- `internal/opportunitypilot/`
- `cmd/twirx-e4-opportunity/`
- `internal/universeimport/grants_bulk.go`
- `internal/universeimport/grants_bulk_test.go`

Agent and integration:

- `internal/atlasagent/agent.go`
- `internal/atlasagent/agent_test.go`
- `cmd/twirx-e4-agent/main.go`
- `cmd/twirx-e4-agent/main_test.go`
- `Makefile`

## Unresolved risks

1. The exact human source-policy decision is still pending, so real
   acquisition is intentionally blocked.
2. The publisher may replace or remove the exact daily object before approval.
   A different object requires a new human decision, not a widened pattern.
3. The real ZIP/XML shape has not been admitted through this parser. Any source
   discrepancy must fail closed and receive evidence-backed review.
4. The approved projection may still contain sensitive text inside the
   allowed eligibility fields. A real run requires an explicit leakage audit
   before publication.
5. No independent C verifier exists yet for the source-specific projection
   manifest or compact segment. Canonical packets, mappings and frames retain
   the existing independent C verification path.
6. A combined real World State plus Opportunity Universe Snapshot, useful
   query benchmark, proof index and restore test do not exist until the real
   Opportunity acquisition succeeds.
7. This branch has not been merged or deployed and does not change the public
   FUTO release.

## Deviations

- The planned real Opportunity corpus and cross-origin source query were not
  produced because the exact human policy approval has not yet been supplied.
  Proceeding without it would violate the project's policy boundary.
- The off-host Borg command's private endpoint and local credential-injection
  prefix are excluded from the public report; the archive identity, integrity
  results and restoration checks are retained.

## Next recommended gate

After the Genesis steward provides the exact approval text:

1. commit the completed decision and one expiring manual work order;
2. perform the one exact sealed acquisition;
3. independently verify the acquisition and privacy projection;
4. compile real Opportunity packets, candidate mappings and frames;
5. scan the public projection for forbidden and unintended personal data;
6. build a combined immutable Universe Snapshot and proof index;
7. run a useful source-derived World State plus Opportunity query;
8. benchmark and restore-test the exact release;
9. update this report with exact real counts and founder review state;
10. leave the PR unmerged and undeployed for founder review.
