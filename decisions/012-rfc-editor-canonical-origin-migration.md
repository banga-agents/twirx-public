# ADR 012: RFC Editor canonical-origin migration

Status: accepted by Genesis steward on 2026-08-11

## Decision

The Atlas origin `rfc-editor-org` retains its stable identity while its exact
canonical origin changes from `https://rfc-editor.org` to
`https://www.rfc-editor.org` for the FUTO historical archive pilot.

Candidate identity derivation removes one conventional leading `www.` label
before producing the origin ID. This narrowly permits an explicitly reviewed
apex-to-`www` publisher-host migration without manufacturing a second origin.
The existing unique-ID check rejects a selection that tries to include both
forms independently.

The selection, admission source, policy evidence and generated Atlas artifacts
must migrate together. An HTTP redirect is not identity authority and does not
expand any sealed work order.

## Authority

The Genesis steward approved the exact migration and bounded scopes in
`reports/futo-policy-decision-proposals.md`. The approval does not authorize a
live RFC Editor content crawl. While policy review is pending, only a sealed
`robots.txt` evidence-collection order is permitted. After the completed
`profile_only` decision, only the exact Common Crawl periods and route in the
approved archive work order are permitted.

## Security and compatibility

- The origin ID and existing review references remain stable.
- Exact canonical host equality remains required by admission, egress and
  archive work-order validation.
- The normalization is limited to one leading `www.` label; arbitrary aliases,
  registrable-domain inference and redirect-derived identity remain forbidden.
- Collision detection fails closed if apex and `www` forms would otherwise
  coexist.
