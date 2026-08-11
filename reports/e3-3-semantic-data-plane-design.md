# E3.3 Semantic Data Plane Alpha — design evidence

Date: 2026-08-11

Design base: `9aebad7ffaa888c070352bf8247b21e24e6b5213`

Recommendation: **PASS for founder review of the design; FAIL for runtime implementation, database deployment, ingestion or public release.**

## Outcome

The Semantic Highway founder direction is represented as a language-neutral,
gated E3.3 architecture without modifying E1, E2 or E3.2 execution behavior.
The design establishes:

- semantic packets as immutable reusable units that preserve native source
  terms, locators and lexical values;
- detached SHA-256 identities and manifest-last batch publication;
- explicit `observed_native`, `provisional_semantic` and
  `attested_semantic` lanes;
- distinct origin, semantic and canon delta classes;
- rebuildable proof-linked current heads and materialized views;
- typed query, result, subscription, materialization and economic contracts;
- exact, lexical and bounded-ontology retrieval followed by hard filters, a
  visible Pareto frontier and explicit caller preference policy;
- a compact agent interface rather than one tool per origin;
- PostgreSQL 18 as a non-normative Genesis implementation choice, with global
  digest identities outside time partitions, append-only runtime roles,
  transactional head/delta/outbox admission and no mandatory vector path;
- a migration path that references existing E1/E2/E3.2 artifacts without
  rewriting them;
- independently testable S1-S10 subgates through packet conformance, storage,
  compiler, deltas, views, query/subscription, corpus, public stream,
  benchmarks and economics.

Natural language may propose a typed query but cannot execute it. No canonical
query field accepts an arbitrary URL, SQL, browser instruction, credential,
remote MCP address, payment instruction or action. A live refresh can request
only an already admitted E3.2 origin/route work order.

## VPS finding and deployment decision

A read-only target audit found ample nominal CPU, RAM and remaining disk, but
the root `/dev/md2` RAID1 array is degraded with only one active member
(`[U_]`). The second NVMe partition is mounted separately for `/data` and
Docker rather than participating in the mirror. Swap was also almost fully
used and PostgreSQL is not installed.

Database installation and state admission are therefore blocked. Required
remediation includes a healthy or explicitly replaced durability design,
swap/shared-workload diagnosis, encrypted off-host base backup and WAL
archiving, and a successful isolated point-in-time restore. No RAID, Docker,
firewall, service, DNS, repository or package state was changed.

Full measured values, read-only commands and exclusions are in
`reports/e3-3-vps-capacity-baseline.md`. The inactive operator and recovery
design is in `deploy/postgresql/`.

## Files changed

### Architecture and decisions

- `decisions/008-semantic-data-plane.md`
- `decisions/009-genesis-data-stack.md`
- `ARCHITECTURE.md`
- `README.md`
- `THREAT_MODEL.md`

### Normative design

- `schemas/cddl/semantic-data-plane.cddl`
- `spec/data-plane/README.md`
- `spec/data-plane/SEMANTIC_PACKET_0_1.md`
- `spec/data-plane/QUERY_DELTA_FABRIC_0_1.md`
- `spec/data-plane/GENESIS_STATE_STORE_0_1.md`
- `spec/data-plane/GENESIS_RELATIONAL_SCHEMA.sql`
- `spec/data-plane/SECURITY_0_1.md`
- `spec/data-plane/BENCHMARK_0_1.md`
- `spec/data-plane/E3_3_SUBGATES.md`
- `tasks/004-e3-3-semantic-data-plane.md`

### Operations and migration evidence

- `deploy/postgresql/README.md`
- `deploy/postgresql/RECOVERY.md`
- `reports/e3-3-migration-inventory.md`
- `reports/e3-3-vps-capacity-baseline.md`
- `reports/e3-3-semantic-data-plane-design.md`

No production Go, C, adapter, origin, test, workflow, dependency, website or
VPS file was changed. Claude's untracked `web/` work and every other untracked
pack/report were left untouched. The unrelated VPS repository named by the
founder was not inspected or modified.

## Invariants implemented in the design

- The protocol remains language-neutral; PostgreSQL/Go/C are implementation
  choices, not normative authority.
- Source claims remain bounded to origin representation and derivation, never
  objective truth.
- Native meaning precedes and survives semantic mapping.
- Missing required evidence fails closed; optional absence has an explicit
  state.
- Packet, delta, result and manifest cores contain no self-digest.
- Origin/semantic/canon change cannot be conflated.
- Runtime state can be rebuilt from immutable artifacts.
- Fixture scope cannot enter public-origin state.
- E3.2 policy/work-order authority remains the only live-refresh boundary.
- Egress workers have no database, registry-write, deployment or secret access.
- Models/vectors propose only and remain optional for correctness.
- Sponsorship, price and revenue cannot establish rank, trust or canon.
- Genesis remains public-source, read-only, browser-free and action-free.

## Commands executed

