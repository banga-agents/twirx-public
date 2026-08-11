# Instructions for coding agents

This repository is public-infrastructure code. Preserve its invariants before optimizing feature velocity.

## Read first

1. `MANIFESTO.md`
2. `CHARTER.md`
3. `THREAT_MODEL.md`
4. `tasks/001-genesis-source-statement-slice.md`

## Non-negotiable rules

- The protocol is language-neutral. Do not describe Go, C, Rust, or another implementation as normative.
- Do not add third-party runtime dependencies without an ADR and explicit maintainer approval.
- Do not use `unsafe`, cgo, native plugins, or shell execution in trusted Go request paths.
- Do not add network access to the C verifier or adapter extraction path.
- Do not bypass URL policy for convenience. Loopback is permitted only under explicit local-fixture configuration.
- Never infer objective truth from provider content. Describe origin representation and derivation.
- Preserve native source term and lexical value before semantic mapping.
- Missing required evidence fails closed. Missing optional content becomes `unresolved`.
- No LLM, browser, adapter, or generated code may promote itself into canonical state.
- Genesis remains read-only.
- Never commit credentials, wallet private keys, seed phrases, personal addresses, or unredacted sensitive receipts.
- Update documentation and conformance fixtures whenever behavior changes.

## Engineering standards

- Prefer small packages with explicit ownership and bounded inputs.
- Return typed errors with useful context; do not panic on untrusted input.
- Use deterministic output structures; avoid map-dependent canonical behavior.
- Every parser change requires malformed and adversarial tests.
- Every performance claim requires a benchmark.
- All normal tests must run without the public internet.
- Build C with GCC and Clang; run ASan and UBSan.
- Keep the independent verifier independent: do not copy opaque generated output into it without a specification and test vector.

## Required completion report

At the end of a task, report:

- files changed;
- invariants implemented;
- tests and exact commands run;
- unresolved risks;
- deviations from the task;
- next recommended gate.
