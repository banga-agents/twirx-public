# ADR 001 — Genesis validation profile

**Status:** Accepted for Gate 1  
**Date:** 2026-08-10  
**Decision owner:** Genesis steward  
**Applies to:** Observation envelope v1 and adapter format 0.1

## Problem

The bootstrap encoded the same observation fields in Go, C, and CDDL but did
not apply identical validation rules. JSON decoding also inherited duplicate
key behavior and resource limits from one implementation. That would allow an
implementation to become normative by accident.

## Decision

### Observation envelope

- The CDDL field order and byte bounds are normative.
- Every text string is valid UTF-8, non-empty, and contains no U+0000.
- `retrieved-at` is the canonical output of RFC 3339 Nano in UTC: a `Z`
  suffix, no numeric offset, at most nine fractional digits, and no trailing
  fractional zeroes.
- Genesis evidence bodies are at most 2,097,152 bytes. The bound is part of
  envelope v1 conformance, not merely a CLI default.
- Go and C consume the same committed positive and negative vectors.

### JSON profile

- A representation contains exactly one top-level JSON value followed only by
  whitespace.
- Duplicate object keys are rejected at every depth.
- Escaped UTF-16 surrogate code units must form a valid pair; lone surrogates
  are rejected instead of being replaced during decoding.
- Manifests and observed JSON have explicit byte, depth, scalar, container,
  and token limits documented with the adapter format.
- A native JSON lexical value is the decoded JSON scalar. Its original token
  spelling remains recoverable from the immutable body and locator.
- JSON Pointer array indexes use the RFC 6901 grammar: `0` or a non-zero digit
  followed by digits. Signs and leading zeroes are rejected.
- Genesis `trim`, `uppercase`, and `lowercase` transforms operate on ASCII
  only. Native Unicode text is preserved, while future Unicode-aware
  normalization requires a versioned transform with a declared Unicode
  version.

### Result publication

- A result is written through a bounded temporary file in the destination
  directory, synchronized, and atomically renamed.
- Resolved empty strings remain explicit values. Unresolved values omit the
  lexical member and are distinguished by status.

## Alternatives considered

- **Keep limits as implementation policy.** Rejected because Go could produce
  observations the independent verifier refused.
- **Accept JSON duplicate keys with last-key-wins behavior.** Rejected because
  it hides source ambiguity and varies across implementations.
- **Adopt a third-party CBOR or JSON package.** Rejected for Gate 1 because the
  required bounded subset is small and adding a runtime dependency needs
  separate maintainer approval.
- **Specify Unicode case conversion immediately.** Deferred because stable,
  cross-language behavior requires a versioned Unicode data contract.

## Security and compatibility effects

The profile fails closed on inputs previously accepted by one implementation.
It prevents ambiguous source statements, bounds privileged parsing, and makes
the independent verifier meaningful. Existing bootstrap fixtures remain
valid. Because Genesis is pre-release, the stricter rules are adopted before
observation v1 is declared stable.

## Reversal and migration

Relaxing a bound or adding a different time, JSON, or Unicode profile requires
a new version or a compatibility decision with public vectors. Released
vectors are immutable.
