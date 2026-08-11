# ADR 003 — Non-cyclic E2 publication digests

**Status:** Accepted

**Date:** 2026-08-10

**Decision owner:** Genesis steward

**Applies to:** E2 typed result and proof-bundle publication

## Problem

The E2 work order asks the canonical result to bind its own result digest and
the bundle-manifest digest, while the final manifest hashes the result
artifact. Those requirements form a cryptographic cycle: changing either
digest changes the bytes whose digest is being computed. A fixed point is not
a meaningful or interoperable publication rule.

## Decision

Use a directed, acyclic publication graph:

```text
content artifacts
      ↓
canonical result bytes
      ↓ SHA-256
result digest
      ↓
final canonical manifest
      ↓ SHA-256
bundle ID
      ↓
API publication record
```

- The canonical result binds the operation, observation, adapter, contract,
  semantic closure, field provenance, and all other inputs required to
  interpret the result.
- `result_digest` is the SHA-256 digest of the complete canonical result bytes
  and therefore exists outside those bytes.
- The manifest lists and hashes every bundle artifact, including the result.
  It is written last and never lists itself.
- `bundle_id` is the SHA-256 digest of the final manifest bytes and therefore
  exists outside those bytes.
- The API publication record exposes both digests and the immutable result
  identifier. It is a transport representation, not a canonical artifact.

The result identifier is `sha256:<result digest>`. Bundle admission succeeds
only when both independent verifiers accept the result and every manifest
entry resolves to bytes of the declared size and digest.

## Security and compatibility effects

The graph prevents unverifiable self-reference, makes partial publication
visible, and supports manifest-last logical transactionality. A directory
without a valid final manifest is not an admitted bundle. The API cannot
alter a canonical artifact by attaching publication metadata.

## Alternatives considered

- **Place zeroes in digest fields while hashing.** Rejected because it creates
  a special preimage representation that consumers could confuse with the
  published bytes.
- **Exclude digest fields from hashing.** Rejected because the phrase
  "digest of this result" would no longer identify one complete byte string.
- **Make the result hash the manifest and omit the result from the manifest.**
  Rejected because the manifest would no longer bind the primary artifact.

## Acceptance record

The Genesis steward accepted this decision on 2026-08-10 subject to the ten
invariants recorded in the accompanying maintainer decision. Acceptance
resolves the cryptographic-topology blocker only; it does not by itself admit
or deploy E2.
