# ADR 008: TWIRX as a semantic data plane

Status: accepted architecture; local S1 engineering authorized by ADR 010

Date: 2026-08-11

## Context

E1 established a language-neutral, independently verifiable evidence spine.
E2 added typed, read-only operations and acyclic proof-bundle publication.
E3.2 adds orthogonal origin state, human admission, and a sealed retrieval
boundary. Those systems are necessary, but an origin registry followed by a
route choice is not the final product architecture.

Repeatedly browsing and interpreting the same public representations wastes
network, compute, model context and review work. It also makes two agents more
likely to derive incompatible meanings from the same source. TWIRX needs a
reusable semantic state layer without turning an interpretation into an
unqualified claim of objective truth.

## Decision

TWIRX is an open semantic data plane for public Web representations:

```text
public Web
    -> bounded observation
    -> continuous deterministic compilation
    -> immutable semantic packets and deltas
    -> proof-linked materialized semantic state
    -> query, comparison and subscription fabric
    -> agent and publisher interfaces
```

The fundamental admitted unit is a **semantic packet**. A packet binds the
source-native subject, term and lexical value; an optional typed value; an
optional versioned semantic mapping; source and observation evidence;
derivation; trust lane; time; freshness; and lifecycle. A packet describes
what an origin represented and how TWIRX derived a view. It does not establish
objective truth merely because it was admitted.

Three change classes remain distinct:

- an `origin` delta records that observed origin representation changed;
- a `semantic` delta records that TWIRX interpretation or mapping changed;
- a `canon` delta records that a shared concept, edge or mapping module changed.

The immutable packet log is not overwritten when interpretation changes.
Materialized state is rebuildable from admitted packets, deltas, contracts and
canon versions. A current-state row is a cache with proof references, not an
independent source of authority.

The stable agent surface is compact: `twirx.query`, `twirx.subscribe`,
`twirx.compare`, `twirx.trace`, `twirx.explain`, `twirx.resolve`, and
`twirx.invoke`. A query planner first applies exact concepts, native lexical
retrieval and bounded ontology expansion. Optional model or vector candidates
may propose possibilities but cannot establish identity, trust, permission,
authority or canon. After hard constraints, the planner exposes the Pareto
frontier across authority, proof, freshness, semantic coverage, latency, cost
and uncertainty. The caller selects an explicit preference policy.

Compiled current state is the default read path. A live origin refresh is a
separate bounded subplan and is allowed only through the E3.2 work-order
boundary. Arbitrary URLs, arbitrary MCP servers, browsers, authenticated
capabilities, payments, writes and actions remain outside this decision.

## Publication and proof topology

Canonical packet, delta, result and manifest objects do not contain the
digest of their own final bytes. Their identifiers are detached SHA-256
digests over the exact canonical bytes. A batch or bundle manifest binds all
constituent digests and is published last. API, CLI, MCP and UI wrappers may
display detached identifiers beside the objects they identify.

## Trust lanes

Packets are admitted into one of three non-equivalent lanes:

```text
observed_native
provisional_semantic
attested_semantic
```

Native evidence is always preserved. A provisional mapping is never rendered
as attested. Neither traffic, sponsorship, a model score nor publisher
commercial value can purchase semantic rank or canon admission.

## Consequences

- One verified interpretation can serve many future agent queries.
- Queries over compiled state normally require no browser, model or origin
  request.
- Cross-origin comparison can preserve disagreement rather than hiding it in
  one synthesized answer.
- Subscriptions publish typed, cursor-addressable change events with proof.
- Origin admission remains the authority for whether live observation is
  permitted; semantic state does not broaden network authority.
- E1 and E2 artifacts become compiler inputs and proof references rather than
  being replaced.
- E3.2 interface, capability, access, offer and economics metadata become
  descriptive inputs to packet compilation and planning, not execution grants.

## Implementation gate

This decision authorizes design and conformance work only. Runtime
implementation begins with E3.3 S1 after E3.2 is admitted and merged. Database
installation, public ingestion and deployment additionally require the storage
and recovery gates in ADR 009. No E3.3 branch is merged or deployed without
founder review.

## Rejected alternatives

- Route selection as the primary product: rejected because it repeats origin
  access and interpretation rather than creating reusable semantic state.
- One hidden scalar ranking score: rejected because it obscures incomparable
  authority, freshness, cost, proof and uncertainty tradeoffs.
- Mutating packets after ontology changes: rejected because it would erase the
  distinction between publisher change and TWIRX reinterpretation.
- An LLM as query authority: rejected because natural language may propose a
  typed query but cannot authorize execution or canon changes.
- A centralized crawler for the entire Web: rejected as the long-term model;
  publisher-native and federated observatories are required for Web scale.
