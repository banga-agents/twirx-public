# E4 Agent Utility Universe foundation report

Date: 2026-08-12

Branch: `agent/e4-agent-utility-universe`

Protected source baseline:
`9b077663a8d3f86d75f4ad18369d073129d81a85`

Validated implementation commit:
`f24eb96962e45e5a8991560733fe7903178fedce`

Recommendation: **FOUNDATION PASS; E4 PUBLIC RELEASE NOT YET ADMITTED**

## Outcome

This branch implements the first bounded E4 foundation without changing or
deploying the protected public FUTO release:

- language-neutral Semantic Frame, Mapping Claim, Ontology Module and Semantic
  Universe contracts;
- deterministic Go codecs and an independent restricted-C verifier;
- 33 shared ontology vectors: 7 accepted and 26 rejected;
- an offline ontology compiler, exact import/closure validation, semantic diff
  and three draft modules;
- a 67-concept draft kernel plus World State and Opportunity modules;
- strict offline World Bank and controlled Grants.gov-shaped importers;
- two immutable Universe Snapshot layout prototypes;
- five deterministic Atlas Agent scenarios, with two controlled resolutions
  and three explicit unresolved outcomes.

No E4 network acquisition, public corpus expansion, model execution, website
deployment or canonical ontology admission occurred.

## Evidence classes and counts

| Class | Source records | Packets | Frames | Public E4 count |
| --- | ---: | ---: | ---: | ---: |
| World Bank tracked controlled replay fixture | 1 | 5 | 1 | 0 |
| Invented Grants.gov-shaped conformance fixture | 1 | 10 | 1 | 0 |
| New E4 current public observations | 0 | 0 | 0 | 0 |
| New E4 archive observations | 0 | 0 | 0 | 0 |

The existing Atlas still contains 500 selected identities. That is a catalog
count, not a claim of 500 compiled or live origins.

Eight E4 source dossiers exist for World Bank, Grants.gov, Crossref, NVD,
Federal Register, APIs.guru, Azure Retail Prices and AWS Price List. All eight
have `network_execution_state: disabled` and E4 review state `pending`.

## Invariants implemented

- frame slots reference only packet digests declared in frame derivation;
- every derivation packet is used by at least one slot;
- semantic frames require mappings; observed-native frames cannot carry them;
- candidate mappings cannot carry a human-review digest;
- released module states require a human-review digest;
- module imports are exact, bounded and cycle-free;
- broader-concept links must resolve through the importing module closure and
  cannot form cycles;
- a universe cannot reference a frame outside its exact module set;
- source bytes are digest-bound and marked stored before parser entry;
- real evidence classes require a nonzero exact policy-decision digest;
- fixture/replay classes cannot imply live policy authority;
- missing World Bank units remain `not_provided`;
- vague grant due-date text remains source-native and semantically unresolved;
- unparseable monetary text remains native and does not become a typed price;
- token-bearing or personal-contact-bearing grant API responses fail closed;
- importers, snapshot prototypes and agent runtime have no network, process,
  plugin or unsafe import path;
- the agent records zero network, browser, live-source and model authority.

## Controlled Universe Snapshot benchmark

Host:

```text
Linux shiva 7.1.3-arch1-1 x86_64
AMD Ryzen 7 6800U with Radeon Graphics
Go 1.26.5-X:nodwarf5 linux/amd64
```

Workload: 10,000 generated `test_fixture` World State frames, four slots per
frame, exact country-slot lookup, no network. These are capacity fixtures and
are not real source-derived frames.

Build/open command:

```bash
go test ./internal/universesnapshot -run '^$' \
  -bench 'Benchmark(Build|Open)(Native|Columnar)10000' \
  -benchmem -benchtime=1x -count=1
```

| Layout | Build time | Serialized bytes | Build allocation | Full verified open | Open allocation |
| --- | ---: | ---: | ---: | ---: | ---: |
| Native postings | 299.8 ms | 18,596,477 | 167,461,600 B | 1.467 s | 242,436,688 B |
| Column scan | 351.0 ms | 22,285,046 | 166,394,624 B | 1.471 s | 242,996,432 B |

Query command:

```bash
go test ./internal/universesnapshot -run '^$' \
  -bench 'Benchmark(Native|Columnar)ExactSlotQuery' \
  -benchmem -benchtime=20x -count=3
```

| Layout | Three observed runs | Allocation |
| --- | --- | ---: |
| Native postings | 50.494 µs, 26.680 µs, 27.041 µs | 744 B/op, 18 allocs/op |
| Column scan | 1.071779 ms, 1.077042 ms, 1.070403 ms | 560 B/op, 13 allocs/op |

