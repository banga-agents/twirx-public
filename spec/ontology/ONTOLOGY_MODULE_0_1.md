# Ontology Module 0.1

**Authority:** Normative for E4.0 module conformance

**Status:** Compiler implementation candidate under founder review

## Module manifest

An immutable module manifest binds an exact module ID and semantic version,
status, imports, concept IDs, frame type IDs, role IDs, mapping-claim digests,
query templates, visualizations, source artifact digest and optional review
decision. Collections that represent sets are strictly sorted and unique.

The Genesis module source is bounded JSON stored under `ontology/modules/`.
The compiler hashes its exact bytes and emits deterministic canonical CBOR.
JSON is an authoring carriage, not a parallel canonical protocol.

An `admitted` or `reviewed` module requires a review-decision digest. A `draft`
module cannot carry one. Imports use exact `module-id@version` references,
resolve within the compilation set, are acyclic and are limited to depth 16.

## Semantic diff

The compiler compares two valid module sources and emits exact changes using
the compatibility classes:

```text
ADDITIVE
RESTRICTIVE
BROADENING
MEANING_CHANGING
IDENTITY_CHANGING
EFFECT_CHANGING
SECURITY_CHANGING
DEPRECATION
```

Removal or change of an existing concept, role, frame or mapping reference is
never classified as additive. A non-additive released-module change requires a
new module version and impact review.

## Generated artifacts

The compiler may derive JSON Schema, CDDL fragments, SDK types, index
declarations, query templates, visualization metadata, model labels and
conformance fixtures. Generated artifacts are bound to the module digest and
cannot promote their own definitions into canon.
