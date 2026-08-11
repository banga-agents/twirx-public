# FUTO documentation synchronization

**Status:** SOURCE PASS — Mintlify deployment skipped for the draft PR;
production verification requires merge or an explicitly authorized preview.

**Sync date:** 2026-08-11

**Documentation revision:**
`633f34a`

**Evidence snapshot:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`

## Result

The tracked Mintlify documentation now matches the FUTO release candidate and
the deployed institutional website. It no longer reports zero policy reviews,
two public registry records, or a standalone USD 6,000 Genesis target.

The start page now explains the project as a language-neutral Semantic Data
Plane: origin representations become immutable observations, source-native
statements, semantic packets, materialized read-only state and typed query
results with provenance.

A dedicated FUTO release page records the exact distinctions:

- 500 selected Atlas identities are not 500 working adapters;
- 15 public-source packets are separate from five fixture packets;
- the 25,018-packet result is fixture-dominated capacity evidence;
- one real archive-origin delta is not a semantic or canon delta;
- the runtime performs zero origin calls at query time;
- the public Query Lab serves one immutable snapshot through its HTTPS edge.

The protocol documentation now includes the immutable Semantic Snapshot
runtime, candidate endpoints, invariants, failure behavior, host limits,
conformance scope and immutable-public status. Atlas, security, roadmap,
quickstart and funding pages all use the same release facts.

## Files changed

- `docs/docs.json`
- `docs/index.mdx`
- `docs/start/futo-release.mdx`
- `docs/start/genesis-status.mdx`
- `docs/start/quickstart.mdx`
- `docs/protocol/semantic-snapshot-runtime.mdx`
- `docs/protocol/atlas-control-plane.mdx`
- `docs/security/live-lab-boundary.mdx`
- `docs/governance/funding-transparency.mdx`
- `docs/governance/roadmap.mdx`

## Commands executed

```bash
scripts/check-docs.sh
python3 -m json.tool docs/docs.json
rg -n \
  "zero completed|zero policy|two cataloged|two public origins|498 catalog|500 policy|Genesis target is USD|not deployed|has not been deployed" \
  docs
git diff --check
```

Results:

- Mintlify configuration parsed: PASS;
- every navigation target exists: PASS;
- JSON syntax: PASS;
- known stale release-language scan: zero match;
- whitespace validation: PASS.

## Invariants preserved

- protocol authority remains language-neutral;
- provider content is described as an origin representation, not objective
  truth;
- archive observations are labeled historical and not current publisher
  statements;
- implemented, public and planned behavior remain separate;
- no arbitrary URL, browser, model, payment, authentication or write authority
  is implied;
- no dependency was added to the repository.

## Unresolved risk

The source change is pushed to draft PR 17 and passes the documentation checks,
but `https://docs.twirx.org/start/futo-release` currently returns `404`. The
external Mintlify production deployment is therefore not claimed synchronized.

## Next recommended gate

After founder review, merge the documentation source or authorize an external
production deployment. Then verify the live FUTO release, snapshot-runtime,
funding, roadmap and Genesis status pages at `https://docs.twirx.org`.
