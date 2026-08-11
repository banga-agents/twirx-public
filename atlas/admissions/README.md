# E3.2 per-origin admission sources

Each child directory is the review source for one Genesis public-origin
dossier. The canonical registry is rendered from these directories; it is not
the place to author a new record.

```text
{origin_id}/record.json
{origin_id}/policy-evidence.json
{origin_id}/decision.json
{origin_id}/evidence/*
```

Run:

```bash
bin/twirx-admission validate --root . --admissions atlas/admissions
bin/twirx-admission atlas-queue --root . --admissions atlas/admissions
bin/twirx-admission review-queue --root . --admissions atlas/admissions
bin/twirx-admission check-canonical --root . --admissions atlas/admissions
```

The derived Atlas queue always covers the exact 500-origin selection and makes
each unprepared dossier visible without generating an approval or duplicating
candidate data into the canonical registry. The current 25 directories are a
pilot batch. `twirx-org` and
`api-worldbank-org` import previously admitted E2 state and evidence. The
other 23 records are agent-prepared catalog dossiers awaiting founder review.
Their decision artifacts say `review_state: pending`, their policy is
`pending + uncertain`, and they cannot render into canonical state.

The pilot contains five provisional commercial/access candidates:
`latimes-com`, `lemonde-fr`, `nytimes-com`, `reuters-com`, and
`washingtonpost-com`. Their digest-bound notes are inputs to human policy
review, not live observations. The records do not claim a current offer,
price, payment protocol, access permission, or executable operation.

## Human review procedure

1. Confirm the canonical origin, registrable domain, publisher identity,
   jurisdiction, languages, domain family, authority class, and aliases.
2. Add bounded local evidence files and their SHA-256 digests to
   `policy-evidence.json`.
3. Review robots, terms, attribution, authentication, rate, retention, access
   and economic declarations, and origin-specific risk. Robots preferences
   are not access authorization.
4. Record one of the six policy decisions, including `uncertain` when the
   evidence does not support a stronger decision.
5. Put the human reviewer, canonical review time, rationale, constraints, and
   exact approval reference in `decision.json`.
6. Rerun validation and inspect the generated dossier before considering a
   canonical update.

A catalog admission and a policy decision are independent. For an already
human-admitted origin with pending policy, a separately approved sealed
`policy_evidence_collection` order may retrieve only `/robots.txt`. It cannot
profile or observe content. Broader retrieval requires a completed
retrieval-permitting policy.

No command in the admission factory creates a review decision, seals a live
route from visitor input, promotes a model result, enables the scheduler, or
deploys the worker.
