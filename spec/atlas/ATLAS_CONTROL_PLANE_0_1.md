# Atlas control plane 0.3

**Authority:** Normative for the E3 offline control plane

**Status:** Orthogonal state and admission factory implemented; no public-origin retrieval admitted

## Purpose

The control plane separates candidate selection, canonical identity, policy,
technical progress, publisher relationship, health, adapter trust and mapping
trust. It validates an exact 500-candidate public-knowledge universe and
derives public counters without granting origin-fetch capability.

The JSON schemas and this document are language-neutral. Any implementation is
non-normative unless admitted through the conformance process.

## Artifacts

```text
atlas/genesis-500/selection.json    exact candidate universe
atlas/admissions/{origin_id}/       per-public-origin source artifacts
atlas/fixtures/{fixture_id}/        controlled fixture source artifacts
atlas/policies.json                 exact policy artifact set
atlas/registry.json                 canonical identities and independent state
origins/catalog.json                E2 bounded execution configuration
generated/e3/admission/             deterministic dossiers and render
generated/e3/admission/atlas-queue.json derived 500-origin work queue
generated/e3/atlas-metrics.json     deterministic derived counts
```

The selection is never a destination allowlist. `atlas/registry.json` is the
canonical identity and state registry. Each E2 execution entry must name a
canonical `registry_id`; host, publisher, scope and alias mismatch fails
closed. The E2 catalog still owns its narrower operation routes and typed
inputs and does not obtain Atlas scheduler authority from this identity link.

Each origin record also carries future-compatible descriptive surfaces:

```text
interfaces
capability candidates + effect classes
access and provisional offer metadata
operational economics
publisher-readiness signals
```

These fields do not grant route authority. An interface declaration can be
publisher-declared, observed, inferred, or archive-derived while remaining
`descriptive_only` or `not_admitted`. E3.2 accepts `admitted` only for an
existing compiled `public_read` operation whose exact interface is already
admitted by E2. WebMCP, browser, authenticated, write, financial, legal,
material, and destructive declarations remain non-executable.

Every non-unknown descriptor is SHA-256-bound to an immutable artifact in the
origin record. Publisher-readiness signals record their source, observation
class, timestamp, freshness, and standard status. Draft, proposed, or
experimental conventions cannot be described as finalized.

## Orthogonal origin state

The active state vocabulary is:

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

Technical stage is ordered only inside its own dimension. No technical state
completes policy review, verifies a publisher, establishes current health or
grants trust to an adapter or mapping. Every non-default state requires its own
dated, human-identifiable evidence where applicable.

`policy.review_state: pending` fails closed and never contributes to completed
policy counts. Its decision is `uncertain`. A completed uncertain decision
requires explicit reviewer and review time. `review_required` is not an active
state value.

## Current registry

The registry imports two qualifying public E2 origins:

- `twirx-org`;
- `api-worldbank-org`.

Their immutable replay representations, contracts, conformance vectors,
generated proof metadata and semantic-closure digests support independent
technical and trust state. Their separate Atlas policy reviews remain pending
and uncertain. TWIRX publisher approval is recorded separately.

The controlled E2 origin has scope `test_fixture`. It is excluded from every
Genesis-500 public-origin state, coverage and semantic count.

## Admission factory

A public-origin directory contains `record.json`, `policy-evidence.json`,
`decision.json`, and bounded evidence files. The factory:

- revalidates origin identity against the exact candidate selection;
- hashes every local policy-evidence file;
- rejects duplicate canonical origins, IDs, and execution aliases;
- validates every orthogonal state independently;
- derives one dossier per origin and one batch report;
- renders only records with explicit completed human admission artifacts;
- preserves controlled fixtures from their separate source directories;
- does not create, recommend, or alter a policy decision.

An explicit human catalog admission and a completed policy review are separate
facts. A record can be canonically cataloged while policy remains
`pending + uncertain`. Such a record cannot produce a retrieval work order.

The E3.2 pilot directory contains 25 catalog dossiers, including the two
previously admitted public E2 origins. Twenty-three are agent-prepared review
proposals and do not render into canonical state. All 25 policy reviews remain
pending.

Five pilot dossiers are provisional subscription/commercial candidates:
`latimes-com`, `lemonde-fr`, `nytimes-com`, `reuters-com`, and
`washingtonpost-com`. Their classification is inferred from a digest-bound
catalog-review proposal, not a fetched publisher representation. No current
price, access term, payment protocol, offer validity, policy permission, or
executable commercial capability is claimed.

The derived `tw.atlas-admission-work-queue/0.1` covers all 500 selected
origins. It reports a selected origin with no per-origin source directory as
`dossier_state: not_prepared` and assigns the next action
`prepare_dossier`. It does not manufacture placeholder evidence. A prepared
dossier retains its explicit admission and policy states and obtains a next
action from those states. The queue is generated data, not an authoring
surface or canonical registry.

## Policy assessment

Every bound policy reproduces the registry origin ID and canonical origin and
binds the exact SHA-256 of the policy-set bytes. A completed policy records:

