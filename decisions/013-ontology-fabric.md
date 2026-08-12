# ADR 013: Executable ontology fabric and semantic frames

Status: accepted architecture; E4.0 implementation candidate requires founder review

Date: 2026-08-12

## Context

ADR 008 established immutable semantic packets as the smallest reusable,
source-bound unit in the TWIRX Semantic Data Plane. Packets preserve evidence
and native meaning, but an agent-useful grant, measurement series, research
work, vulnerability or commercial offer normally requires several related
statements. Treating every task as an unstructured scan over individual
packets would discard useful domain structure and would not scale to the E4
corpus target.

Conventional ontology documents also leave too much runtime behavior implicit.
TWIRX needs versioned semantic definitions that compile into bounded contracts,
validators, indexes, query templates and visualization declarations without
allowing an implementation, model or generated artifact to become normative
authority.

## Decision

E4 adds three layers without replacing the Semantic Packet Protocol:

```text
semantic packet
    atomic source-bound statement
        -> semantic frame
           proof-linked n-ary agent object
               -> semantic universe
                  versioned domain package and query surface
```

A semantic frame is immutable and self-digest-free. Every populated frame slot
references at least one admitted packet digest. A frame value is a typed
semantic view; it never replaces the contributing packet's native term,
lexical value, locator, source identity or derivation. Unknown, unresolved and
contradictory slots remain explicit.

An ontology module is an immutable, versioned definition of concepts, frame
types, roles, constraints, mapping-claim references, query-template references
and visualization references. The canonical module manifest is deterministic
CBOR. The Genesis authoring carriage is bounded JSON so the compiler can remain
standard-library-only; JSON is not a second canonical wire authority.

The ontology compiler contract is designed to validate authoring records and
emit canonical module, frame, mapping and universe artifacts plus generated
indexes. The E4.0 implementation emits canonical module and universe
manifests plus an index; frame and mapping encoders are exercised through the
shared conformance corpus. Later admitted generators may emit Go, TypeScript,
JSON Schema, CDDL fragments, query macros and visualization metadata as
derived conveniences. Language-neutral prose, CDDL and cross-implementation
conformance remain authoritative.

Semantic identity is scoped. The initial relation vocabulary distinguishes:

```text
exact_identity
legal_identity
publisher_record_for
alias
representation_of
version_of
tracks
derived_from
possible_match
same_under_context
```

No unqualified universal `sameAs` relation is admitted. A model may propose a
mapping claim or frame slot but cannot publish, review or promote it.

## Canonical identities

```text
module_digest  = SHA-256(exact ontology-module-manifest bytes)
mapping_digest = SHA-256(exact mapping-claim-core bytes)
frame_digest   = SHA-256(exact semantic-frame-core bytes)
```

The identified core never contains its own digest. Constituent batches and
snapshots bind detached digests and publish a manifest last.

## Compatibility

The module semantic diff classifies changes as `ADDITIVE`, `RESTRICTIVE`,
`BROADENING`, `MEANING_CHANGING`, `IDENTITY_CHANGING`, `EFFECT_CHANGING`,
`SECURITY_CHANGING` or `DEPRECATION`. Any non-additive change requires a new
module version and an impact report. Existing packets remain immutable; a new
interpretation produces a semantic or canon delta, never an origin delta.

## Security and authority

- Native statements survive every frame and mapping.
- Missing required slot evidence fails closed.
- Optional absence is explicit and cannot become a fabricated value.
- Module imports are exact, acyclic and bounded.
- No generated code, model output, sponsorship or query traffic can enter
  canon automatically.
- Genesis remains read-only and accepts only public-source evidence.

## Consequences

- Agents can retrieve useful composite objects while retaining packet-level
  proof.
- Ontology definitions become testable executable domain architecture.
- Packet and frame counts remain separate, preventing a vanity scale claim.
- Canon changes remain visibly distinct from source changes.

## Rejected alternatives

- Flatten frames into packets: rejected because n-ary context, slot
  completeness and conflicts would become implicit.
- Store only frames: rejected because source-native atomic evidence would be
  erased.
- Use a model-generated ontology directly: rejected because it creates an
  unreviewed authority path.
- Add a YAML runtime dependency: rejected because bounded JSON authoring is
  sufficient and does not expand the trusted dependency surface.
