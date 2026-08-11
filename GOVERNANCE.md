# Governance During Genesis

Typed Web is founder-led, specification-bound, and publicly inspectable during Genesis.

## Decision classes

| Class | Examples | Required record |
|---|---|---|
| Implementation | Refactor, internal package layout, test helper | Pull request and tests |
| Architecture | Storage abstraction, wire encoding, isolation boundary | ADR |
| Protocol | TWIR semantics, provenance fields, trust states | RFC plus conformance vectors |
| Security-critical | URL policy, signature rules, action risk | Threat-model update, security review, RFC/ADR |
| Constitutional | Ownership, funding rights, protocol guarantees | Charter amendment |

## Admission rule

Automated systems may generate candidates. Only the admission path may create canonical releases. Genesis admission is a reviewed repository change containing fixtures, expected outputs, compatibility analysis, and provenance.

## Contributor influence

Influence follows demonstrated responsibility:

- accurate implementation;
- review quality;
- security discipline;
- maintenance over time;
- documentation and conformance work;
- respect for the charter.

Money, popularity, institutional status, nationality, ideology, or access to a model provider do not buy technical authority.

## Conflict handling

Disagreement should be resolved through explicit claims, evidence, experiments, benchmarks, threat analysis, and reversible decisions where possible. Competing modules or implementations are preferable to forcing false consensus into the protocol.
