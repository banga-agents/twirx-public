# E4.2 World State admission and compact snapshot report

Date: 2026-08-12

Branch: `agent/e4-2-world-state-admission`

Stacked base: `agent/e4-agent-utility-universe`

Validated implementation commit:
`34584966a521cdd16e4f4529b0eb4d8a5ea97fe7`

Protected FUTO release baseline:
`9b077663a8d3f86d75f4ad18369d073129d81a85`

Recommendation: **ENGINEERING PASS; FOUNDER MERGE AND DEPLOYMENT REVIEW PENDING**

## Outcome

E4.2 turns the offline E4.1 World Bank compiler into one small real World
State release candidate without broadening the founder-approved route:

- 36 exact requests derived from the existing E2 contract allowlist;
- 36 independently verified immutable current-response evidence spools;
- 35 eligible JSON source records and one retained rejected response;
- 175 canonical Semantic Packets;
- 175 candidate Mapping Claims;
- 35 proof-linked Semantic Frames;
- one manifest-last compact immutable segment;
- one complete frame-to-source proof index;
- independent restricted-C validation of all 385 canonical CBOR objects;
- a reproducible 100,000-frame controlled compact-runtime capacity test.

No scheduler, arbitrary URL, browser, model, authenticated route, payment,
write action, production database, VPS mutation, public deployment, merge, or
canon promotion occurred.

## Exact source authority

The acquisition reuses the completed human decision in
`atlas/admissions/api-worldbank-org/decision.json`, whose exact digest is:

```text
sha256:751f417e96ffda179c5b0b4b4fba4e0a0d926a2d01f00ceae1c7353d08e6772c
```

Only the existing E2 operation route was used:

```text
https://api.worldbank.org/v2/country/{country}/indicator/{indicator}
  ?date={year}&format=json&per_page=1
```

The fixed dimensions were:

```text
countries:  CHL, USA, VNM
indicators: NY.GDP.MKTP.CD, SP.POP.TOTL
years:      2020, 2021, 2022, 2023, 2024, 2025
```

The plan was manual-once, limited to 36 requests and 9 MiB retained bytes,
spaced by at least five seconds, and expired within 24 hours. The program
derives destinations through the committed operation contract and origin
catalog; it accepts no URL from an operator or public caller. The broader E4
World Bank dossier remains pending and is not authorized by this work.

## Acquisition evidence

The acquisition started as a manual local invocation. Eight responses reached
manifest-complete immutable spools before that invocation was interrupted. The
resumed invocation reverified and reused those eight spools, issued the
remaining 28 requests, and wrote the summary last.

Therefore:

```text
network retrievals across both invocations: 36
network requests in the final resumed invocation: 28
verified spools reused by the final invocation: 8
final independently verified response spools: 36
retained response bytes: 11,878
scheduler jobs: 0
```

The committed `network_requests` and per-entry `network_executed` fields refer
to the final resumed invocation, not lifetime acquisition traffic. The
acquisition directory and this report preserve that distinction explicitly.

Thirty-five responses were HTTP 200 `application/json`. The following current
response failed closed and remains immutable rejected evidence:

```text
order: e4-wb-chl-sp-pop-totl-2021
status: 400
media type: text/html
bytes: 1,647
observation: sha256:5e31e524941a26235c6ccab476ef100abd99cd05787f4350268531dc528f6ddc
representation: sha256:0d41cf6a8bf98174b592374f2acd981947f42847cc2e50b3774822ea84a5cada
rejection: response_not_200_json
```

It contributes no packet, mapping claim, frame, or factual statement.

## Compiled release

The undeployed release candidate lives at
`generated/e4/releases/world-bank-e2-matrix`.

| Evidence | Count |
| --- | ---: |
| Current public response spools | 36 |
| Eligible source records | 35 |
| Rejected source responses | 1 |
| Semantic Packets | 175 |
| Candidate Mapping Claims | 175 |
| Semantic Frames | 35 |
| Canonical CBOR objects | 385 |
| Compact segments | 1 |
| Proof-index entries | 35 |

