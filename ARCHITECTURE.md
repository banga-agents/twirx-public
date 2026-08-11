# Genesis Architecture Baseline 0.3

## Mission boundary

Typed Web continuously compiles admitted origin representations into reusable,
typed and proof-bearing semantic state while preserving source-native meaning
and complete derivation evidence.

The implemented trust spine proves bounded read-only paths. The E3.3 design
adds immutable semantic packets, materialized state and deltas; it is not an
arbitrary scraper, truth oracle, browser replacement, model or payment system.

## Foundational invariants

1. **Source-statement fidelity** — result claims are bounded to observed origin representations and declared derivations.
2. **Native meaning permanence** — semantic normalization is additive.
3. **Language sovereignty** — specifications and conformance define the protocol.
4. **Local-first operation** — the complete proof runs without cloud services.
5. **Open extension, governed admission** — publication does not imply trust or canon status.
6. **Closed operational contracts** — executable operations are finite, typed, bounded, and effect-classified.
7. **Zero self-promotion** — untrusted components produce candidates only.
8. **Visible failure** — missing required evidence fails; optional absence becomes explicit.

## Genesis components

| Component | Implementation | Trust role |
|---|---|---|
| CLI and control path | Go | Orchestrates bounded local operations |
| Safe fetch | Go standard library | Produces untrusted response bytes under policy |
| Evidence store | Filesystem CAS | Preserves immutable body bytes by SHA-256 |
| Observation envelope | Deterministic CBOR | Binds retrieval metadata to evidence digest |
| Primary verifier | Go | Validates envelope and CAS body |
| Independent verifier | Restricted C | Independently validates canonical envelope and evidence |
| Adapter runtime | Deterministic Go JSON extraction | Creates source statements and semantic views offline |
| Controlled origin | Go test server | Reproducible fixture, not a production source |
| Documentation | Mintlify MDX | Technical canon and contributor guidance |
| Public site | Static HTML/CSS | Minimal technical project interface |

## E2 additions

Engineering Gate E2 adds a bounded, catalog-only Live Provenance Lab. It uses
one canonical operation contract to generate CLI, JSON Schema, OpenAPI, and a
local MCP binding. It publishes manifest-last proof bundles and extends
independent verification to typed results. It does not add arbitrary-URL
ingestion, a browser, a model, write operations, or public remote MCP.

## E3.0 and E3.1 additions

Genesis Atlas 500 begins with an offline control plane. The exact 500-origin
universe is a candidate selection, not a network allowlist. The canonical
origin registry reports catalog, policy, technical, publisher, health,
adapter-trust, and mapping-trust state independently. Deterministic metrics
cannot promote evidence from one dimension into another. The E3 process has no
HTTP client and its read-only API binds only to loopback.

E3.1 adds a separate policy artifact, exact digest binding from canonical
records, an offline RFC 9309 evaluator, and a deterministic dry-run frontier.
The frontier emits origin IDs and budgets rather than URLs and always declares
network access disabled. It is evidence for scheduling logic, not egress
authority.

The frontier iterates the exact candidate selection, not only the canonical
registry subset. Every one of the 500 candidates receives a deterministic
blocked, deferred, or scheduled outcome. A derived admission work queue
similarly distinguishes prepared dossiers from candidates that still require
dossier preparation. Neither artifact grants network or canonical-write
authority.

TWIRX and the World Bank E2 origin are cataloged with E2 technical evidence,
but their Atlas policy reviews remain `pending` and `uncertain`. TWIRX has an
independent `publisher_approved` state. Both schedulers remain disabled.
Separately, an
Observatory command proves process-separated, evidence-first retrieval against
one literal-loopback robots fixture. It writes the body and observation before
parsing and can replay the result offline. The fixture does not satisfy a live
origin's robots review and cannot update any registry state.

```text
exact candidate selection
      ↓ identity-preserving catalog review
canonical origin registry + separately bound policy artifacts
      ↓ independent evidence for each state dimension
orthogonal public metrics, bounded discovery API, and dry-run frontier
```

