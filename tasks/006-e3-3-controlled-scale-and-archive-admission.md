# Task 006: E3.3 controlled scale proof and archive admission boundary

## Status

Controlled 25,000-packet capacity candidate implemented locally at
`c433ad33d5ba76f5ae0112fd3b58f89cf293e6ca`; evidence is recorded in
`reports/e3-3-controlled-scale-and-archive-readiness.md`. Archive retrieval and
public archive-packet admission remain blocked pending completed human policy
decisions and a separately reviewed importer.

## Objective

Exercise the immutable Semantic Snapshot compiler, segmented carriage,
proof reconciliation and read-only query runtime at the funding-demo packet
scale without fabricating public-origin, archive, mapping or freshness claims.

## Required outputs

- deterministic controlled representation with 25,000 source-native fields;
- observation-bound packet compilation with native terms, lexical values and
  exact extraction-plan digests;
- segmented packet batches and proof indexes within artifact bounds;
- explicit public-packet and fixture-packet counters;
- fixtures excluded from public views, queries, traces and HTTP defaults;
- an explicit loopback-only fixture flag for capacity testing;
- Go verification of every packet/proof relation and restricted-C sampling of
  canonical packet and snapshot objects;
- 5,000-request loopback stress evidence at the 25,000-packet scale;
- exact Meridian read-only capacity preflight without deployment or mutation;
- an honest report separating capacity proof from archive/public evidence.

## Archive admission prerequisites

The later Common Crawl importer must accept only sealed work orders bound to a
canonical Atlas origin, completed human policy decision evidence, permitted
representative routes, selected collection IDs and explicit request/byte
budgets. It must implement `deploy/snapshot/COMMON_CRAWL_IMPORT.md` and pass its
offline adversarial suite before any network execution.

## Exclusions

- no automatic Atlas or policy approval;
- no Common Crawl, live-origin or arbitrary-URL request in this gate;
- no controlled fixture counted as a public packet or archive profile;
- no semantic mapping or canon promotion from generated fixture data;
- no PostgreSQL, Object Storage, Storage Box or Meridian mutation;
- no browser, model, action, payment, authentication or public deployment;
- no merge or deployment without founder review.

## Acceptance

The scale snapshot is deterministic for exact inputs, fully reverified before
queries, segmented without exceeding artifact limits, returns an exact
source-native field only when fixtures are explicitly enabled, returns no
fixture result through public defaults, survives the bounded stress workload
with zero origin-network requests, and leaves all E1/E2/E3.2/S1 behavior
unchanged.