All mappings remain `candidate`; frames remain in the
`provisional_semantic` lane. World Bank content is represented as an origin
statement, not asserted as objective truth. Every packet preserves the native
term, lexical value and exact JSON locator before candidate semantic mapping.

Release identities:

```text
release manifest bytes:
  sha256:48d89cd4e230322507a5fff954f4140691a27c758f7cfe33c7fd42cf6879ede8

compact segment:
  sha256:a86b857c2118586352c5f72b2c36932e5fcc2c774c60b6ecf3d554a62d61c6e8
  52,773 bytes

proof index:
  sha256:04a0dae39aa529a47ad0ec3db8ab689e6ab9852c5aee3e03b8dc6591b4491645
  38,539 bytes
```

The manifest is self-digest-free and written only after all CAS objects,
segment bytes, and proof-index bytes exist. Verification rehashes every
constituent, opens the compact segment read-only, traces all 35 frames, and
resolves every frame through its packets, observation, representation, policy
decision and original immutable spool.

## Compact snapshot invariants

The new dependency-free segment has a language-neutral documented binary
profile and does not redefine protocol identity. It provides:

- detached whole-segment SHA-256 before open;
- strict section and count bounds;
- sorted dictionaries, frame digests and posting keys;
- canonical CBOR frame reconciliation;
- complete posting-index recomputation;
- delta-coded bounded frame IDs;
- exact bounded queries;
- lazy canonical-frame trace;
- symlink rejection and read-only file mapping;
- no network, process, plugin, unsafe, or mutable-state authority.

The restricted-C verifier remains independent. It verifies canonical packet,
mapping-claim and frame objects, not the derived query segment. The Go release
verifier separately verifies and reconciles the complete segment and proof
topology. A future independent compact-segment verifier remains desirable
before treating this implementation format as a stable public interoperability
profile.

## Controlled 100,000-frame capacity evidence

Host:

```text
Linux shiva 7.1.3-arch1-1 x86_64
AMD Ryzen 7 6800U, 8 cores / 16 threads
Go 1.26.5-X:nodwarf5 linux/amd64
```

The generator created 100,000 invented `test_fixture` World State frames.
They are zero real origins, zero source records, zero observations and zero
source-derived packets.

The exact segment was built twice and compared byte-for-byte:

```text
segment: sha256:d588ec7a4d54b9e90d9752f1a553b7c6bdb214f699fa00b86d771dbd5614b08c
bytes: 130,417,488
repeat build: identical
network requests: 0
```

One observed run:

| Metric | Result |
| --- | ---: |
| Fixture generation | 1.668 s |
| Compact build | 3.768 s |
| Build resident high-water | 1,200,214,016 B |
| Verified cold open | 2.779 s |
| Steady RSS after open/query/GC | 170,401,792 B |
| Runtime resident high-water | 270,802,944 B |
| Runtime swap | 0 B |
| Exact queries | 1,000 |
| Query p50 | 1.180 ms |
| Query p95 | 2.239 ms |
| Query p99 | 2.976 ms |

This meets the ADR 015 100,000-frame edge targets of under 2 GiB release,
under 512 MiB steady RSS, under 100 ms p95 and zero origin calls on the measured
workstation. It does not prove Meridian behavior, one-million-packet behavior,
real multi-universe utility, or public-service throughput. Build high-water RSS
also confirms that compilation belongs off Meridian.

## Tests and exact commands

Pre-commit and implementation-commit validation included:

