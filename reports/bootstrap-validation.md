# Bootstrap validation report

**Date:** 2026-08-10  
**Artifact:** uncommitted Genesis bootstrap generated for local initialization  
**Claim level:** development validation, not an external audit

## Toolchains

```text
Go: go1.23.2 linux/amd64
GCC: 14.2.0
Clang: 17.0.0
Node.js: 22.16.0
```

The repository recommends Go 1.26 for current development, while the implementation deliberately remains compatible with the available Go 1.23 toolchain used for this validation.

## Commands completed

```bash
make clean
make build
go vet ./...
make test
make benchmark
make demo
```

## Results

- All Go packages compiled and passed tests.
- The C verifier compiled with GCC under C2x mode and strict warnings.
- The C verifier compiled and ran under Clang AddressSanitizer and UndefinedBehaviorSanitizer.
- Valid evidence was accepted.
- Corrupted CAS evidence was rejected.
- The end-to-end slice observed the controlled origin, verified the evidence in Go and C, stopped the origin, and then extracted the result offline.
- Native lexical value `usd` remained preserved while the semantic view normalized it to `USD` through a declared transformation.
- Every emitted field carried body, observation, adapter, locator, transform, and mapping provenance.
- Mintlify configuration parsed and every navigation target existed.

## Microbenchmark

The JSON Pointer lookup microbenchmark on the validation host reported approximately:

```text
195.7 ns/op
64 B/op
4 allocs/op
```

This measures one in-memory pointer lookup only. It is not an end-to-end latency claim and does not decide the long-term implementation language. Whole-pipeline benchmarks will be added after Gate 1 stabilizes canonical behavior.

## What this proves

- The smallest evidence-to-result spine is executable.
- The result can be reproduced from preserved bytes after the origin is unavailable.
- Two language implementations agree on the valid bootstrap envelope.
- Body corruption is detectable.
- Native and semantic views can coexist without overwriting each other.

## What this does not prove

- production-grade SSRF isolation;
- resistance to every parser exploit;
- duplicate-key-safe JSON behavior;
- arbitrary website compatibility;
- public multi-tenant safety;
- browser or model isolation;
- canonical ontology governance;
- action or payment safety;
- cryptographic release provenance;
- external security review.

These are later gates, not implied capabilities.
