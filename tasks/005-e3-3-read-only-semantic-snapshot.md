# Task 005: E3.3 read-only Semantic Snapshot demonstration

## Status

Local implementation candidate complete at
`aa16fbb3151964d6c62d90ed3b5596d0b3caba9a`; evidence is recorded in
`reports/e3-3-read-only-semantic-snapshot.md`. Merge, public activation, Object
Storage publication and Meridian changes require founder review.

## Objective

Produce a reproducible immutable snapshot from admitted E2 replay evidence and
serve bounded semantic query and packet trace operations without a mutable
database, source network access, browser, model or action capability.

## Required outputs

- deterministic offline builder with manifest-last publication;
- complete semantic reconciliation on import;
- 500 Atlas identities with maturity claims preserved;
- actual packets only from admitted E2 replay evidence;
- fixture exclusion from public counters and runtime defaults;
- two proof-linked materialized demo views;
- bounded CLI and literal-loopback HTTP runtime;
- fixed project, population and two-origin smoke queries;
- malformed, tamper, fixture-boundary, live-refresh and public-bind tests;
- fuzz coverage for new untrusted JSON parsers;
- query benchmark and bounded loopback stress workload with exact host and
  corpus scope;
- commit-bound evidence report with actual and target counts separated.

## Exclusions

- no PostgreSQL or mutable authoritative state;
- no Common Crawl or live-origin network retrieval;
- no semantic delta claim without two evidenced snapshots;
- no arbitrary URL, SQL, filesystem path or remote tool input;
- no browser, model, MCP execution, payment, authentication, write or action;
- no Object Storage, Storage Box, Meridian, DNS or deployment mutation;
- no merge or deployment without founder review.

## Acceptance

The same source revision and explicit build time produce the same snapshot ID;
the complete snapshot verifies offline; fixed public queries resolve without a
network request; a fixture is invisible by default; tampering fails closed;
the service rejects non-loopback binding; the full E1/E2/E3.2/S1 suite remains
unchanged; and the report states every missing funding-demo target.
