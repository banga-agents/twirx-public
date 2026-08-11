# ADR 011: Reconcile the stranded PR 15 resolver contracts

Status: accepted for the FUTO readiness integration candidate

Date: 2026-08-11

## Context

Pull request 15 (`agent/e3-3-semantic-contracts`) was merged into
`agent/e3-2-future-compatible-origin-model` after that branch had already been
merged into `main`. Its two commits, `09c6031` and `c10ce26`, therefore never
entered the default branch. The work contains useful security and product
constraints, but its route-centric JSON request and scoring objects predate
ADR 008 and the independently verified Semantic Data Plane S1 contracts.

Importing PR 15 unchanged would create two purportedly normative request
families and two incompatible selection policies:

- PR 15 defines `tw.semantic-request/0.1` as JSON and selects a route through a
  hidden weighted scalar score;
- S1 defines `tw.semantic-query/0.1` as deterministic CBOR and exposes bounded
  ontology, trust, freshness, proof, economics, conflict and caller-preference
  dimensions without making route selection the primary product.

The FUTO readiness release requires one authoritative packet/query family and
does not authorize a new resolver runtime.

## Decision

The language-neutral Semantic Data Plane 0.1 family is the sole authoritative
packet/query/delta/snapshot family for this release:

```text
schemas/cddl/semantic-data-plane.cddl
spec/data-plane/
conformance/data-plane/
internal/dataplane/
```

The canonical query object is `tw.semantic-query/0.1`. The canonical query
result binds a detached plan digest, but this release does not admit a
standalone Access Plan wire object or a route-selection runtime.

PR 15 is recovered through the matrix below. Useful invariants are retained in
the current ADRs, threat model, Atlas vocabulary, query contract and migration
inventory. Superseded JSON schemas and the fixed weighted ranking policy are
not copied into the integrated tree. This avoids presenting unverified JSON as
a second canonical protocol.

## Reconciliation matrix

| PR 15 area | Disposition | Authoritative owner or rationale |
| --- | --- | --- |
| Semantic Request JSON schema and vectors | **Supersede** | `semantic-query-core` in `schemas/cddl/semantic-data-plane.cddl`, with shared Go/C vectors, is the sole canonical query contract. Natural-language input remains proposal-only. |
| Access Plan JSON schema and vectors | **Port concept only** | `semantic-query-result-core.plan-digest`, `QUERY_DELTA_FABRIC_0_1.md`, and structured trace/explanation reserve proof-linked planning without admitting a second wire object. |
| Origin Capability JSON schema and vector | **Supersede** | `tw.origin-registry/0.3` owns orthogonal interface, capability, effect, access and trust state; semantic packets may describe admitted capability state without creating route authority. |
| Commercial Offer JSON schema and vectors | **Supersede** | Atlas evidence-bound provisional offers plus `packet-kind = offer` and query economics preserve source-stated terms. No offer, price or payment object becomes execution authority. |
| Route-provider lifecycle | **Retain as future interface design** | The exact E2 contract and E3.2 sealed work order remain the only execution authorities. No provider implementation is admitted by FUTO readiness. |
| Compact seven-operation meta-interface | **Port principle** | ADR 008 owns the compact `twirx.query/subscribe/compare/trace/explain/resolve/invoke` surface. The public snapshot release exposes only implemented read-only operations. |
| Hard-filter-first policy | **Retain** | ADR 008, `TRUST_AND_RANKING_TABLES_0_1.md`, the canonical query constraints and the root threat model require hard filters before preference. |
| Fixed weighted scalar ranking | **Reject** | ADR 008 deliberately uses a visible Pareto frontier and explicit caller policy. One hidden aggregate score would obscure incomparable authority, proof, freshness, cost and uncertainty. |
| Stable deterministic tie behavior | **Port invariant** | Canonical byte order, explicit integer ordinals and deterministic query/result encoding remain required; database or map iteration order cannot decide results. |
| Non-pay-to-rank invariant | **Retain** | ADR 008 and `THREAT_MODEL.md` prohibit sponsorship, revenue or offer presence from altering trust, authority, semantic rank or canon admission. |
| Natural-language proposal boundary | **Retain** | Natural language may propose a typed query; it cannot authorize execution, network access, payments, mappings or canon changes. |
| Resolver threat model | **Port controls** | The root threat model now owns prompt/content injection, endpoint substitution, semantic confusion, economic capture, effect laundering, SSRF, replay, privacy and resource-exhaustion controls. |
| Benchmark and VPS plan | **Supersede** | `spec/data-plane/BENCHMARK_0_1.md`, snapshot stress evidence, ADRs 009/010 and the measured Meridian report own current capacity and deployment limits. |
| Atlas 0.3 migration plan | **Supersede** | `reports/e3-3-migration-inventory.md` preserves exact E1/E2/E3.2 identities and defines fail-closed packet migration without inventing approvals. |
| Resolver documentation page and task | **Supersede** | `spec/data-plane/`, `tasks/004-e3-3-semantic-data-plane.md`, and generated public evidence describe the accepted architecture and actual implementation status. |
| PR 15 evidence report | **Retain as historical Git evidence** | Commit `c10ce26` proves the stranded preparation branch was reviewed and tested; it is not evidence that those JSON objects are current protocol authority. |

