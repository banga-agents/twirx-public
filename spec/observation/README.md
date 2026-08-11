# Observation specification

Genesis normative encoding: `../../schemas/cddl/observation.cddl`.

An observation envelope binds request metadata and retrieval time to a SHA-256-addressed body. The envelope does not contain the body itself.

## Version 1 validation profile

The fixed CDDL array order, ranges, and byte bounds are normative. CDDL
`.size` on text and byte strings counts encoded bytes, as specified by
[RFC 8610 section 3.8.1](https://www.rfc-editor.org/rfc/rfc8610#section-3.8.1).
All text fields are non-empty valid UTF-8 and exclude U+0000. Unsigned values
use their shortest CBOR representation, indefinite forms and trailing bytes
are rejected, and the body digest is exactly 32 bytes.

`retrieved-at` is canonical RFC3339Nano UTC with a `Z` suffix. Fractional
seconds, when present, contain one through nine digits and do not end in zero.
The body bound is 2,097,152 bytes. Evidence is conforming only when its size
and SHA-256 digest match the envelope.

The public positive and negative corpus is
`../../conformance/observation/vectors.json`. The Go and independent C
implementations consume that same corpus.

Version 1 is immutable after release. New transport evidence should be represented through a new version or separately hashed artifact rather than silently changing the existing array layout.
