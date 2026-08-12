# E4 importer fixtures

These representations exercise source-specific parsers without authorizing
network retrieval or making public-source corpus claims.

- `grants-fetch-controlled.json` follows the documented Grants.gov
  `fetchOpportunity` response shape but contains invented controlled data,
  an empty token field and no personal contact fields. It is a `test_fixture`,
  not a statement made by Grants.gov.
- The tracked World Bank E2 fixture remains under `origins/fixtures/` and is
  likewise counted only as controlled replay evidence in E4 importer tests.

Real source evidence requires an admitted work order, immutable stored
representation, exact policy-decision digest and a release report that keeps
real, archive, replay and fixture counts separate.
