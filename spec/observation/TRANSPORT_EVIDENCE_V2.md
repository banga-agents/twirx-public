# Transport Evidence v2

**Authority:** Normative for E2 conformance

Observation Envelope v1 remains byte-for-byte unchanged. E2 adds a separate
immutable transport artifact that records the complete accepted redirect
chain and only these final representation-relevant headers:

```text
content-encoding
content-language
content-location
content-type
```

Header names are lower-case and entries are sorted by name then value. Cookies,
authorization fields, arbitrary headers, request credentials, and local trace
metadata are not recorded. Redirect destinations are revalidated by URL and
resolved-address policy before use. The artifact is content-addressed and its
digest is bound by the typed result.
