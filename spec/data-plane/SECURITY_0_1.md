# Semantic data plane security model 0.1

**Authority:** Normative E3.3 security requirements

**Status:** Design; implementation and target-host evidence pending

This document extends `THREAT_MODEL.md` and does not weaken E1, E2 or E3.2.
The semantic highway adds durable interpretation, cross-origin joins and
subscriptions; each increases the cost of admitting a false, stale or
misclassified statement. Evidence and admission remain prior to convenience.

## Trust boundaries

```text
untrusted public representation
  -> E3.2 sealed work-order retrieval
  -> immutable evidence spool/CAS
  -> bounded deterministic parser/adapter
  -> candidate packet batch
  -> independent verification + human/canon policy
  -> append-only admitted log
  -> rebuildable materialized state
  -> bounded query/subscription output
```

The egress worker cannot reach the database or canonical-control credentials.
The parser cannot access the network. The query fabric cannot mint work orders,
publisher identity, policy decisions, mapping trust or canon admission. The
public edge cannot submit SQL or filesystem paths.

## Protected properties

- exact preservation of native source term, locator and lexical value;
- distinction between origin statement, TWIRX interpretation and canon change;
- packet and batch byte integrity;
- no partial manifest or partial database admission;
- no public packet derived from private/authenticated material;
- origin/policy/route authority cannot be broadened by semantic state;
- provisional meaning cannot appear as attested;
- queries preserve conflicts and explicit stale/unresolved state;
- subscription cursors are durable, monotonic and gap-detectable;
- sponsorship and commercial value cannot buy ranking or canon authority;
- model/vector/browser output cannot self-promote.

## Principal threats and controls

### Poisoned or adversarial content

Origin content may contain prompt injection, deceptive schemas, oversized
values, recursive structures, ambiguous encodings, malicious compression or
claims designed to collide semantically. Treat all content as data. Apply
E3.2 response/decompression limits before parsing, bounded decoders afterward,
explicit media/charset handling, and no instruction following.

### Semantic overwrite and provenance laundering

Two claims may share words while differing by jurisdiction, valid time, unit,
language, subject identity or scope. Semantic keys include every
meaning-changing dimension. Native fields and proof references are mandatory
in results. Materializations preserve contributing packets and conflicts.

### Delta misclassification

An ontology remap can falsely appear to be a publisher change. Delta class is
canonical. Origin deltas require different source evidence; semantic/canon
deltas preserve the unchanged source evidence reference. Invalid class/kind
combinations fail conformance.

### Replay, rollback and split state

Old valid packets can be replayed as current; a manifest and database head can
diverge; an outbox event can be lost. Batches bind prior batch where declared,
heads use deterministic source/time ordering, admission is transactional, and
outbox delivery uses durable cursors. Rollback to older application code cannot
silently accept a newer unsupported protocol version.

### Database privilege escape

SQL injection or compromised application roles could mutate history, load
extensions or read operator secrets. Public inputs are never SQL. Runtime roles
have no schema ownership, superuser, filesystem/program execution, replication
or extension privileges. Immutable log tables deny runtime update/delete.
PostgreSQL is loopback/Unix-socket only and absent from the public firewall.

### Resource exhaustion

Queries can expand large graphs, request excessive proof, create expensive
sorts, retain cursors forever or force origin refresh. Canonical queries bound
selectors, graph depth/cost, sources, rows, packets, proof bytes, live origins
and deadlines. Database roles have connection, statement, lock, temporary file
and idle-transaction limits. Subscriptions have rate, expiry and retention
limits. Origin work remains under E3.2 budgets/circuit breakers.

### Commercial manipulation

An offer or sponsor can misstate price, hide constraints or pay to rank.
Offer packets bind origin evidence, currency, validity and freshness. Publisher
price and TWIRX operator cost are distinct objects. Hard cost constraints apply
before Pareto selection. Funding class and sponsor identity are auditable but
excluded from semantic authority and ranking.

### Model/vector candidate laundering

A plausible candidate may be mistaken for a reviewed mapping. Candidate source,
model/version and score are separate. Only `provisional_semantic` may contain a
model-derived proposal. Deterministic operation without vectors/models remains
required. Promotion requires the same human/publisher/canon evidence path as
any other candidate.

### Backup compromise and incomplete recovery

Same-host backups do not protect against host loss. Backups are encrypted
off-host, least privilege and manifest-verified. Restore begins isolated with
public delivery disabled. Database and CAS integrity plus rebuild equivalence
must pass before service returns.

## Required adversarial suites

S1/S2/S6 must include at least:

- malformed/non-canonical CBOR, trailing bytes and unsupported versions;
- digest substitution, unequal bytes under one identity and manifest omission;
- missing observation/representation/adapter/mapping/canon evidence;
- native-field removal during semantic mapping;
- attested lane with candidate/disputed/revoked mapping;
- cross-jurisdiction/time/unit/language semantic-key collision attempts;
- origin change mislabeled semantic, and semantic change mislabeled origin;
- duplicate/out-of-order/replayed batches and cursor gaps;
- transaction kill between identity, log, head, delta and outbox steps;
- role attempts to update/delete immutable rows, install extensions, read files,
  execute programs, change roles or access unrelated schemas;
- unbounded ontology cycles, path explosion and integer overflow;
- query proof/row/source/deadline exhaustion;
- live refresh without an admitted work order;
- sponsor/cost fields attempting to influence authority ranking;
- stale, retracted, revoked and fixture packets entering public current views;
- backup corruption, missing WAL/CAS artifact and materialization rebuild
  mismatch.

The existing E3.2 network adversarial suite remains mandatory for every live
refresh path, including IPv4/IPv6 private ranges, rebinding, redirect
revalidation, alternate IP encodings, unexpected schemes/ports, decompression,
TLS/Host mismatch, revocation and emergency disablement.

## Incident response states

An operator can independently:

- disable ingestion globally;
- revoke an origin/work order;
- suspend a compiler contract or adapter;
- dispute/revoke a mapping or canon module;
- suspend a materialization;
- stop subscription delivery;
- place the API in proof-only/read-unavailable mode.

These controls append state/evidence and fail closed. They do not delete or
rewrite history. A sensitive-data incident may invoke a separately governed
retention/removal procedure; public outputs must disclose any resulting proof
unavailability without repeating the sensitive value.
