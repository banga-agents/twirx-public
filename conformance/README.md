# Conformance corpus

The corpus is shared protocol evidence for independent implementations.

- `fixtures/` contains immutable valid source representations and evidence bodies.
- `expected/` contains semantic expectations that do not depend on retrieval time.
- `adversarial/` contains malformed, ambiguous, deceptive, or resource-hostile inputs.
- `observation/vectors.json` binds canonical CBOR inputs to envelope and evidence expectations.
- `extraction/vectors.json` binds source bodies and adapters to extraction expectations.
- `adapters/` contains conformance-only adapter manifests.
- `observatory/v1/` contains the literal-loopback worker job used to prove
  evidence-first robots retrieval and stopped-origin offline replay.
- `archive-acquisition/` contains valid and adversarial acquisition-manifest
  vectors for the fixed-host, sealed-work-order Common Crawl helper.
- `e3-s1/vectors.tsv` contains deterministic-CBOR packet, batch, delta, query,
  subscription, result, materialization, economic and Semantic Snapshot
  vectors shared by the Go and restricted-C E3.3 S1 verifiers.

The observation manifest is consumed unchanged by Go tests, Go fuzz seeds,
the independent C verifier, and the C libFuzzer corpus generator. The
extraction manifest is consumed by Go conformance tests and seeds the bounded
JSON extraction fuzzer. Vector inputs are repository-relative so another
implementation can consume them without importing Go code.

Adversarial files are intentionally not all valid single JSON documents. In
particular, `adversarial/trailing-json.json` exists to prove that a second
top-level value is rejected.
