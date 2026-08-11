# FUTO policy-decision proposals

**Status:** HUMAN REVIEW REQUIRED — this document grants no retrieval or
admission authority

**Prepared:** 2026-08-11

## Purpose

This review sheet translates the FUTO pack's three founder templates into the
repository's canonical origin IDs and orthogonal admission vocabulary. Codex
has prepared the scope and known evidence. Only Richi, acting as Genesis
steward, may approve, amend or reject the decisions.

An approval must be explicit. Until then, all three source artifacts retain
their present policy states: `pending + uncertain`; the scheduler remains
disabled; no acquisition work order exists.

## Proposal 1 — TWIRX publisher-authored origin

```text
origin_id:       twirx-org
canonical:       https://twirx.org
proposed state:  completed + permit_live
scope:           publisher-authored project status, public reports,
                 public funding status and published interface declarations
effect:          public_read only
```

Proposed constraints:

- read-only;
- exact publisher-authored public routes only;
- no private repository content, personal data or unpublished reports;
- preserve project attribution and source-native values;
- scheduler remains disabled for this FUTO release;
- current publication is not promoted to objective truth;
- no authenticated, write, payment, browser or model action.

Existing evidence:

- `CHARTER.md`;
- `decisions/005-twirx-origin-admission.md`;
- `contracts/e2/contracts.json`;
- `origins/catalog.json`;
- `origins/fixtures/twirx-project-status.json`;
- `reports/gate-e2-live-provenance-lab.md`.

Evidence still required before the repository validator can record the
completed retrieval-permitting decision:

- an observed and digest-bound `https://twirx.org/robots.txt` result, or an
  explicitly evidenced unavailable result;
- a final human terms/risk determination;
- canonical review timestamp and reviewer statement.

## Proposal 2 — World Bank Indicators

```text
origin_id:       api-worldbank-org
execution alias: world-bank-indicators
canonical:       https://api.worldbank.org
proposed state:  completed + permit_with_constraints
scope:           documented E2 Indicators API route family only
effect:          public_read only
```

Proposed constraints:

- use the existing bounded E2 country/indicator/year route only;
- identify TWIRX, preserve World Bank attribution and cache responses;
- obey published request and concurrency limits;
- no broad crawl, undocumented endpoint discovery or authentication;
- preserve provider wording and derivation; do not claim objective truth;
- scheduler remains disabled for this FUTO release;
- no write, payment, browser or model action.

Existing evidence:

- `contracts/e2/contracts.json`;
- `origins/catalog.json`;
- `origins/fixtures/world-bank-chl-population-2024.json`;
- `reports/gate-e2-live-provenance-lab.md`;
- the official Indicators API documentation currently referenced in the
  admission dossier.

Evidence still required before the repository validator can record the
completed retrieval-permitting decision:

- an observed and digest-bound `https://api.worldbank.org/robots.txt` result,
  or an explicitly evidenced unavailable result;
- final human confirmation of the documented API terms, attribution, rate,
  retention and origin-specific risk treatment;
- canonical review timestamp and reviewer statement.

## Proposal 3 — RFC Editor historical archive pilot

```text
origin_id:       rfc-editor-org
proposed canonical host: https://www.rfc-editor.org
proposed state:  completed + profile_only
scope:           exact homepage captures in two sealed Common Crawl periods
effect:          historical archive observation only
```

Why this candidate:

- public-interest standards infrastructure;
- no authentication or personal-data use is proposed;
- exact homepage records exist in two bounded archive periods;
- the page is suitable for a visible historical representation comparison;
- archive classification prevents a historical capture from becoming a
  current publisher statement.

Proposed exact archive periods and candidate captures discovered during
operator preparation:

```text
CC-MAIN-2025-30
route:      https://www.rfc-editor.org/
timestamp:  20250708102138
length:     15237 bytes
offset:     802952064
WARC:       crawl-data/CC-MAIN-2025-30/segments/
            1751905933639.15/warc/
            CC-MAIN-20250708102057-20250708132057-00647.warc.gz
provider:   L6XIA742BVM752M3MXQUXL3JB5M6Y75M

CC-MAIN-2026-25
route:      https://www.rfc-editor.org/
timestamp:  20260607205341
length:     36598 bytes
offset:     830637590
WARC:       crawl-data/CC-MAIN-2026-25/segments/
            1780687572337.15/warc/
            CC-MAIN-20260607191654-20260607221654-00647.warc.gz
provider:   5EZCTVCDHO5Y5HWQYAA5WBGRTHABTG4H
```

Proposed constraints:

- exact two periods, exact homepage route and at most one capture per period;
- historical archive evidence only;
- `current_publisher_statement: false`;
- no live content crawl, authenticated content or personal-data extraction;
- bounded CDX response, byte ranges, WARC bodies and retained representation;
- preserve RFC Editor attribution and source-native lexical values;
- no scheduler, browser, model, payment, action or canon promotion;
- a completed acquisition manifest still does not admit semantic mappings.

Identity issue requiring founder decision:

The selected Genesis-500 record currently uses `https://rfc-editor.org`, while
the exact usable captures use `https://www.rfc-editor.org/`. The founder must
approve migration of the canonical origin to `https://www.rfc-editor.org`
before any work order can bind those captures. The selection, admission source
and all derived canonical artifacts must change together; an alias or redirect
must not silently expand authority.

Evidence still required before the repository validator can record the
completed retrieval-permitting decision:

- explicit human catalog admission and canonical-host migration;
- a reviewed robots outcome bound to the approved canonical origin;
- final human terms, attribution, retention and risk determination;
- canonical review timestamp and reviewer statement.

## Exact founder response requested

Richi may approve the proposal without retyping it by stating:

```text
I approve the three FUTO policy proposals in
reports/futo-policy-decision-proposals.md as Genesis steward, including the
RFC Editor canonical-origin migration to https://www.rfc-editor.org. Use the
current UTC time as reviewed_at. The approved scopes and constraints are
exactly those in the proposal; no broader retrieval is authorized.
```

Any amendment should name the affected origin, field and replacement value.
Approval authorizes Codex to materialize the human decision artifacts and the
bounded policy-evidence work orders. It does not authorize a scheduler, broad
crawl, arbitrary URL, live RFC Editor crawl, semantic mapping or deployment.