- explicit human reviewer and canonical UTC review time;
- robots outcome, crawler product token and evidence digest when retrieved;
- terms, attribution, authentication, rate, retention and risk decisions;
- sorted evidence references.

A retrieval-permitting decision requires successful or explicitly unavailable
robots assessment, completed terms review, no authentication and accepted
risk. Robots rules remain crawler requests and are not access authorization.

## Dry-run frontier

`twirx-atlas plan` requires a caller-supplied canonical UTC time and emits a
deterministic `tw.atlas-frontier-plan/0.1` document containing origin IDs and
budgets, never destination URLs. It performs no fetch and always states:

```json
{
  "mode": "dry_run",
  "network_access": "disabled"
}
```

The frontier emits exactly one decision for every selected origin. An origin
without a canonical catalog record is blocked as `catalog_review_pending`; a
cataloged origin with pending policy is blocked as `policy_review_pending`.
Only a public origin with completed `permit_live` or
`permit_with_constraints`, positive bounded budgets and an active scheduler
can become a dry-run job. Pending, denied, catalog-only, profile-only,
uncertain, disabled, cooling-down and not-yet-due origins are blocked or
deferred visibly. The committed state has zero jobs.

## Counting model

Metrics publish exact maps for every independent dimension. A separate
`technical_at_or_beyond` map is valid because `technical.stage` alone is an
ordered pipeline. Selection candidates default to pending/uncertain policy,
unprofiled technical state, unclaimed publisher, unknown health and no adapter
or mapping trust until canonical evidence overrides that state.

Fixture counts are separate. Fixtures do not contribute to the 500-origin
state, coverage, semantic or learning counters.

Capability counters separately report interface kinds, effect classes,
candidate status, admitted public-read operations, access class, provisional
commercial offers, publisher-native declarations, machine-readable payment
declarations, readiness signals, and measured economics. The canonical public
counter currently reports zero commercial offers because the five provisional
records are not admitted; the internal admission-batch counter reports five.

## Invariants

- exactly 500 unique candidate IDs and canonical credential-free HTTPS origins;
- candidate IDs are derived from the canonical host after removing one
  conventional leading `www.` label, preserving identity across an explicitly
  reviewed apex-to-`www` publisher-host migration while duplicate-ID checks
  prevent both forms from entering the universe separately;
- exact domain-family quotas from ADR 004;
- candidate identity hints are explicit null or empty values;
- public registry records preserve a selected identity;
- public registry records preserve their selected primary domain family;
- execution-catalog aliases are unique and canonically bound;
- only an explicit completed human admission artifact renders canonical state;
- pending proposals never render canonical state;
- pending policy does not count as completed;
- every non-default state carries its own required evidence;
- descriptive interface, capability, access, offer, economics, and readiness
  state cannot become route authority;
- only existing compiled public-read E2 operations may have E3.2 `admitted`
  capability status;
- repository evidence paths are relative, bounded regular files and match
  their recorded SHA-256 digests;
- scheduler state, priority factors, failure count, cooldown and budgets are
  explicit and bounded;
- metrics are regenerated from validated artifacts;
- the admission work queue and dry-run frontier each cover all 500 selected
  origins exactly once;
- WSIM readiness remains false unless every documented corpus threshold passes.

## Failure behavior

Unknown fields, duplicate JSON keys, trailing data, malformed URLs, duplicate
identities or aliases, quota drift, missing state evidence, false human-review
claims, policy mismatch,
unsafe evidence paths, digest substitution, unsafe scheduler state, invalid
timestamps and oversized documents fail closed with context. The HTTP surface
rejects non-GET methods, arbitrary-URL input, unknown or repeated filters and
oversized results.

## Security considerations

The Atlas control process contains no HTTP client and its server requires a
literal loopback listener. Candidate content cannot write policy, registry or
canonical state. A dry-run frontier is not egress authority. The separate
local-fixture worker proves evidence-first retrieval only against literal
`127.0.0.1`; it grants no selected-origin network authority. The separate
sealed egress candidate is specified in `SECURE_EGRESS_PILOT_0_1.md` and
remains disabled and unactivated.

## Conformance

```bash
bin/twirx-atlas validate --root .
bin/twirx-atlas metrics --root .
bin/twirx-atlas plan --root . --at 2026-08-11T00:00:00Z
bin/twirx-atlas stress --root . --at 2026-08-11T00:00:00Z --rounds 1 --workers 8
bin/twirx-admission validate --root . --admissions atlas/admissions
bin/twirx-admission atlas-queue --root . --admissions atlas/admissions
bin/twirx-admission check-canonical --root . --admissions atlas/admissions
go test ./internal/atlas ./internal/admission ./internal/atlasapi ./internal/atlasstress ./internal/origincatalog ./cmd/twirx-atlas ./cmd/twirx-admission
```

Conforming implementations derive the same selection and policy-set digests,
orthogonal state counters, family counts, frontier decisions and readiness
state from the committed artifacts.
