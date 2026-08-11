# FUTO F0 engineering integration evidence

Date: 2026-08-11

Status: **PASS for draft founder review; not merged or deployed**

Tested commit: `ef648d29303168f9676b2586b0fc96efd3fe76e2`

Base: `origin/main` at `669b20506c82484942d5d8e907b17d8061312be3`

Draft review: <https://github.com/banga-agents/TWIRX/pull/16>

## Outcome

The eight tested Semantic Data Plane, immutable snapshot, controlled-scale and
sealed archive-import commits were pushed without force and opened as a draft
pull request against `main`. The untracked website, reports and architecture
packs were not staged or published.

The stranded PR 15 commits were reviewed file by file. ADR 011 records the
retain/supersede/port/reject matrix and establishes `tw.semantic-query/0.1` in
`schemas/cddl/semantic-data-plane.cddl` as the sole authoritative query
contract for the release. PR 15's parallel JSON Semantic Request and Access
Plan objects were not imported as wire authority. Its fixed weighted scalar
ranking was rejected in favor of ADR 008's visible Pareto dimensions and
explicit caller preference.

Useful PR 15 invariants remain present in the accepted tree: hard filters
precede preference; natural language has no execution authority; compact agent
operations replace origin-specific tool proliferation; sponsorship cannot buy
semantic rank; origin capability and offer metadata remain evidence-bound and
non-executable; and arbitrary URL, MCP, browser, authentication, payment and
write paths remain forbidden.

## Diff introduced by reconciliation

- `decisions/011-pr15-resolver-reconciliation.md`: authoritative schema choice,
  area and file-level disposition, compatibility/security effects and reversal
  conditions.
- `tasks/004-e3-3-semantic-data-plane.md`: adds ADR 011 to the required reading
  order and states the single-query-family rule.

No production, verifier, adapter, origin, workflow, dependency or deployment
file changed during reconciliation.

## Exact validation

```bash
GOMAXPROCS=2 make test
GOMAXPROCS=2 go test -race ./...
GOMAXPROCS=2 go vet ./...
git diff --check
```

Results:

- all Go package and offline integration tests passed;
- all 21 one-second Go fuzz targets passed;
- the observation restricted-C verifier accepted two valid vectors, rejected
  14 invalid vectors and rejected corrupted evidence under ASan/UBSan;
- E2 shared restricted-C result and artifact conformance passed;
- E3.3 S1 restricted-C conformance passed: 56 total, 16 accepted and 40
  rejected;
- the observation, E2 and data-plane libFuzzer targets each completed 5,000
  runs without a crash;
- the end-to-end source-statement and Semantic Snapshot integrations passed;
- the snapshot integration retained 18 packets, two public origins, excluded
  fixtures and made zero origin-network requests;
- documentation configuration/navigation, race detection, vet and whitespace
  checks passed.

Normal validation used no public-origin network access.

## Preserved invariants

- Language-neutral specifications and conformance remain protocol authority.
- Native source terms, locators and lexical values survive semantic mapping.
- Missing required evidence fails closed; optional absence stays explicit.
- Canonical cores remain self-digest-free and manifests publish last.
- Origin, semantic and canon deltas remain distinct.
- Atlas state dimensions, adapter trust and mapping trust remain orthogonal.
- No generated object, model, adapter or natural-language layer can admit
  itself or grant execution authority.
- Genesis remains read-only and browser-, model-, payment- and action-free.
- E1, E2 and E3.2 implementation behavior is unchanged.

## Unresolved risks and blockers

- PR 16 remains a draft and requires founder review; it is not merged.
- The public sanitized repository has not yet been created.
- Zero human origin-policy decisions are complete.
- No Common Crawl network acquisition or real archive import has occurred.
- No genuine origin delta or public Query Lab exists.
- The untracked website source is outside this PR and is not bound to this
  tested commit.
- GitHub-hosted CI has not run and is not claimed.

## Deviations

PR 15 was not cherry-picked verbatim. That would have violated the explicit
one-contract-family requirement by importing duplicate normative JSON request
objects and a route-centric weighted ranking policy. ADR 011 instead preserves
the compatible invariants and records exact supersession.

## Next gate

Founder review of draft PR 16, followed by F1 construction and independent
secret/history audit of a fresh public release repository. No public
repository, merge or deployment should precede that review.

## Post-review integration update

PR 16 was subsequently merged into `main` as
`fbf1d6a019f4f1c263f5ae92997c5f21887550c9`. This update records the later
repository state without changing the evidence or conclusions that applied at
the original report time. The FUTO policy decisions, real archive acquisition,
archive packet/delta compiler and final readiness audit are follow-up work and
remain subject to draft founder review in
<https://github.com/banga-agents/TWIRX/pull/17>.
