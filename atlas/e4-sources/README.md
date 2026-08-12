# E4 source policy and capacity dossiers

Each directory contains one source scope. Dossiers are policy evidence, not
work orders. Network execution is disabled by default and pending dossiers
grant no retrieval authority. The Grants.gov dossier is the sole exception in
this directory with a completed exact E4 decision and one consumed manual-once
work order; its scheduler remains disabled and no later file is authorized.

The World Bank dossier records the already approved E2 route as historical
scope but does not broaden it into E4 bulk retrieval. The other dossiers record
publisher documentation and proposed limits only. A future retrieval requires
a completed, exact source-specific policy decision and a separately sealed
work order through the existing egress boundary.

Normal tests validate all dossiers without public internet access.

The Grants.gov dossier has an exact approved policy proposal at
`reports/e4-opportunity-policy-decision-proposal.md`, a completed decision at
`atlas/e4-decisions/grants-gov-20260811/decision.json` and a consumed
manual-once work order. Its acquisition, private projection, privacy-safe
public compilation and release implementation are documented in
`spec/ontology/OPPORTUNITY_SOURCE_PILOT_0_1.md`.
