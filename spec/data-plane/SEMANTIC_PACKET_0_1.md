# Semantic Packet Protocol 0.1

**Authority:** Normative for E3.3 packet and delta conformance

**Status:** Design frozen; Go/restricted-C S1 implementation candidate under
founder review

## Meaning

A semantic packet is the smallest immutable reusable unit produced by TWIRX.
It states what an origin representation contained, where it was located, how
it was typed, which optional semantic mapping was applied, and which evidence
and versions support the derivation. Admission does not turn provider content
or a TWIRX interpretation into objective truth.

Supported packet kinds are:

```text
claim
state
capability
offer
relationship
event
measurement
document
```

The first compiler implementation may admit only a documented subset but must
reject unsupported kinds explicitly.

## Canonical packet core

The canonical layout is `semantic-packet-core` in
`schemas/cddl/semantic-data-plane.cddl`. It contains, in fixed order:

1. version and kind;
2. source-native subject plus zero or more canonical identity candidates;
3. source-native predicate plus an optional semantic predicate;
4. the exact native lexical value, representation media/language and optional
   typed value;
5. sorted contextual dimensions, jurisdiction, language and scope;
6. observation, assertion, valid and source-modification time;
7. origin, representation digest, native locator and native-schema reference;
8. observation, adapter, extraction-plan, transformation, mapping, semantic
   closure and compiler identities;
9. epistemic lane, extraction/mapping status, explicit confidence state,
   authority and freshness;
10. lifecycle, retention and disclosure class;
11. a currently empty extension array.

All required source and derivation digests are raw 32-byte SHA-256 values in
canonical CBOR. A non-applicable optional digest is `nil`, not an all-zero
digest. A public packet cannot contain credentials, cookies, authenticated
source material or unreviewed personal data.

## Native fidelity

Every packet preserves the source-native subject, predicate, locator and
lexical value before semantic mapping. The semantic predicate and typed value
are separate optional fields. A mapping cannot replace or normalize away the
native value.

An optional missing value uses one explicit unknown state:

```text
unknown
not_observed
not_provided
not_applicable
withheld
redacted
unresolved
contradictory
invalid
confirmed_absent
```

It cannot be represented as an empty string, zero, false or a fabricated typed
value. A missing required value rejects packet compilation.

## Typed values

The initial type vocabulary is:

```text
boolean
integer
decimal
text
date
datetime
duration
uri
identifier
```

The canonical typed value contains a type and lexical representation. Lexical
forms are validated by the named type but remain text to avoid language and
floating-point differences. The Genesis lexical profile is:

```text
boolean       true | false
integer       optional "-", then 0 or a non-zero digit plus digits
decimal       canonical integer, ".", one or more digits
date          YYYY-MM-DD with a valid Gregorian calendar date
datetime      YYYY-MM-DDTHH:MM:SSZ with a valid date and UTC second precision
duration      P[nD][T[nH][nM][n[.fraction]S]], at least one component
uri           absolute URI: ASCII scheme, colon and non-empty remainder;
              no whitespace or control characters
identifier    bounded non-empty UTF-8 identifier text
text          bounded non-empty UTF-8 text
```

Unit and currency are separate optional semantic identifiers. Currency uses
three uppercase ASCII letters. A typed value is present only when its status
is `resolved`.

## Epistemic lanes

```text
observed_native
provisional_semantic
attested_semantic
```

- `observed_native` preserves deterministic extraction without a canonical
  semantic mapping.
- `provisional_semantic` carries a visible candidate/review-pending mapping.
- `attested_semantic` requires the mapping artifacts and trust decision defined
  by the admitted canon policy.

A confidence value is allowed only for a provisional candidate and is an
integer from 0 through 1,000,000. It never upgrades a lane or mapping status.

## Packet identity

```text
packet_digest = SHA-256(exact canonical semantic-packet-core bytes)
```

The packet core contains no `packet_digest`. A digest collision where the same
identifier is presented with unequal bytes fails closed and raises an
integrity incident.

## Semantic key and current head

A semantic key is a detached SHA-256 digest over a canonical key object that
includes the origin, native/canonical subject, predicate, valid-time scope,
jurisdiction, language and all meaning-changing context dimensions. The key
prevents an observation for one scope from silently replacing another.

A current head is a rebuildable projection:

```text
semantic key -> current packet digest -> prior digest -> delta -> cursor
```

It is not part of the immutable packet. Equal observation time is resolved by
declared source sequence and then packet digest; database arrival order is
never authoritative.

## Packet batch

A compiler run publishes:

1. packet cores;
2. delta cores;
3. rejection records;
4. deterministic metrics;
5. a canonical batch manifest written last.

The manifest binds the compiler contract, origin, observation set, packet and
delta digests, rejection-report digest, metrics digest, previous batch where
applicable, and artifact sizes. Packet and delta entries are sorted by digest.
One batch contains at most 32,768 packets and 32,768 deltas; the compiler must
split larger work without splitting an individual observation's atomic
publication unit. The general artifact list contains at most 64 non-packet,
non-delta artifacts.
Every referenced artifact must exist and rehash before admission. The manifest
does not list or contain its own digest.

```text
batch_id = SHA-256(exact canonical batch-manifest bytes)
```

If publication stops before the manifest exists, no batch exists. Admission is
atomic: either all identities, packets, deltas, heads and outbox rows commit or
none do.

## Delta classes

Delta cores are immutable and self-digest-free.

### Origin delta

The observed representation or source-native statement changed:

```text
added | modified | withdrawn | restored | source_retracted
```

### Semantic delta

TWIRX interpretation changed while the source evidence did not:

```text
mapped | remapped | narrowed | broadened | disputed | attested | de_attested
```

### Canon delta

A concept, edge, mapping or closure module changed:

```text
module_added | module_superseded | mapping_superseded | closure_changed
```

The class is part of the canonical object and cannot be collapsed to a generic
`updated` event. A semantic or canon delta must preserve the unchanged source
packet/evidence reference so clients cannot mistake reinterpretation for a
publisher statement.

## Lifecycle

```text
current | superseded | withdrawn | stale | retracted | invalid
```

Lifecycle transitions append deltas and change current projections. They do
not rewrite or erase packet history. Retention-law or privacy deletion is a
separate governed process and must leave a non-sensitive tombstone when the
applicable policy permits.

## Rejection requirements

Conforming implementations reject:

- non-canonical CBOR, maps, indefinite lengths, trailing bytes or unknown
  extensions;
- wrong lengths, unsorted sets, duplicates or out-of-range counts;
- missing required evidence or mismatched digests;
- semantic values that omit the native term/value/locator;
- semantic/attested lanes without required mapping and closure references;
- typed values whose status, type or lexical form conflict;
- origin deltas without changed origin evidence;
- semantic/canon deltas misrepresented as origin changes;
- public packets with disallowed disclosure/retention classification;
- a manifest published before, or inconsistent with, its artifacts.