```bash
go test ./...
go vet ./...
make build
make docs-check
make verify-e4-worldstate
make test-go-fuzz
make test-c
make test-c-fuzz

go test -run='^$' -fuzz='^FuzzOpenCompact$' \
  -fuzztime=3s ./internal/universesnapshot

bin/twirx-e4-capacity build \
  --frames 100000 \
  --segment /tmp/twirx-e4-capacity-100000-final.twux \
  --report generated/e4/capacity/controlled-100000-build.json

bin/twirx-e4-capacity open \
  --frames 100000 \
  --queries 1000 \
  --segment /tmp/twirx-e4-capacity-100000-final.twux \
  --digest sha256:d588ec7a4d54b9e90d9752f1a553b7c6bdb214f699fa00b86d771dbd5614b08c \
  --report generated/e4/capacity/controlled-100000-open.json

sha256sum \
  /tmp/twirx-e4-capacity-100000-final.twux \
  /tmp/twirx-e4-capacity-100000-repeat.twux
cmp -s \
  /tmp/twirx-e4-capacity-100000-final.twux \
  /tmp/twirx-e4-capacity-100000-repeat.twux

find atlas/e4-acquisitions atlas/e4-plans generated/e4/capacity \
  generated/e4/releases -type f -name '*.json' -print0 |
  xargs -0 -n 50 jq empty

bash -n scripts/test-c-e4-worldstate-release.sh
git diff --check
```

All commands passed. The C suite used Clang ASan/UBSan and verified exactly
175 packets, 175 mapping claims and 35 frames. The C fuzz suite completed 5,000
libFuzzer runs for each restricted verifier harness. The full Go fuzz target
set included new compact-segment and exact-plan parsers.

## Files changed

Implementation and authority:

- `internal/worldstatepilot/`
- `cmd/twirx-e4-worldstate/`
- `atlas/e4-plans/world-bank-e2-matrix.json`
- `atlas/e4-acquisitions/world-bank-e2-matrix/`
- `internal/universesnapshot/compact.go`
- `internal/e4capacity/`
- `cmd/twirx-e4-capacity/`
- `generated/e4/releases/world-bank-e2-matrix/`
- `generated/e4/capacity/`
- `scripts/test-c-e4-worldstate-release.sh`

Integration and documentation:

- `Makefile`
- `.gitattributes`
- `spec/ontology/UNIVERSE_SNAPSHOT_PROTOTYPES_0_1.md`
- `tasks/010-e4-agent-utility-universe-alpha.md`
- E4 acquisition, release and capacity README files
- this report

No website, Query Lab, FUTO released snapshot, VPS configuration, production
database, origin registry decision, or prior E1-E3 implementation behavior was
changed.

## Unresolved risks and deviations

- This is one real origin, two indicators, three countries and six years—not a
  complete World State universe and not 500 working origins.
- One authorized current request returned an upstream HTTP 400 and is absent
  from the compiled set by design.
- Mappings remain candidate and frames provisional; no ontology canon promotion
  occurred.
- The compact segment has complete Go verification but no second independent
  implementation of that derived index format.
- The real release has not been restored from Object Storage/Storage Box or
  exercised on Meridian; no deployment is authorized by this PR.
- The 100,000-frame measurement is controlled capacity evidence, not a real
  corpus or public load test.
- The final acquisition summary records the resumed invocation's 28 network
  calls plus eight reused spools; the two-invocation lifetime count is explained
  in this report rather than represented by a dedicated schema field.
- Opportunity, Research, Security/Standards and Agent Economy have no newly
  admitted real E4 frames.
- No genuine E4 delta or cross-origin E4 materialized view exists yet.

## Deviations from the E4 program

The broader E4 plan proposed a much larger World Bank indicator set. This gate
intentionally did not request or infer authorization for it. It reused only the
exact founder-reviewed E2 route values, retained a failure as evidence, and
kept all candidate semantics below canon authority.

The analytical-sidecar comparison in ADR 015 remains deferred because no new
runtime dependency or sidecar process has founder approval. The compact native
implementation is measured but not declared the permanent winner.

## Next recommended gate

**Founder review of this E4.2 candidate, then E4.3 Opportunity admission and
cross-origin utility.**

After founder approval and merge—but before deployment—restore this exact
release from off-host immutable storage and repeat the read-only measurement on
Meridian under ADR 010 limits. In a separate scoped PR, complete a Grants.gov
policy/privacy decision, admit a bounded real Opportunity batch, and demonstrate
the first useful cross-origin frame query without models or browser execution.
