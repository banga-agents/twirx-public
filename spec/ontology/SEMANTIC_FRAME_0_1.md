# Semantic Frame 0.1

**Authority:** Normative for E4.0 frame conformance

**Status:** Implementation candidate under founder review

## Purpose

A semantic frame composes admitted semantic packets into one agent-useful
n-ary object while preserving packet-level proof. Examples include an
indicator observation, grant opportunity, research work, vulnerability or
commercial offer.

A frame is an interpretation produced by a named compiler and ontology module
set. It is not an origin statement and does not become objective truth through
admission.

## Canonical core and identity

The fixed-order `semantic-frame-core` in the Semantic Data Plane CDDL binds:

- exact universe and frame type;
- a source-stable native key and optional canonical identity candidates;
- sorted, unique semantic roles;
- explicit slot values, absence and conflict state;
- context and valid time;
- epistemic lane and completeness;
- the complete contributing packet set;
- module-set, mapping and compiler identities;
- immutable lifecycle lineage.

```text
frame_digest = SHA-256(exact canonical semantic-frame-core bytes)
```

The core is self-digest-free.

## Slot proof law

Every slot references at least one admitted packet digest. The sorted union of
all slot packet references MUST equal the frame derivation packet set. This
prevents both unproved slots and hidden contributing packets.

`resolved` slots contain one or more typed values. Every non-resolved status
contains no value. `cardinality = one` permits at most one typed value. Multiple
source values are never silently collapsed; their conflict state is
`preserved` or `unresolved`.

## Native fidelity

Frame values do not duplicate or replace packet-native terms. Trace follows:

```text
frame slot -> packet digest -> native term + lexical value + locator
           -> representation + observation + derivation
```

If a required source field lacks evidence, frame compilation fails. An optional
missing field uses an explicit packet/slot absence state.

## Epistemic lanes

- `observed_native` frames carry no semantic mapping identifiers.
- `provisional_semantic` frames bind one or more candidate mapping claims.
- `attested_semantic` frames bind only reviewed mapping claims in an admitted
  module set.

Completeness is an integer from 0 through 1,000,000 and describes populated
declared slots; it is not truth confidence.

## Required rejection

Reject non-canonical CBOR, trailing bytes, maps, indefinite lengths, oversized
arrays, duplicate or unsorted roles/digests/identifiers, invalid typed values,
unproved slots, unused derivation packets, illegal lane/mapping combinations,
time reversal, invalid lifecycle lineage and non-empty 0.1 extensions.
