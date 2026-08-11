# ADR 009: Genesis semantic data stack

Status: accepted design; production-state admission amended by ADR 010

Date: 2026-08-11

## Context

The current content-addressed filesystem is appropriate for immutable
representations, canonical observations, proof bundles and conformance
artifacts. E3.3 also needs transactional packet admission, current heads,
relational ontology traversal, materialized cross-origin views, durable query
cursors, an outbox, and economic accounting. Those requirements should be met
without prematurely operating a distributed streaming, search or graph stack.

## Decision

The Genesis stack preserves the existing language-neutral contracts, Go
trusted implementation, restricted-C independent verification, deterministic
CBOR/CDDL, content-addressed evidence and Wasm adapter boundary.

It adds the following implementation components behind protocol boundaries:

| Concern | Genesis implementation |
| --- | --- |
| Authoritative operational state | PostgreSQL 18 |
| Exact and relational lookup | PostgreSQL B-tree and ordinary relational indexes |
| Native/lexical search | PostgreSQL full-text search, `pg_trgm`, and `unaccent` |
| Ontology traversal | versioned relational concept/edge tables and compiled closures |
| Optional vector candidates | `pgvector`, disabled by default and not required for correctness |
| Immutable packet and delta history | append-only time-partitioned PostgreSQL tables plus filesystem CAS |
| Current semantic state | transactional proof-linked head and materialization tables |
| Events | transactional outbox in the same commit as state changes |
| Public subscriptions | server-sent events with persisted delta cursors |
| Corpus exports | Apache Parquet, with Arrow-compatible schema |
| Local corpus analysis | DuckDB over exported Parquet, outside the trusted request path |
| Observability | OpenTelemetry-compatible metrics and traces |
| Public edge | Caddy |

These are implementation selections, not normative protocol requirements.
An independent implementation may use a different store if it produces and
verifies the same canonical artifacts and conformance behavior.

`pgvector` may retrieve candidates only. Vector similarity cannot establish a
canonical concept, publisher identity, origin authority, access permission,
mapping trust or execution authorization.

NATS JetStream is deferred until measured outbox lag, independent nodes,
regional consumers or subscription fan-out demonstrate a need. Kafka,
Kubernetes, Elasticsearch, a dedicated graph database and ClickHouse are not
part of Genesis.

## Transaction and immutability rules

- Canonical bytes enter the filesystem CAS before a packet admission
  transaction references them.
- A packet identity table provides global digest uniqueness independently of
  time partitioning.
- Packet and delta log roles receive `INSERT` and `SELECT`, never `UPDATE` or
  `DELETE`; schema ownership belongs to a separate migration role.
- A state transition, materialized head change, delta and outbox row commit in
  one transaction.
- Consumers acknowledge an outbox event only after their durable side effect.
- Materialized state can be dropped and rebuilt from immutable artifacts and
  admitted log order.
- All ordering keys and planner costs are deterministic integers. Canonical
  output never depends on database row, map or floating-point iteration order.

## Host and network boundary

PostgreSQL listens only on a Unix socket and loopback. It is never exposed by
Caddy or the host firewall. The API/compiler receives a least-privilege
database role. The E3.2 egress worker receives no database credentials, no
registry write access, no deployment access and no secret access.

The initial resource profile is deliberately bounded and must be tuned from
measurements rather than host capacity alone:

```text
max_connections       40
shared_buffers         1 GiB
work_mem               8 MiB
maintenance_work_mem   256 MiB
temp_file_limit        1 GiB per session
statement_timeout      role-specific and bounded
lock_timeout           role-specific and bounded
idle transaction limit enabled
PostgreSQL service memory ceiling: 4 GiB initial
```

The semantic store begins with a 50 GiB soft allocation and a 75 GiB hard
operational alarm. Evidence retention remains governed separately by the CAS
policy. These are initial operating limits, not performance claims.

## Activation gates

The measured VPS currently has a degraded root RAID1 array with only one
active member, nearly exhausted swap, no PostgreSQL installation, and a shared
Docker workload. Therefore PostgreSQL installation and data admission are
blocked until all of the following are evidenced:

1. the root storage mirror is healthy or the founder accepts and documents a
   replacement durability design;
2. swap pressure and competing workload ownership are understood;
3. encrypted off-host base backup and continuous WAL archiving are configured;
4. a point-in-time restore drill succeeds on an isolated target;
5. roles, filesystem ownership, limits, loopback binding and firewall state
   pass target-host verification;
6. S1 contracts and adversarial conformance are admitted;
7. founder review explicitly authorizes activation.

Backups stored only on the same VPS or second local device do not satisfy the
off-host recovery requirement.

## Consequences

- Genesis gets transactional joins, search, graph closure, current state and
  replayable events on one operational system.
- Protocol objects remain portable and independently verifiable outside
  PostgreSQL.
- Optional analytics and model tooling cannot enter the trusted query path by
  default.
- The degraded storage condition is a release blocker rather than a hidden
  operational risk.

## Rejected alternatives

- PostgreSQL as the only evidence store: rejected because canonical source
  representations and bundles remain content-addressed artifacts.
- A separate message broker immediately: rejected until measured demand
  exceeds the transactional outbox.
- A mandatory vector database: rejected because exact, lexical and graph
  retrieval must remain sufficient for deterministic operation.
- Installing PostgreSQL before recovery design: rejected because it would put
  authoritative operational state on a known single-device failure path.
