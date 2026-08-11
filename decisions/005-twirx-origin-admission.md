# ADR 005 — TWIRX origin A1/A2 admission

**Status:** Superseded in part by ADR 006

**Date:** 2026-08-10

**Decision owner:** Genesis steward

**Applies to:** The `twirx-org` record in the Genesis Atlas

## Decision

Admit `https://twirx.org` as a reviewed Genesis Atlas catalog record. The
project controls the origin and can attest its publisher identity and intended
public-purpose use from repository evidence. Under ADR 006 this supports
`catalog.state: cataloged` and `publisher.status: publisher_approved`; it does
not support a completed policy assessment.

The historical access value was `review_required`, not `allow`. The live
`robots.txt` representation has not been retrieved through an admitted
isolated worker, and the external-origin risk review is not complete. The
record therefore has zero request and storage budgets, a disabled refresh
class, and a disabled scheduler.

ADR 006 corrects the active representation to
`policy.review_state: pending` plus `policy.decision: uncertain`.
`review_required` never counts as completed policy evidence.

## Evidence classes

- `CHARTER.md` records the project's public-infrastructure purpose and
  governance constraints.
- This decision records the Genesis steward's publisher-side approval for the
  project origin to enter the reviewed Atlas catalog.
- `atlas/policies.json` records the exact access assessment. It does not claim
  a live observation or third-party authorization.

Repository control is evidence about the TWIRX project surface. It is not a
claim that every representation at the origin is objectively true, current,
or safe to retrieve.

## Security effect

The record cannot enter the retrieval frontier while policy review is pending
and uncertain. A later `permit_live` decision requires a separate policy-artifact
revision with successful or explicitly unavailable robots evidence, completed
terms review, accepted risk review, bounded budgets, and explicit maintainer
approval. That later revision changes only policy state; it must not promote a
technical, publisher, health, adapter-trust, or mapping-trust claim.

## Non-decision

This decision does not fetch `twirx.org`, activate a crawler, deploy a worker,
publish the repository, alter VPS services, admit another origin, or claim
Atlas live health or publisher signing.
