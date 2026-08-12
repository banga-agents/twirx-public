# E4 Opportunity source-policy decision proposal

**Prepared at:** 2026-08-12T03:58:08Z

**State:** APPROVED BY GENESIS STEWARD AT 2026-08-12T04:33:44Z

## Proposed decision

```text
origin: grants-gov-api
catalog.state: cataloged
policy.review_state: completed
policy.decision: permit_with_constraints
technical.stage: profiled
publisher.status: unclaimed
health.status: unknown
```

The proposed source is the U.S. Grants.gov daily public XML extract. The
publisher describes it as a daily database export intended for download and
import by power users and database owners. The approved API-use terms permit
searching, displaying, analysing and retrieving grant information, require a
non-endorsement notice, and forbid false representation.

This proposal intentionally does not authorize the documented `search2` or
`fetchOpportunity` POST APIs. Their example responses contain token-shaped
fields, and detailed opportunity responses may contain named contact data.

## Exact proposed retrieval scope

Exactly one immutable source object:

```text
https://prod-grants-gov-chatbot.s3.amazonaws.com/extracts/GrantsDBExtract20260811v2.zip
```

The publisher linked that object from its public XML-extract page as the
August 11, 2026 enhanced daily extract. No filename pattern, index traversal,
later daily file, redirect host, API call, attachment, or arbitrary URL is
authorized.

The acquisition must use exact identity-encoded HTTP byte ranges, with:

```text
execution mode:             manual once
scheduler:                  disabled
concurrency:                1
maximum requests:           96
maximum response per range: 1,048,576 bytes
maximum total transfer:     100,663,296 bytes
maximum reconstructed ZIP:  100,663,296 bytes
maximum decompressed XML:    536,870,912 bytes
maximum source records:     250,000
request timeout:            30 seconds
minimum request interval:   2 seconds
redirects:                  0
```

Every range is an independently content-addressed GET observation. The ZIP is
reconstructed only after every range and its `Content-Range` reconcile. The
final ZIP digest, ordered range manifest, publisher filename, retrieval time,
policy digest and work-order digest are retained. Missing, overlapping,
duplicated, inconsistent or oversized ranges fail closed.

## Approved-field projection

The raw XML is evidence, not a public data product. Only these source-native
fields may enter the deterministic public projection when present:

```text
OpportunityID
OpportunityNumber
OpportunityTitle
OpportunityCategory
CategoryOfFundingActivity
CFDANumbers
EligibleApplicants
AdditionalInformationOnEligibility
AgencyCode
AgencyName
PostDate
CloseDate
LastUpdatedDate
EstimatedSynopsisPostDate
EstimatedSynopsisCloseDate
EstimatedSynopsisCloseDateExplanation
EstimatedAwardDate
EstimatedProjectStartDate
ExpectedNumberOfAwards
EstimatedTotalProgramFunding
AwardCeiling
AwardFloor
ArchiveDate
CostSharingOrMatchingRequirement
Version
```

The projection preserves each admitted native term, lexical value and XML
locator. Dates, amounts and eligibility codes may receive candidate typed
representations only when their exact source syntax validates. Missing fields
remain `not_provided` or `unresolved`; eligibility is never inferred.

## Mandatory exclusions

The following must not enter packets, frames, public snapshots, fixtures,
logs, reports or Git history:

```text
GrantorContactEmail
GrantorContactEmailDescription
GrantorContactName
GrantorContactPhoneNumber
GrantorContactText
applicant accounts or submissions
API tokens or access keys
attachments or full announcement documents
free-form Description
```

The exact raw ZIP may be retained only in private immutable evidence storage
and the independently encrypted Storage Box archive. It must not be committed
or served by the public proof endpoint. The public proof may expose its digest,
source URL, source date, approved projection and deterministic exclusion
report.

## Attribution and representation constraints

Every public view using this source must display:

> This product uses the Grants.gov public data source but is not endorsed or
> certified by the U.S. Department of Health and Human Services.

TWIRX must not modify a source value and still represent the modified value as
the original Grants.gov statement. Typed values and mappings remain visibly
derived and candidate-only. Commercial ranking, eligibility conclusions,
application submission and grant advice are outside this decision.

## Security and operational constraints

- no authentication, API key, cookie, browser, LLM, payment or write action;
- no automatic daily refresh or scheduler job;
- no arbitrary URL, caller-supplied route, filename or range;
- exact host allowlist: `prod-grants-gov-chatbot.s3.amazonaws.com`;
- controlled DNS resolution; private, loopback, link-local, metadata,
  multicast and reserved addresses denied;
- redirects forbidden and TLS/Host verification mandatory;
- evidence before ZIP parsing; no XML external resources or entity expansion;
- ZIP entry count, path, compression ratio and total expanded bytes bounded;
- emergency kill switch and exact work-order revocation remain active;
- a separate founder decision is required for any later file or live refresh.

## Evidence reviewed

- https://www.grants.gov/help/xml-extract/
- https://www.grants.gov/xml-extract
- https://www.grants.gov/system-to-system/grantor-system-to-system/schemas/opportunity-detail
- https://www.grants.gov/api/terms-conditions
- `atlas/e4-sources/grants-gov-api/dossier.json`
- `reports/e4-agent-utility-universe-foundation.md`

## Exact approval text

The Genesis steward may approve only this proposal by stating:

```text
I approve the exact E4 Opportunity policy proposal in
reports/e4-opportunity-policy-decision-proposal.md. Use the current UTC time
as reviewed_at. No route, file, field, retention class or execution beyond
that proposal is authorized.
```

The Genesis steward supplied that exact statement. The completed decision is
bound at `atlas/e4-decisions/grants-gov-20260811/decision.json`. That decision
authorizes only the exact scope above and does not authorize a later daily
file, broader route, scheduler, public raw evidence, or semantic promotion.
