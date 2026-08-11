# ADR 004 — E3 Genesis Atlas 500 scope and admission model

**Status:** Accepted for staged implementation; state model superseded by ADR 006

**Date:** 2026-08-10

**Decision owner:** Genesis steward

**Applies to:** Engineering Gate E3

## Decision

Engineering Gate E3 is Genesis Atlas 500. Its public acceptance target is an
exactly enumerated universe of 500 public-knowledge origins whose breadth and
depth are reported without promoting evidence across independent state
dimensions. ADR 006 replaces the A0-through-A9 roll-up below; the subgate rows
remain historical planning context, not the active state vocabulary.

The gate is delivered through independently testable subgates:

| Subgate | Required outcome |
|---|---|
| E3.0 | Accepted scope, measured capacity, exact 500-candidate selection, schemas, deterministic validation, and evidence-derived counters |
| E3.1 | 500 A1 catalog records and 500 human-reviewed A2 policy records; controlled scheduler remains dry-run until its egress boundary passes |
| E3.2 | 300 bounded A3 profiles and 100 immutable A4 observation packages |
| E3.3 | 50 A5 native-schema packages and 25 A6 deterministic adapters with malformed/adversarial fixtures |
| E3.4 | 12 reviewed A7 cross-origin operation families, 8 monitored A8 origins, and 1 publisher-authored or publisher-verified A9 origin |
| E3.5 | Public semantic discovery, proof downloads, three reproducible controlled comparisons, learning-ledger statistics, operations evidence, and complete gate report |

E3 is admitted only when every subgate and every floor in the E3 work order
passes. Partial subgates are reported as implementation progress, never as
gate admission.

The active acceptance report derives catalog, completed policy, technical,
publisher, health, adapter-trust and mapping-trust counts separately. No one
dimension implies another.

## Acquisition authority

Publisher-native declarations, targeted live observations, and archive/index
observations are distinct evidence classes. Archive evidence is historical.
Discovery is not admission. A candidate record cannot authorize a fetch,
mapping, adapter, or live operation.

Direct observation requires an A2 policy decision, an admitted catalog
destination, bounded per-origin budgets, DNS and redirect revalidation, and a
separate controlled egress boundary. Unreachable or uncertain policy fails
closed. E3.0 performs no candidate-origin network fetches.

## Semantic authority

Native terms and lexical values are preserved before mapping. Mappings are
versioned reviewable claims tied to evidence and context; they do not assert
objective truth. Models may propose candidates but cannot promote maturity,
admit adapters, alter the canon, or gain execution authority.

## Public agent boundary

The Atlas agent may search evidence-derived catalog and semantic indexes. It
may invoke only registry-admitted read-only operations. Users cannot supply an
outbound destination, browser instruction, credential, write action, payment
action, or canonical-state mutation.

## Model subgate

WSIM Seed remains inactive until repository evidence proves all four corpus
floors: 10,000 observations, 2,000 adjudicated mappings, 1,000 hard negatives,
and 50 origin-held-out origins. A false readiness claim is a gate failure.

## Non-decision

This ADR does not admit E2, authorize deployment, authorize crawling, change
repository visibility, or declare any selected candidate reachable or safe to
access. Those require their own evidence and founder review.
