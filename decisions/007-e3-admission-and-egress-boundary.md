# ADR 007: E3 admission factory and sealed egress boundary

Status: accepted for an unactivated E3.2 implementation candidate

Date: 2026-08-11

## Context

Genesis Atlas needs repeatable per-origin review and limited public-origin
retrieval. A single manually edited registry cannot preserve review
provenance, and a worker that accepts a caller-supplied URL would create an
SSRF and control-plane escape surface. Policy, technical maturity, publisher
status, health, adapter trust, and mapping trust are independent state
dimensions. Catalog admission cannot imply that policy review is complete.

## Decision

Each public-origin proposal has its own directory containing a record, a
digest-bound policy-evidence artifact, and an admission decision. The factory
validates identity against the accepted 500-candidate selection, rejects
duplicate identities and aliases, and renders deterministic dossiers and
canonical artifacts. It never creates a decision. Only a completed decision
with `reviewer_type: human` can enter the canonical registry. A completed
catalog admission may preserve `policy.review_state: pending`; it does not
convert `uncertain` into a completed assessment.

Controlled conformance origins have separate test-fixture source records.
They remain in the canonical registry for execution reconciliation but are
excluded from Genesis-500 public-origin counters.

Public retrieval accepts only a root-owned, sealed work-order ID. A
`policy_evidence_collection` order is limited to the exact `/robots.txt`
route of a human-admitted origin while policy remains pending; it cannot
profile or observe content. Every other order requires a completed
retrieval-permitting policy. The work order binds an admitted origin, exact
HTTPS URL and host allowlist, policy and decision digests, human approval
reference, validity interval, and bounded redirect, byte, duration, and
circuit-breaker limits. There is no URL CLI argument. DNS is resolved and
every returned address is revalidated before each connection. Redirects
repeat URL, host, port, DNS, and address checks.

The worker writes the work order before retrieval, writes content-addressed
representation and observation evidence before any parsing, and writes the
manifest last. Downstream consumers rehash the complete spool before parsing.
A disabled control artifact, emergency stop, origin/order revocation,
circuit breaker, and global concurrency lease fail closed.

The deployment candidate uses a dedicated unprivileged account, a strict
read/write path set, cgroup-scoped private-range firewall rules, no secret or
database visibility, and explicit memory, CPU, task, file, and execution-time
limits. It is not installed or activated by repository commands.

## Consequences

- Batch preparation can be automated without automatic approval.
- Agent-prepared records remain visibly pending until a human signs the
  decision artifact.
- The existing E2 public origins and controlled fixture use the same canonical
  vocabulary without weakening E2 evidence.
- Live retrieval cannot begin until a human completes policy evidence and a
  control-plane operator seals a work order.
- Dynamic public egress still depends on target-host enforcement and an
  operator-reviewed allowlist; application checks are not treated as the only
  boundary.
- E3.2 can test network adversaries offline, but target-host activation and
  real-origin observations remain separate founder-reviewed acts.

## Rejected alternatives

- A visitor-supplied URL endpoint: rejected because it creates arbitrary
  network reachability and SSRF risk.
- Automatic policy approval from robots, terms, or model output: rejected
  because those inputs cannot promote themselves into canonical state.
- A single 25- or 500-record manually edited registry: rejected because it
  obscures per-origin evidence and review ownership.
- Removing private-range denial for convenience: rejected because DNS and
  redirects can change the destination after initial validation.
- Installing the service as part of tests: rejected because repository
  validation must not mutate the VPS or unrelated services.