The native exact index is the better query candidate in this controlled
workload and produces the smaller release. No production selection is made:
JSON activation allocation is high, steady-state RSS was not isolated, the
workload is synthetic, and an actual isolated analytical sidecar has not been
approved or measured.

## Controlled Atlas Agent demonstration

```bash
go run ./cmd/twirx-e4-agent --root . \
  --scenario world-state.controlled-development

go run ./cmd/twirx-e4-agent --root . \
  --scenario opportunity.controlled-funding
```

Both return a visible typed query, immutable layout plan, frame result and
packet-digest linkage. The wrapper states that fixtures are not public counts.
Research, Security and Agent Economy scenarios return `unresolved` until their
source-specific frames are admitted.

## Source-policy findings

- The World Bank importer can safely parse stored V2 representations, but the
  existing E2 policy decision does not authorize the proposed E4 bulk route.
- Grants.gov documents unauthenticated `search2` and `fetchOpportunity`
  endpoints. Its documented response examples include a token-shaped field
  and public contact/person fields. The implemented fetch profile rejects
  either. A daily bulk-extract route needs its own exact-origin, privacy,
  retention and decompression review before real ingestion.
- The remaining six source dossiers are discovery/design inputs only.

Primary documentation reviewed:

- https://datahelpdesk.worldbank.org/knowledgebase/articles/889392
- https://www.grants.gov/api/api-guide
- https://www.grants.gov/api/common/fetchopportunity
- https://www.grants.gov/help/xml-extract/
- https://www.crossref.org/documentation/retrieve-metadata/rest-api/
- https://nvd.nist.gov/developers/vulnerabilities
- https://www.federalregister.gov/developers/documentation/api/v1
- https://github.com/APIs-guru/openapi-directory
- https://learn.microsoft.com/rest/api/cost-management/retail-prices/azure-retail-prices
- https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/using-the-aws-price-list-bulk-api.html

## Tests and exact commands

Commands completed during foundation development include:

```bash
go test ./internal/dataplane
go test ./internal/ontologyfabric ./cmd/twirx-ontology
go test ./internal/e4vectors ./cmd/twirx-e4-vectors
go test ./internal/e4source
go test ./internal/universeimport
go test ./internal/universesnapshot
go test ./internal/atlasagent ./cmd/twirx-e4-agent
go run ./cmd/twirx-ontology validate --root .
go test ./...
go vet ./...
make build
make test-go-fuzz
make test-c
make test-c-fuzz
make docs-check
make generate-e4-ontology
make generate-e4-vectors
./scripts/test-c-e4-ontology.sh /tmp/tw-verify-dp
./scripts/test-c-e4-ontology.sh /tmp/tw-verify-dp-sanitized
```

All commands above passed in a post-commit rerun against implementation commit
`f24eb96962e45e5a8991560733fe7903178fedce`. The C suite included Clang
ASan/UBSan conformance and 5,000 libFuzzer runs for each restricted verifier
harness. The Go fuzz target set included the new module, World Bank, Grants,
native snapshot and columnar snapshot parsers. The subsequent evidence commit
changes only this report.

## Unresolved risks and deviations

- E4 has zero newly admitted real source-derived packets or frames.
- Only two draft universes exist; three planned universes remain contracts and
  scenarios only.
- Mapping Claims are candidates, not reviewed canon.
- The ontology compiler skeleton does not yet emit SDK types, visualization
  components, database declarations or model labels.
- The Universe Snapshot prototypes are JSON implementation artifacts, not an
  admitted manifest-last public snapshot format.
- Full verified open allocation is too high to project safely to 100,000
  frames; a compact segment codec and isolated steady-RSS measurement are
  required.
- A true analytical sidecar was not added because no dependency/process ADR
  has founder approval.
- Frame traces expose packet identities, but the E4 prototype does not yet
  package complete packet/source proof bundles.
- No genuine E4 delta, cross-origin materialized view, WSIM model or public
  visual UI exists on this branch.

## Next recommended gate

**E4.2 — Real World State admission and compact Universe Snapshot.**

1. Approve an exact bounded World Bank E4 route and retention budget.
2. Acquire a small immutable current batch through the admitted worker.
3. Produce real packets and frames with a proof index and rejection report.
4. Replace JSON frame-body storage with a compact bounded segment and measure
   cold open plus isolated steady RSS at 100,000 controlled frames.
5. Publish no new counter until independent verification and founder review.

In parallel, prepare a separate Grants.gov bulk-extract privacy and archive
decision rather than weakening the fail-closed fetch profile.