The Atlas-500 runtime workload drives all 500 identities through the actual
loopback HTTP surface under bounded concurrency, validates all frontier and
admission outcomes, and verifies restart recovery. It measures the control
plane at full catalog breadth; it does not replace the policy, profile,
observation, schema, adapter, semantic, or live-origin depth gates.

E3.2 future-compatibility metadata records interfaces, candidate capabilities,
effect classes, access economics, provisional offers, and publisher-readiness
signals without adding a resolver or any new execution route. The only
admitted capability records reproduce four existing E2 public-read operations.
All browser, WebMCP, authenticated, write, financial, legal/material, and
destructive routes remain descriptive and non-executable.

Continuous observation, compilation, semantic admission, and agent execution
remain later E3 subgates. Candidate selection never becomes an egress
allowlist.

## E3.3 semantic data plane design

ADR 008 changes the primary product topology without replacing the earlier
trust boundaries:

```text
admitted public representation
      ↓ evidence-before-parsing observation
deterministic continuous compiler
      ↓ native statement + optional reviewed mapping
immutable semantic packet log
      ↓ classified origin / semantic / canon deltas
proof-linked current state and materialized views
      ↓ exact + lexical + bounded ontology planning
typed query, comparison, trace, explanation and subscription fabric
```

A canonical packet binds source-native subject, predicate, locator and lexical
value, typed value, optional semantic mapping, evidence, derivation, trust lane,
time, freshness, lifecycle, retention and disclosure. It does not contain its
own digest and does not claim objective truth. Manifests publish last.

Origin deltas mean the observed provider representation changed. Semantic
deltas mean TWIRX interpretation changed. Canon deltas mean ontology or mapping
modules changed. The three cannot be collapsed into one generic update.

Materialized state is a rebuildable cache over immutable history. The planner
applies exact, native lexical and bounded graph retrieval; hard source, trust,
freshness, cost, effect and proof constraints; a visible Pareto frontier; and
an explicit caller preference. Optional model/vector candidates cannot grant
identity, policy, trust, rank or canon authority.

ADR 009 selects PostgreSQL 18, relational ontology closures, full-text/trigram
search, a transactional outbox, SSE cursors, and Parquet/DuckDB exports for the
Genesis implementation. These are non-normative implementation choices.
Database activation is blocked until E3.2 admission, S1 conformance, VPS
storage remediation and an isolated point-in-time recovery drill pass.

The S1-S10 sequence in `spec/data-plane/E3_3_SUBGATES.md` keeps contract,
storage, compiler, delta, view, query, corpus, public-stream, benchmark and
economic evidence independently reviewable.

## Deferred or gated components

- PostgreSQL semantic state store (designed; activation blocked);
- S3-compatible CAS;
- canonical adapter packages and signatures;
- Wasm/TWABI adapter isolation;
- semantic packet/module compiler (E3.3 S3);
- a general-purpose Typed Web IR compiler;
- WebMCP and general SDK generation;
- browser discovery workers;
- learning-ledger exports (E3.3 S7) and semantic induction model (later research branch);
- publisher verification and federation;
- action mandates, receipts, and settlement;
- chain anchoring.

## Gate namespaces and sequence

Engineering gates (`E`) have executable acceptance evidence. Public-foundation
milestones (`P`) cover documentation and release preparation. ADR 002 defines
the distinction.

```text
E1  Source-Statement Evidence Spine
P1  Public Foundation
E2  Live Provenance Lab
E3  Genesis Atlas 500
E4  Deterministic Compiler Alpha
E5  Hardened Invite Alpha
E6  Browser Discovery
E7  Publisher-Native Interfaces
E8  Federated Registry
E9  Safe Typed Actions
```

A later gate may not weaken an earlier invariant without a public protocol
decision and migration. `Public Alpha` is not earned until E3 passes.
