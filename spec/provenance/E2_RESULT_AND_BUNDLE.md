# E2 typed result and proof bundle

**Authority:** Normative for E2 conformance

**Status:** Accepted by ADR 003

The schemas in `schemas/cddl/result.cddl` and
`schemas/cddl/bundle-manifest.cddl` define the canonical deterministic-CBOR
structures. The formats use fixed-order arrays and the bounded CBOR profile
already exercised by E1.

A resolved field always preserves both the source-native term and lexical
value before its semantic term, type, and lexical value. Transformations and
the mapping identifier are explicit. Missing optional content uses
`unresolved` with both presence flags set to zero. A required unresolved field
does not produce a result.

The canonical result does not contain its own digest or the final manifest's
digest. ADR 003 defines the non-cyclic publication graph. The final manifest
is written only after every named content artifact exists and has been hashed.
A bundle is admitted only if `manifest.cbor` exists, every entry validates,
`result.cbor` independently validates, and its digest equals the manifest's
result identifier. Every listed artifact is non-empty and no larger than
4 MiB.

`result_digest` is the SHA-256 digest of the exact canonical result bytes.
`bundle_id` is the SHA-256 digest of the exact canonical manifest bytes. Both
are detached publication metadata and remain outside the canonical objects
they identify.

Required E2 artifacts are:

```text
adapter.cbor
contract.cbor
input.cbor
observation.cbor
representation.body
result.cbor
semantic-closure.cbor
transcript.json
transport.cbor
manifest.cbor       # written last; never lists itself
```
