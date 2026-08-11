# Engineering Gate E2 — Live Provenance Lab

## Objective

Produce one bounded public implementation in which a person or local agent
selects a reviewed origin, inspects its typed contract, invokes a read
operation, traces every field to a source-native statement and declared
derivation, downloads the proof bundle, and verifies it offline.

## Required boundary

- Catalog identifiers and bounded typed input only; no arbitrary URL.
- Read effects only.
- No browser, model, authentication, private origin, write, payment,
  blockchain, public remote MCP, or automatic submitted-URL fetch.
- Preserve every E1 invariant and format.

## Acceptance

1. TWIR Core 0.1 and one canonical contract source generate CLI, JSON Schema,
   OpenAPI, and local MCP bindings.
2. The catalog contains at least three reviewed origins, including one
   publisher-authored interface and one official external JSON API.
3. At least five operations execute with typed input.
4. Observation v1 remains unchanged; Transport Evidence v2 binds redirects
   and a strict response-header allowlist.
5. Canonical typed results preserve native and semantic views, transformations,
   mappings, exact closure, and digest bindings.
6. Proof bundles include the representation and are published manifest-last.
7. Go re-extracts and verifies the result. Restricted C independently verifies
   the canonical result, manifest, observation, representation, digest
   bindings, and semantic closure without network or writes.
8. Shared positive and adversarial Go/C vectors, Go fuzz targets, C libFuzzer,
   GCC, Clang, ASan, UBSan, race, vet, and offline end-to-end tests pass.
9. The HTTP Lab binds only to loopback, has bounded parsing/output, per-client
   and per-origin token buckets, a global concurrency cap, deadlines, strict
   headers, no credentialed CORS, and no directory exposure.
10. Fresh and offline replay modes work. `make demo-e2` performs a real local
    MCP tool call and verifies replay after origin access ends.
11. Benchmark, load, controlled browser-comparison, and exact unresolved-risk
    reports identify the measured host and scope.
12. The final report records exact commands and does not label the project
    Public Alpha; that label requires E3.

ADR 003 must be explicitly accepted before E2 is admitted because it resolves
the non-cyclic result and manifest publication graph.
