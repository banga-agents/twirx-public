# Genesis state store 0.1

**Authority:** Normative state invariants; PostgreSQL layout is non-normative

**Status:** Design only; deployment blocked by ADR 009

## Separation of authority

The protocol requires immutable canonical artifacts, deterministic admission,
globally unique identities, ordered deltas, rebuildable current state and
durable subscriptions. It does not require PostgreSQL.

The Genesis implementation separates:

```text
CAS                         exact canonical bytes
packet/delta log            immutable admission and ordered change
current heads               rebuildable current projection
ontology/canon              versioned concepts, edges, mappings and closures
materializations            rebuildable cross-origin views
transactional outbox        durable delivery intent
economic log                measured work/cost/revenue without rank authority
```

JSON or relational columns used for search are projections. They cannot be
used to regenerate canonical bytes unless a conformance-tested canonical
decoder/encoder independently reproduces the original digest.

## Global identity and partitioning

PostgreSQL unique constraints on a partitioned table must include every
partition key. E3.3 therefore does not pretend that a partition-local digest
constraint is global.

Each immutable class has a small non-partitioned identity table:

```text
packet_digest -> observed_at + canonical size
delta_digest  -> global cursor + occurred_at + canonical size
```

The time-partitioned log references that identity and uses a composite primary
key including time. This yields global digest uniqueness and a globally
monotonic durable delta cursor while preserving bounded partition maintenance.

Monthly partitions are created before their valid interval. A missing
partition fails admission; there is no catch-all default partition that can
grow silently. A partition is detached or dropped only by the migration owner
after retention and proof availability checks.

## Append-only and transactional behavior

Runtime roles cannot update or delete packet/delta identities or log rows.
One admission transaction locks semantic keys in their canonical byte order
and atomically writes:

- missing immutable identities and log rows;
- a new current head when deterministic ordering selects it;
- the exact classified delta;
- affected materialized-state cursor/work record;
- the economic measurement event where present;
- an outbox event.

Any error rolls the transaction back. A manifest is already complete and
verified in the CAS before the transaction begins. Database commit does not
make a partial filesystem publication valid.

## Index families

The Genesis schema supports coordinated, non-authoritative indexes:

| Index | Purpose |
| --- | --- |
| packet/semantic digest | exact proof and identity lookup |
| origin + observation time | origin history and refresh processing |
| native term trigram/text | source-native lexical retrieval |
| canonical concept/subject/predicate | exact semantic lookup |
| concept/edge/closure | bounded ontology expansion |
| semantic key/current head | current state |
| valid/observed time | temporal filtering |
| capability/effect/access | admitted execution candidate lookup |
| delta sequence | durable subscription replay |
| packet/derivation digests | trace and proof lookup |

Vector retrieval is absent from the required schema. If later enabled, it
stores only a candidate feature linked to an immutable source and model
version. Exact operation without that index remains a conformance requirement.

## Ontology storage

Canon modules, concepts, labels, edges and compiled closures are versioned and
append-only at the module-version boundary. An edge contains an integer path
cost and mapping/trust status. Floating-point weights are prohibited in
deterministic planning.

A closure row binds its producing module-set digest and path digest. A canon
update writes a new module/closure version and emits a canon delta; it never
changes the packet that carried an older mapping.

## Current heads and materializations

A current head contains a semantic-key digest, current and prior packet
identity, update delta, and cursor. It can be recreated from admitted history.

A materialization has:

- canonical definition and definition digest;
- exact canon/module set;
- trust, source, freshness and conflict policy;
- through-sequence cursor;
- manifest binding the complete packet contribution set;
- normalized rows plus a row-to-packet join.

Rebuild admission requires the same manifest/result digest from the same log
and canon inputs. A mismatch suspends the materialization rather than silently
replacing it.

## Outbox and subscriptions

The outbox row stores a topic, stable key, canonical payload digest and delta
cursor. A dispatcher lease can expire and be reclaimed. Delivery may be at
least once; client-visible event identity and cursor make duplicates safe.

`LISTEN/NOTIFY` is an optional wake-up signal only. The persisted row is the
delivery authority. Deleting delivered rows follows a retention checkpoint
that preserves the public earliest-resumable cursor.

## Database failure behavior

- missing partition: reject admission;
- digest with unequal size/bytes: integrity incident and reject;
- unknown origin/policy/contract/canon identity: reject;
- invalid trust transition: reject;
- materialization digest mismatch: suspend view and serve explicit unavailable;
- cursor gap: stop subscription delivery and expose the gap;
- CAS unavailable: fail proof-required reads and all writes;
- replica/read snapshot lag: expose snapshot/cursor and enforce freshness;
- database unavailable: existing static proof artifacts remain verifiable, but
  current query/subscription service reports unavailable rather than stale as
  current.

## Genesis relational design

`GENESIS_RELATIONAL_SCHEMA.sql` is a reviewed design artifact, not an
executable migration. S2 must split it into ordered, reversible migrations,
create explicit partitions, apply role grants and verify a fresh database plus
restore path before it becomes operational.
