# Mapping Claim 0.1

**Authority:** Normative for E4.0 mapping conformance

**Status:** Implementation candidate under founder review

A mapping claim records a versioned, scoped proposal or reviewed decision that
connects one source-native term to a semantic concept or frame role. It never
alters the source packet.

The canonical core binds origin, optional native schema, native term, optional
locator pattern, semantic concept and optional role, relation, universe,
jurisdictions, languages, conditions, status, evidence packets, module,
review-decision digest and supersession.

Relations are deliberately scoped:

```text
exact close broad narrow contextual candidate
```

`reviewed` requires a human review-decision digest. `candidate` must not carry
one. `revoked` and `disputed` preserve their prior evidence and cannot erase an
earlier claim. A mapping cannot assert publisher identity, access permission or
objective truth.

```text
mapping_digest = SHA-256(exact canonical mapping-claim-core bytes)
```
