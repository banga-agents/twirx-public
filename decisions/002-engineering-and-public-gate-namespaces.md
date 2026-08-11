# ADR 002 — Engineering and public gate namespaces

**Status:** Accepted

**Date:** 2026-08-10

**Decision owner:** Genesis steward

**Applies to:** Project roadmap and release terminology

## Problem

The original architecture numbered technical research steps as `Gate 1`,
`Gate 2`, and later gates. The Public Alpha plan also needs milestones for the
website, documentation, treasury, and release preparation. Reusing one number
sequence for both would make evidence claims ambiguous. The new Live
Provenance Lab also combines a deliberately small part of several earlier
research steps, so simply renumbering one document would preserve the
conflict.

## Decision

TWIRX uses two permanent namespaces:

- `E0`, `E1`, `E2`, ... identify engineering gates with executable acceptance
  criteria and evidence reports.
- `P0`, `P1`, `P2`, ... identify public-foundation milestones such as the
  website, documentation, treasury, and public-readiness work.

The current sequence is:

```text
E1  Source-Statement Evidence Spine
P1  Public Foundation
E2  Live Provenance Lab
E3  Genesis Atlas 500 (amended by ADR 004)
E4  Deterministic Compiler Alpha
E5  Hardened Invite Alpha
```

Earlier documents that say `Gate 1` refer to E1. Earlier speculative gate
numbers are superseded as roadmap labels, not silently treated as completed
work. Their technical subjects remain future work unless an engineering gate
with explicit acceptance evidence includes them.

The public release label remains `Genesis Preview` through E2. `Public Alpha`
requires the separate E3 acceptance gate. ADR 004 replaces the earlier
six-origin floor with the staged Genesis Atlas 500 orthogonal state floors, semantic
discovery, operations, and performance/security evidence.

## Consequences

- A public milestone cannot be mistaken for protocol conformance.
- E2 may prove a bounded typed-interface slice without claiming the universal
  compiler, arbitrary-origin ingestion, or browser discovery.
- Reports and generated status data must name both the namespace and the gate
  title.

## Reversal and migration

Changing a gate's acceptance boundary requires a new decision and an updated
evidence report. Published report files retain their original wording; newer
documents may annotate their historical `Gate 1` label as E1.
