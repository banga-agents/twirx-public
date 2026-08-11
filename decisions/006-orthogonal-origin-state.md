# ADR 006 — Orthogonal origin state and canonical E2/Atlas identity

**Status:** Accepted

**Date:** 2026-08-11

**Decision owner:** Genesis steward

**Applies to:** E2 origin identity and Engineering Gate E3

## Decision

Replace the Atlas A0-through-A9 aggregate maturity interpretation with these
independent origin state dimensions:

```text
catalog.state       candidate | cataloged
policy.review_state pending | completed
policy.decision     permit_live | permit_with_constraints | profile_only |
                    catalog_only | deny | uncertain
technical.stage     unprofiled | profiled | observed | native_schema |
                    compiled | semantically_linked | live
publisher.status    unclaimed | domain_verified | publisher_approved |
                    publisher_signed
health.status       unknown | healthy | degraded | stale | suspended | revoked
adapter_trust       none | candidate | reviewed | conformant | revoked
mapping_trust       none | candidate | reviewed | disputed | revoked
```

Technical stage remains ordered only inside the technical dimension. It does
not imply a completed policy review, publisher identity, current health,
adapter trust or mapping trust. Publisher approval does not imply live health
or retrieval permission. Adapter conformance does not promote a semantic
mapping.

## Pending policy

`review_required` is removed from the active vocabulary. An incomplete review
is `policy.review_state: pending` and fails closed. Its decision is
`uncertain`, but it is not counted as a completed uncertain decision. A
completed uncertain decision requires explicit human reviewer and review-time
evidence.

Only a completed `permit_live` or `permit_with_constraints` decision can make
an origin eligible for the live-operation scheduler. The scheduler, request
budgets and storage budgets remain disabled in this decision.

## Canonical identity

`atlas/registry.json` is the canonical origin identity and state registry.
The E2 execution catalog retains its bounded routes, typed inputs and
operation ownership, but every entry names a canonical `registry_id`, and E2
startup fails closed if publisher, host, scope or alias binding disagrees with
the registry.

The qualifying E2 real origins are imported as `twirx-org` and
`api-worldbank-org`. Their immutable replay representations, contracts,
conformance vectors, generated proof metadata and semantic-closure digests are
bound as exact evidence. Their technical state does not complete their
separate Atlas policy review.

`controlled-origin-lab` is represented as a `test_fixture`. It remains usable
under the explicit E2 local-fixture mode and is excluded from every
Genesis-500 public-origin count, coverage count and semantic count.

## Base topology

PR #4 merged before PR #8. The exact admitted E2 head
`7ef731ce7d8b8c865f0aedf634fc407e93a5693c` is an ancestor of the PR #8 merge
`ef7c74bcfe3f673b77d14e21047139e1a26217c7` and occurs once in reachable
history. The PR #4 merge commit is not itself an ancestor because the later
stack was based on the admitted E2 head before GitHub created that merge
commit. No E2 implementation commit is duplicated, and this correction is
based on the merged PR #8 tip.

## Security effect

- Selection remains non-authoritative for network access.
- Unknown or pending policy fails closed.
- Controlled fixtures cannot inflate public-origin progress.
- Exact E2 evidence files are rehashed when the canonical registry loads.
- The Atlas frontier remains dry-run with network access disabled and zero
  jobs on the committed state.
- No state artifact can authorize a user-supplied URL.

## Non-decision

This decision does not activate live retrieval, complete any public-origin
policy assessment, deploy a service, admit E3, add a dependency, alter E1/E2
operation behavior or authorize all 500 candidates for processing.