## File-level disposition

| PR 15 paths | Disposition |
| --- | --- |
| `schemas/json/semantic-request.schema.json`, `conformance/resolver/*semantic-request*` | superseded by the S1 semantic query contract and conformance |
| `schemas/json/access-plan.schema.json`, `conformance/resolver/*access-plan*` | not ported; plan identity remains detached and no canonical plan object is admitted |
| `schemas/json/origin-capability.schema.json`, matching vector | superseded by the Atlas 0.3 registry and packet capability kind |
| `schemas/json/commercial-offer.schema.json`, matching vectors | superseded by Atlas offer candidates, packet offer kind and economic query bounds |
| `spec/resolver/RANKING_AND_EXPLANATION_0_1.md` | hard-filter, explanation and non-pay principles retained; scalar weights rejected |
| `spec/resolver/ROUTE_PROVIDER_0_1.md`, `META_INTERFACE_0_1.md` | future design only; compact-surface principle retained in ADR 008 |
| `spec/resolver/THREAT_MODEL_0_1.md` | controls ported to the root threat model through the Semantic Data Plane work |
| `spec/resolver/ATLAS_MIGRATION_0_1.md` | superseded by `reports/e3-3-migration-inventory.md` |
| `spec/resolver/BENCHMARK_AND_CAPACITY_PLAN_0_1.md` | superseded by the data-plane benchmark and measured snapshot reports |
| PR 15 changes to `README.md`, `ARCHITECTURE.md`, `THREAT_MODEL.md`, docs navigation and Task 003 | superseded by later Semantic Data Plane and snapshot changes on the integration branch |
| `reports/e3-3-semantic-resolver-preparation.md` | historical branch evidence only; this ADR records its disposition in the integrated history |

## Compatibility and security effects

- No E1, E2 or E3.2 implementation behavior changes.
- No new runtime, dependency, network route, endpoint or execution authority is
  introduced.
- Existing Atlas capability and offer records remain descriptive unless their
  independent admission dimensions permit otherwise.
- The snapshot runtime remains read-only and model-, browser-, payment- and
  arbitrary-URL-free.
- A future Access Plan or route-provider canonical object requires a new ADR,
  a language-neutral specification, shared independent conformance vectors and
  founder admission.

## Alternatives rejected

- Cherry-pick PR 15 unchanged: rejected because it creates competing normative
  query schemas and restores route selection as the center of the product.
- Retain the JSON schemas as an equal transport: rejected because no normative
  JSON-to-canonical-CBOR mapping or independent verifier exists.
- Delete all PR 15 ideas: rejected because its hard filters, compact interface,
  non-pay-to-rank rule, proposal boundary and threat analysis remain valuable
  and are already represented in the accepted architecture.
- Implement the resolver during FUTO readiness: rejected because the release
  train is limited to real evidence, a genuine delta and a read-only public
  snapshot proof.

## Reversal conditions

A future gate may standardize a distinct Access Plan or JSON representation
only after specifying its relationship to the canonical query/result family,
adding independent conformance, proving deterministic identity and showing a
need not met by packet trace and explanation artifacts.
