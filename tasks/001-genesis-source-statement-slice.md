# Codex Work Order 001 — Genesis Source-Statement Slice

## Mission

Bring the existing bootstrap to a reviewable Gate 1 implementation of the smallest complete Typed Web path:

```text
controlled origin
→ safe observation
→ content-addressed body
→ canonical observation envelope
→ Go verification
→ independent C verification
→ deterministic offline extraction
→ native statement + semantic view
→ field-level provenance
```

Do not widen scope. Make this path exact.

## Starting point

The repository already contains a runnable bootstrap. Treat it as untrusted pre-alpha code to audit against the specifications, not as automatically correct.

Run first:

```bash
make clean
make build
make test
make demo
```

Record baseline failures before editing.

## Required work

### 1. Audit specification-to-code agreement

Verify that:

- `schemas/cddl/observation.cddl` exactly matches Go and C field order and bounds;
- deterministic CBOR rejects non-shortest integer encodings, indefinite forms, trailing bytes, wrong field counts, and wrong hash length;
- Go and C agree on every public conformance vector;
- the result's native lexical value is never replaced by its transformed semantic value;
- every resolved and unresolved field contains complete provenance.

Resolve every ambiguity in specification text rather than allowing one implementation to become normative by accident.

### 2. Harden JSON decoding

- Reject multiple top-level JSON values and trailing non-whitespace bytes in adapter manifests and observed JSON.
- Define duplicate-key behavior. For Gate 1, prefer deterministic rejection over last-key-wins ambiguity.
- Add bounded JSON nesting and scalar-length policy or document the exact temporary limitation and create the next security task.

### 3. Make result writes atomic

The CLI currently writes `result.json` directly. Replace this with bounded atomic write behavior equivalent to observation bundle writes. Never leave a partial canonical-looking result.

### 4. Expand conformance vectors

Add committed fixtures and expected outcomes for:

- valid minimal envelope;
- non-canonical unsigned integer;
- indefinite array;
- trailing bytes;
- body hash mismatch;
- body size mismatch;
- missing required source field;
- missing optional source field becoming `unresolved`;
- wrong media type;
- adapter origin mismatch;
- lowercase currency preserved natively and normalized semantically;
- JSON pointer escapes `~0` and `~1`;
- provider string containing prompt-injection text treated as ordinary data.

The test suite must execute these without external network access.

### 5. Fuzz the trust boundaries

Add native Go fuzz targets for:

- `observation.UnmarshalCBOR`;
- JSON pointer decoding and resolution;
- adapter manifest decoding;
- result extraction over bounded JSON values.

Seed each target with valid and malformed conformance vectors. Fuzz targets must never panic or allocate without policy bounds.

Add a documented libFuzzer or AFL++ harness plan for the C verifier. Implement it now only if the local toolchain supports it without adding runtime dependencies.

### 6. Validate the network policy

Add tests for:

- loopback denial by default;
- explicit loopback fixture permission;
- private IPv4 and IPv6 rejection;
- carrier-grade NAT rejection;
- embedded credentials rejection;
- non-standard port rejection under public policy;
- redirect revalidation;
- response body limit after decompression.

Do not claim production-grade DNS-rebinding resistance solely from unit tests. State residual risk accurately.

### 7. Produce the Gate 1 evidence report

Create `reports/gate-1-genesis.md` containing:

- commit identifier;
- toolchain versions;
- test commands;
- pass/fail totals;
- C sanitizer results;
- benchmark results;
- known residual risks;
- exact statement of what Gate 1 proves and does not prove.

## Acceptance criteria

All must be true:

- `make test` passes offline.
- `make demo` completes and extraction still succeeds after the controlled origin stops.
- Both Go and C reject every invalid observation vector.
- Corrupted CAS evidence is rejected.
- Required-field drift fails closed.
- Optional-field absence is explicit and provenance-bearing.
- No new third-party runtime dependency exists.
- No `unsafe`, cgo, native plugin, browser, model, database, blockchain, or write action is introduced.
- Documentation describes actual behavior, not intended future behavior.
- The Gate 1 report identifies residual weaknesses without marketing language.

## Explicit non-goals

Do not implement:

- arbitrary schema inference;
- HTML or browser extraction;
- PostgreSQL;
- Wasm execution;
- MCP, WebMCP, or OpenAPI runtime bindings;
- ontology reasoning;
- cryptographic signatures;
- wallets or payment actions;
- public deployment.

Those belong to later gates.

## Stop conditions

Stop and report rather than silently redesign if:

- a normative rule conflicts with another normative rule;
- the C and Go implementations cannot agree without changing the wire specification;
- a security property requires a new dependency or operating-system privilege;
- a test requires public-network access;
- scope expansion appears necessary.
