# TWIRX semantic data plane 0.1

**Authority:** Normative design for E3.3 S1

**Status:** Contracts frozen; local S1 engineering authorized by ADR 010; no
production-state or deployment admission

This directory defines the language-neutral objects at the E3.3 boundary.
Go, C, PostgreSQL, CBOR libraries, HTTP, MCP and any future implementation are
non-normative. Conformance is determined by the canonical encoding rules,
schemas, independently reviewed vectors and required failure behavior.

## Object graph

```text
representation + observation + contract + adapter + mapping/canon
                              |
                              v
                    semantic packet core
                              |
                 SHA-256(exact canonical bytes)
                              |
                         packet batch
                              |
                  manifest published last
                              |
           immutable log + current materializations
                              |
                   query / compare / subscribe
                              |
                    proof-linked result core
```

Canonical cores never contain their own digest. Detached identifiers are
lowercase SHA-256 values over exact deterministic-CBOR bytes. A wrapper may
show those identifiers but cannot change the identified object.

## Normative files

- `SEMANTIC_PACKET_0_1.md` defines packet, batch, identity and delta behavior.
- `QUERY_DELTA_FABRIC_0_1.md` defines typed query, subscription and result
  behavior.
- `GENESIS_STATE_STORE_0_1.md` defines implementation-independent state-store
  invariants and the Genesis relational realization.
- `SEMANTIC_SNAPSHOT_0_1.md` defines the immutable manifest-last release and
  read-only edge boundary.
- `READ_ONLY_SNAPSHOT_RUNTIME_0_1.md` defines the bounded offline build,
  reconciliation and materialized-query profile used by the grant demo.
- `TRUST_AND_RANKING_TABLES_0_1.md` freezes lane/delta transitions and the
  deterministic planner ordinals without creating a hidden aggregate score.
- `SECURITY_0_1.md` defines trust boundaries and fail-closed behavior.
- `BENCHMARK_0_1.md` defines reproducible value and performance evidence.
- `E3_3_SUBGATES.md` defines the independently reviewable S1-S10 sequence.
- `schemas/cddl/semantic-data-plane.cddl` defines the canonical array layouts
  and bounds.

E4 adds the Ontology Fabric contracts in `spec/ontology/`. Semantic Frames
compose packets but never replace their native statements; Mapping Claims,
Ontology Module manifests and Semantic Universe manifests have independent
canonical identities and explicit human-review boundaries. The E4 snapshot
and agent prototype documents are explanatory until their separate release
gate passes.

The prose and schema must agree. A contradiction blocks conformance rather
than allowing an implementation to choose the more convenient interpretation.

## Scope

E3.3 covers public-source, read-only semantic state; deterministic packet and
delta compilation; rebuildable materialized views; typed queries; proof-linked
comparisons; and durable subscriptions.

It does not authorize:

- arbitrary URL or arbitrary MCP access;
- browsers or LLMs in the trusted compilation/execution path;
- authenticated private data;
- writes, purchases, payments, bookings or material actions;
- automatic policy, publisher, adapter, mapping or canon promotion;
- model/vector output as identity, truth, trust or authority;
- public live ingestion before its separate security and target-host gates
  pass.

## Determinism profile

- Canonical protocol structures are fixed-order arrays.
- Collections that represent sets are sorted by their exact canonical bytes;
  duplicates are rejected.
- Ordered histories preserve explicit sequence order.
- Timestamps are canonical UTC strings with second precision unless a field
  explicitly requires a different precision.
- Digests are 32-byte values in CBOR and `sha256:` lowercase hexadecimal in
  human/display formats.
- Monetary and decimal quantities use bounded canonical decimal lexical text,
  never binary floating point.
- All planner scores, path costs and telemetry counters use integers.
- Unknown and absent states are explicit; an absent value never silently
  becomes false, zero or empty.

## Required independent verification

S1 requires two independent implementations to accept the same valid vectors
and reject the same malformed/adversarial vectors for packet cores, manifests,
deltas, queries and query results. The restricted-C verifier remains offline
and receives no network, filesystem traversal or database capability beyond
explicit caller-provided bytes.
