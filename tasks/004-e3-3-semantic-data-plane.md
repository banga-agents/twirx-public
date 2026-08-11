# Task 004: Engineering Gate E3.3 — Semantic Data Plane Alpha

## Status

S1 implementation candidate complete on the parent branch. E3.2 is admitted
on `origin/main` at merge commit `669b205`. The founder has authorized the
local-only immutable snapshot demonstration described by Task 005. S2 has not
started. Production database and public deployment remain blocked by ADR 009,
ADR 010 and `reports/e3-3-vps-capacity-baseline.md`.

## Objective

Transform admitted public Web representations into immutable, proof-bearing
semantic packets; maintain versioned semantic state and classified deltas; and
serve typed query, comparison, trace, explanation and subscription operations
without re-browsing or model interpretation on ordinary compiled-state reads.

## Read first

1. `MANIFESTO.md`
2. `CHARTER.md`
3. `THREAT_MODEL.md`
4. `decisions/008-semantic-data-plane.md`
5. `decisions/009-genesis-data-stack.md`
6. `decisions/010-meridian-stateless-snapshot-edge.md`
7. `decisions/011-pr15-resolver-reconciliation.md`
8. `spec/data-plane/README.md`
9. `spec/data-plane/SEMANTIC_SNAPSHOT_0_1.md`
10. `spec/data-plane/E3_3_SUBGATES.md`
11. `reports/e3-3-migration-inventory.md`
12. `reports/e3-3-vps-capacity-baseline.md`
13. `reports/e3-3-s1-snapshot-preimplementation.md`

## Work sequence

Implement S1 through S10 in separate reviewable changes. Do not begin a
subgate merely because its interfaces are described. The prior subgate's
conformance, security and evidence report must be admitted first.

ADR 010 adds the Semantic Snapshot export/import contract to S1. Local packet,
delta, query, result, materialization and snapshot conformance may proceed.
PostgreSQL is local-development-only until S2 and remains forbidden on
Meridian. Common Crawl network ingestion and public snapshot activation are
later, separately admitted operations.

The normative contracts are language-neutral. PostgreSQL and Go are Genesis
implementation choices only. Preserve the restricted-C verifier's independence
and offline boundary.

ADR 011 records the file-level recovery of the stranded PR 15 resolver work.
For this release, `tw.semantic-query/0.1` is the sole authoritative query
contract. The earlier JSON Semantic Request, Access Plan, capability and offer
schemas are not parallel wire authorities and no resolver runtime is admitted.

## Non-negotiable invariants

- Preserve all E1/E2/E3.2 behavior and complete local conformance.
- Evidence is stored and verified before parsing or compilation.
- Native subject, term, locator and lexical value precede semantic mapping.
- Packet/delta/result cores contain no self-digest; manifests publish last.
- Missing required evidence fails closed; optional absence is explicit.
- Origin, semantic and canon deltas remain distinct.
- Current/materialized state is rebuildable and carries proof references.
- Natural language and models may propose but cannot execute or promote.
- No sponsor, offer or payment affects source authority, semantic rank or canon.
- No arbitrary URL, arbitrary MCP, browser, authentication, write or action.
- The E3.2 egress worker receives no database, registry-write or secret access.
- Normal tests use no public internet.
- No merge, deployment or public performance claim without founder review.

## Required completion evidence per subgate

- exact final commit SHA and base;
- files changed and exact diff summary;
- normative behavior and preserved invariants;
- complete commands and results;
- valid, malformed, adversarial, fuzz and independent-verifier evidence as
  applicable;
- benchmark/economic raw artifacts for any claim;
- target-host changes and rollback, if separately authorized;
- unresolved risks, deviations and next recommended subgate.

## Funding-demonstration acceptance target

The full target is 500 cataloged origins, 100 completed policy decisions, 100
safely profiled, 50 observed, 25 native schemas, 12 deterministic adapters,
eight live public-read origins, at least 100,000 admitted packets, three
materialized views and one public semantic delta stream. These are target
floors, never present-tense claims until canonical evidence derives them.
