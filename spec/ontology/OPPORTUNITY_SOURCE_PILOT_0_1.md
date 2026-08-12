# Opportunity source pilot 0.1

**Authority:** Explanatory E4.5 implementation profile

**Status:** Exact one-shot policy decision completed; acquisition, private
projection, privacy-safe compilation and local release verification complete

This profile describes one source-specific implementation for the public
Grants.gov daily XML extract. It does not change the language-neutral Semantic
Packet, Semantic Frame, Mapping Claim or Universe contracts.

## Authority topology

```text
exact founder-approved policy proposal
  -> exact completed decision
  -> expiring manual-once work order
  -> kill-switch and revocation control
  -> sealed range acquisition
  -> private immutable ZIP evidence
  -> bounded contact-free XML projection
  -> offline packet/frame compilation
  -> candidate semantics only
```

No network operation is authorized by an E4 source dossier or by this profile.
The implementation accepts no caller-supplied URL. It requires the exact
source object, completed decision digest, exact budgets and a currently valid
work order. A disabled control, emergency stop, revocation, expired order or
authority mismatch fails before network access.

## Sealed acquisition

The approved acquisition profile used sequential
one-MiB identity-encoded byte ranges. Each range response and its evidence are
stored before the reconstructed archive receives those bytes. Publication is
complete only when the acquisition manifest is written last.

Verification MUST reconcile:

- the exact HTTPS URL, host, path and filename;
- every requested and returned byte range;
- status 206, media type and zero redirects;
- `Content-Range`, total length and range continuity;
- every range digest against the reconstructed archive;
- work-order and human-decision digests;
- request, byte, time and record budgets;
- disabled scheduler and private raw-evidence state.

The trusted projection/parser path has no network, process, plugin, cgo or
`unsafe` authority.

## Private evidence and approved private projection

The exact ZIP, contained XML and approved projection are private evidence. The projection contains
only the fields in the approved policy proposal. Contact fields, free-form
description, attachments, applicant accounts, submissions and credentials
MUST NOT enter packets, frames, public snapshots, fixtures, logs, reports or
Git history.

The ZIP reader rejects traversal paths, directories, encryption, unapproved
entry types, duplicate names, excessive entry counts, oversized expansion and
excessive compression ratios. The XML reader rejects DTDs, directives, entity
expansion, unexpected nesting, duplicate non-repeatable fields, overlong
scalars and invalid source-specific dates or numeric strings.

Projection verification permits exactly three artifacts:

```text
approved-projection.json
projection-report.json
projection-manifest.json
```

The manifest is written last and binds the private acquisition manifest,
source archive, source XML, private projection and exclusion report.

## Semantic compilation

The offline compiler preserves:

- the source-native term and lexical value;
- the exact XML record and field locator;
- the raw XML representation digest;
- the observation, policy, module and adapter derivation;
- explicit `not_provided` and `unresolved` frame slots.

Dates published without an exact instant remain native date strings and do
not become invented timestamps. Amounts receive a decimal typed value only
when their source syntax validates; no currency is inferred. Eligibility codes
remain source identifiers, and no eligibility conclusion is generated.

Every mapping remains `candidate` with no review-decision digest. Every frame
remains in the `provisional_semantic` lane until a separate semantic review.

## Public release profile

The local E4.5 release identifier is `tw.e4-opportunity-release/0.1`. It binds
the exact one-shot acquisition and projection digests, the Opportunity module
set, every packet and mapping segment, an Opportunity frame segment, a
combined World State plus Opportunity query segment, and a privacy report.
The manifest is written last and its detached SHA-256 is the release identity.

Public artifact segments use `tw.artifact-segment/0.1` with magic `TWAS0001`.
They contain one artifact class only, sort entries by detached digest, reject
duplicates, and bind each canonical CBOR body to its digest. The compact query
segment remains bounded to 1,000 public results. Complete release admission
uses the separate integrity-only canonical frame visitor and does not broaden
that public query limit.

The allowed eligibility field contained contact-like material in the real
projection. Therefore every non-empty `AdditionalInformationOnEligibility`
value becomes a proof-linked packet whose native status is `withheld`, whose
native lexical value is empty and whose typed value is absent. Empty source
fields become `not_provided`. Frames link those packets and preserve the
status. The public release includes no eligibility lexical value.

The complete Go verifier rehashes and parses every public constituent,
reconciles every frame-to-packet reference and rejects unused packets. The
restricted-C verifier independently checks a deterministic sample whose
selection policy and release identity are committed. This is explicitly not a
claim that C verified the entire million-packet corpus.

## Public attribution

Any future public surface using this source MUST include the non-endorsement
notice frozen in the exact human policy decision and visibly distinguish the
publisher's values from TWIRX-derived types and candidate mappings.

## Current exclusions

- no scheduler or automatic daily refresh;
- no Grants.gov POST API;
- no contact or full-description publication;
- no browser, model, authentication, payment, application or write action;
- no arbitrary URL, route, filename, port or redirect;
- no automatic mapping or canon promotion;
- no public deployment before founder review and off-host release admission;
- no inference that a source record is currently open or that any applicant is eligible.
