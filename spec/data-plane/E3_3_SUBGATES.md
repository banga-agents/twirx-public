# Engineering Gate E3.3 — Semantic Data Plane Alpha

**Authority:** Normative gate sequence

**Status:** S1 implementation candidate complete; founder admission pending

E3.3 is divided into independently reviewable subgates. Passing a subgate does
not authorize a later one. Each final commit receives complete offline
validation, security evidence, exact artifact counts and founder review before
merge or deployment.

ADR 010 also permits a local read-only Semantic Snapshot demonstration before
the mutable S2 store. That demonstration may exercise narrow portions of S3,
S5 and S6 over admitted replay evidence, but it does not pass or reorder those
subgates. Its profile is `READ_ONLY_SNAPSHOT_RUNTIME_0_1.md`.

## Entry conditions

- E3.2 is admitted and merged without weakening its disabled scheduler, human
  review, evidence-before-parsing, sealed work orders or network isolation.
- ADR 008 and ADR 009 are accepted.
- The exact E3.2 base is recorded; no published branch is history-rewritten.
- Website/documentation work is preserved independently and cannot become
  technical evidence by presentation alone.

## S1 — Packet, query and delta contracts

Implementation candidate: the language-neutral contracts, deterministic-CBOR
Go implementation, independent restricted-C verifier, shared conformance
corpus and offline snapshot-directory verification are complete on the S1
review branch. This statement is not S1 admission and does not authorize S2,
network ingestion, database installation or deployment.

Deliver:

- normative CDDL and prose for packet core, batch manifest, delta, query,
  subscription, result, materialization manifest and economic event;
- deterministic canonical encoding profile;
- valid, boundary, malformed and adversarial vectors;
- primary and independent restricted-C verification;
- detached digest and manifest-last publication tests;
- exact delta-class and trust-lane transition tables.

Accept when Go and C accept/reject the same complete vector corpus under GCC,
Clang, ASan and UBSan; normal tests remain offline; fuzz targets cover every
new parser; native terms and lexical values survive round trip; and no database
or live-network dependency is required.

## S2 — PostgreSQL state store

Deliver:

- ordered reversible migrations from the reviewed design;
- explicit monthly partitions and global digest/cursor identities;
- least-privilege roles and append-only runtime grants;
- transactional packet/head/delta/outbox admission;
- ontology, closure, materialization, query, subscription and economics tables;
- operational limits, observability and schema-integrity tests;
- encrypted off-host backup, WAL archive and isolated PITR drill evidence.

Accept only after the VPS degraded-storage and swap blockers are resolved, no
public 5432 listener exists, the E3.2 worker cannot reach the database, killed
transactions leave no partial state, immutable mutation probes fail, a clean
database and restored database pass identical tests, and materializations
rebuild to identical digests.

## S3 — Deterministic packet compiler

Deliver:

- E1/E2 artifact migration dry run and reconciliation;
- deterministic native-statement-to-packet compiler;
- explicit observed/provisional/attested lanes;
- atomic manifest-last batches with rejection artifacts;
- idempotent CAS and state-store admission;
- initial public-source retention/disclosure enforcement.

Accept when the TWIRX, World Bank and controlled-fixture reconciliation corpus
produces byte-identical packets across repeated runs, optional absence remains
explicitly unresolved, required evidence fails closed, fixture scope cannot
enter public counters, and no compiler component has network access.

## S4 — Delta engine

Deliver:

- deterministic semantic-key construction;
- origin, semantic and canon delta classifiers;
- immutable current-head transitions and replay;
- lifecycle/retraction handling;
- gap, replay, reordering and concurrent-admission tests.

Accept when a source change, mapping-only change and canon-only change produce
three different exact delta classes; replay rebuilds identical heads; equal
times use the specified deterministic tie-break; and no reinterpretation is
reported as a publisher change.

## S5 — Materialized semantic views

Deliver at least three founder-admitted cross-origin definitions:

- latest country indicators;
- recent research publications;
- current public commercial capabilities/offers.

Every view binds definition, canon, packet set, cursor, conflict policy and
output digest. Accept when incremental maintenance and full rebuild agree,
stale/retracted/fixture data obeys the definition, disagreement is preserved,
and an observed offer cannot trigger payment or claim availability beyond its
evidence/freshness.

## S6 — Query and subscription fabric

Deliver:

- `twirx.query`, `twirx.compare`, `twirx.trace`, `twirx.explain`,
  `twirx.resolve`, `twirx.invoke` and `twirx.subscribe` behind one compact
  typed interface;
- exact/lexical/bounded-graph retrieval;
- hard filters, Pareto frontier and explicit deterministic preference policy;
- materialized-state planning plus bounded E3.2 live-refresh subplans;
- SSE and polling with persisted resumable cursors;
- complete plan/proof/conflict explanations.

Accept when deterministic no-model/no-vector operation passes; natural
language cannot execute directly; sponsorship cannot alter rank; arbitrary
URLs/MCP/browser/actions remain impossible; restart/reconnect loses no
committed delta; and cursor expiry/gaps are explicit.

## S7 — Corpus and ingestion proof

Deliver evidence toward the funding-demonstration floors:

```text
500 cataloged origins
100 completed policy decisions
100 safely profiled origins
50 immutable observed origins
25 native schemas
12 deterministic adapters
8 live read-only origins
100,000+ admitted semantic packets
```

Counts are orthogonal and exact. This subgate uses only founder/human-admitted
origins, explicit work orders and bounded budgets. Common Crawl/archive inputs
remain historical evidence. Accept only the achieved counts; never convert a
target into a claim. No bulk retrieval begins before S1-S6 and target-host
security pass.

## S8 — Public semantic change stream

Deliver one read-only public stream over an admitted view, with public schema,
proof links, retained-cursor policy, rate limits, emergency suspension and
static replay evidence. Accept when Caddy exposes only the intended endpoint,
restart/reconnect behavior is proven, private/control-plane metadata is absent,
and disabling delivery does not corrupt the immutable log.

## S9 — Performance and value benchmarks

Deliver the frozen workload and all metrics in `BENCHMARK_0_1.md`, including
materialized/live/native routes and controlled browser/model comparisons where
applicable. Accept when raw evidence, exact commands, commit, corpus, host,
configuration, failures and exclusions are published; cold/warm/steady results
are separated; and no universal multiplier is inferred from one fixture.

## S10 — Economic telemetry

Deliver proof-linked work, resource, review, cost, funding and revenue events;
effective semantic cost per verified invocation; storage/maintenance
projections; and a public aggregate with privacy thresholds. Accept when
measured and estimated values are distinct, money uses explicit currency and
method version, sponsor/payment fields cannot change semantic rank, and
operator economics do not masquerade as publisher prices.

## E3.3 final admission

E3.3 passes only when S1-S10 each have a commit-bound report, all earlier
engineering gates still pass, the exact public counts are regenerated from
canonical artifacts, the VPS recovery/security evidence is current, and the
founder reviews the complete system. Public Alpha language must state achieved
scope, not the larger future Web-scale goal.

## Explicit exclusions

- arbitrary URL submission or public crawling;
- automatic policy/publisher/canon/mapping approval;
- browser execution;
- production LLM execution authority;
- authenticated/private origins;
- writes, payments, purchases, bookings or other material actions;
- arbitrary remote MCP servers;
- mandatory vector retrieval;
- distributed stream/federation before measured need;
- merge or deployment without founder review.