```bash
make test
go test -run='^$' -fuzz='^FuzzDecisionJSON$' -fuzztime=1s ./internal/admission
go test -run='^$' -fuzz='^FuzzRegistryJSON$' -fuzztime=1s ./internal/atlas
GOMAXPROCS=4 make test
make generate-e2 generate-e3 generate-e3-admission
git diff --exit-code -- generated/e2 generated/e3
make demo demo-e2 demo-e3 demo-e3-worker
make stress-e2 stress-e3-500
go test -race ./...
make vet
git diff --check
git diff --exit-code -- generated/e2 generated/e3
```

The read-only VPS command list is recorded separately in
`reports/e3-3-vps-capacity-baseline.md`.

## Validation results

- all Go package tests passed;
- all 14 one-second Go fuzz targets passed;
- observation C verifier accepted two valid vectors, rejected 14 invalid
  vectors and rejected corrupted evidence under Clang ASan/UBSan;
- shared E2 result/artifact Go/C conformance passed under Clang ASan/UBSan;
- observation and E2 C libFuzzer targets each completed 5,000 runs without a
  crash;
- offline end-to-end source statement, semantic view and provenance passed;
- race detector and `go vet` passed;
- deterministic E2, E3 metrics and 25-dossier admission generation reproduced
  the tracked artifacts with no diff;
- E1, E2, E3.1 and literal-loopback egress-worker demonstrations passed;
- E2 stress passed 100 deterministic invocations and five distinct proof
  bundles; HTTP load produced 20 successes and 30 expected rate limits at
  concurrency eight, 11.181 ms mean, 63.654 ms p95 and 20,300 KiB peak RSS;
- Atlas loopback stress passed 50,000 requests at 32 workers, 23,632.237
  requests/second, 3,253 microseconds p95 and 23,736 KiB peak RSS;
- Atlas stress preserved 500 origins, 25 prepared dossiers, 475 unprepared
  dossiers, 500 frontier decisions and zero frontier jobs;
- documentation configuration/navigation and whitespace checks passed;
- no public-origin network access occurred.

The first plain `make test` passed. Two later repetition attempts at the
host's default 16 fuzz workers each reported Go's `context deadline exceeded`
after a different one-second fuzz target completed its interval
(`FuzzDecisionJSON`, then `FuzzRegistryJSON`). Each affected target immediately
passed in isolation. The complete unchanged suite then passed with
`GOMAXPROCS=4`, preserving the same target set and fuzz duration while reducing
worker contention. No crash, mismatch or failing corpus input was produced.

These are preservation results for existing E1/E2/E3.2 behavior. They are not
E3.3 packet/query/database performance evidence.

## Unresolved risks and blockers

1. E3.2 is still a draft/unadmitted gate. E3.3 runtime work cannot begin until
   its exact base is admitted and merged.
2. The CDDL is a design contract. S1 must implement canonical codecs, two
   independent verifiers, valid/adversarial vectors and fuzzing before it is an
   executable protocol claim.
3. The relational SQL is a reviewed outline, not a migration, and was not
   executed because PostgreSQL is absent and deployment is blocked.
4. The VPS root array has a known single-device failure path, swap pressure is
   unexplained and unrelated shared workloads are not resource-bounded for
   coexistence.
5. No encrypted off-host backup, WAL archive or point-in-time restore evidence
   exists.
6. Exact ordinal tables for Pareto dimensions and all valid trust/delta
   transitions must be frozen in S1 conformance vectors.
7. Retention and disclosure policies for real public packets require founder
   review before corpus admission.
8. The funding-demo counts—100 policy decisions, 100 profiles, 50 observations,
   25 schemas, 12 adapters, eight live origins, 100,000 packets, three views
   and one stream—are acceptance targets, not achieved state.
9. The one-second Go fuzz harness is intermittently deadline-sensitive at 16
   workers on this development host. CI should use an explicit evidence-backed
   worker count or longer fuzz duration without dropping a target; the test was
   not weakened in this branch.

## Deviations

- The architecture pack requested ADR numbers 005 and 006. Those numbers are
  already occupied by admitted repository decisions, so this branch uses the
  next non-conflicting numbers, ADR 008 and ADR 009. Existing decision history
  was not overwritten.
- No E3.3 runtime implementation was started because the pack explicitly
  sequences it after E3.2 admission.
- PostgreSQL was not installed locally or on the VPS, and the SQL design was
  not executed, because storage/recovery gates fail.
- No E3.3 PR was opened or published in this design pass. The earlier published
  route-centric draft is not force-pushed or history-rewritten; a future clean
  PR should explicitly supersede it after founder review.
- Website and Mintlify content were left to Claude as directed.

## Next recommended gate

Founder review should accept or amend ADR 008, ADR 009, the canonical contract
surface and the deployment blocker. In parallel, a host administrator should
repair or formally replace the degraded storage design and establish off-host
recovery. After E3.2 is admitted and merged, begin **S1 only**: implement the
language-neutral packet/query/delta codecs, independent Go/C verification,
adversarial vectors and fuzz targets. Do not install PostgreSQL or begin
large-origin ingestion until S1 and the host/recovery gates pass.
