# Contributing

Typed Web is pre-alpha public infrastructure. Correctness, provenance, and reviewability matter more than feature count.

## Before changing code

Read:

- `MANIFESTO.md`
- `CHARTER.md`
- `SECURITY.md`
- `THREAT_MODEL.md`
- `AGENTS.md`

## Development loop

```bash
make build
make test
make benchmark
```

All normal tests must run without public internet access. Network-dependent tests must be explicit and must be recordable as replay fixtures.

## Change requirements

A pull request should include:

- the invariant or problem addressed;
- tests proving the change;
- documentation updates;
- compatibility and security effects;
- evidence for performance claims;
- an ADR or RFC when protocol behavior changes.

## Dependency policy

Genesis deliberately uses no third-party Go runtime dependency. Adding a dependency requires a written rationale covering:

- why the standard library or a small internal implementation is insufficient;
- maintainer and release provenance;
- license;
- transitive dependency surface;
- vulnerability and update process;
- replacement strategy;
- impact on language and infrastructure sovereignty.

## Security

Do not open public issues for suspected vulnerabilities. Follow `SECURITY.md`.

## Semantic changes

Never silently change the meaning of a released term or mapping. Add a version, compatibility classification, migration explanation, and impacted-adapter list.
