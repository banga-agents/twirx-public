# ADR 015: Canonical corpus and segmented Universe Snapshot

Status: accepted design; implementation selection requires measured benchmark

Date: 2026-08-12

## Context

The E3 Semantic Snapshot runtime proves bounded immutable queries, but one rich
in-memory object per packet is not the intended million-packet edge design.
Meridian remains a stateless shared host under ADR 010 and cannot receive a
mutable database, compiler, broad corpus or ingestion worker.

## Decision

Separate the canonical corpus archive from the agent-query edge release:

```text
evidence CAS + canonical CBOR packet/frame batches
    -> partitioned analytical exports
        -> immutable Universe Snapshot
           dictionaries + frame segments + postings + hot views + proof refs
```

Agents query frames, materialized views and compact posting indexes. Packet
bodies load lazily for trace, proof, conflict inspection and native-source
comparison. Canonical external identifiers remain in dictionaries; compact
integer identifiers are snapshot-local and never become protocol identity.

Two bounded prototypes must be measured before the runtime selection is
frozen:

- a native immutable segment store using deterministic dictionaries, sorted
  postings and bounded reads;
- an isolated read-only analytical sidecar over immutable columnar artifacts.

The benchmark must compare deterministic build, proof linkage, dependency and
supply-chain surface, peak RSS, release bytes, query latency, cold start and
restore complexity. No sidecar is admitted to the trusted path without a
separate dependency ADR and founder approval.

## Edge limits

The ambitious E4 engineering targets are:

```text
1,000,000+ canonical packets in off-host object storage
100,000+ frames indexed at the edge
under 2 GiB edge release
under 512 MiB steady runtime RSS
p95 under 100 ms for curated structured queries
zero origin calls for materialized queries
```

These are targets, not present-tense claims. Actual generated metrics remain
separate. ADR 010's stricter host ceilings remain controlling.

## Publication

The Universe Snapshot extends the existing acyclic Semantic Snapshot topology.
All segments and indexes exist and verify before the self-digest-free manifest
is written last. Activation stages a new content-addressed release, validates
every constituent and fixed smoke query, then switches atomically. Meridian
holds no write credentials.

## Consequences

- The canonical proof corpus remains portable and independently verifiable.
- The edge can scale by lazy segment access without redefining packet/frame
  identity.
- A future mutable state store can import the same canonical artifacts.

## Rejected alternatives

- Load all packet objects into RAM: rejected as the default scale path.
- Install PostgreSQL on Meridian: rejected by ADR 010.
- Choose DuckDB, Parquet or another sidecar before measurement and approval:
  rejected because it adds an unjustified dependency and process boundary.
- Put only materialized answers in the release: rejected because trace and
  independent proof must remain available.
