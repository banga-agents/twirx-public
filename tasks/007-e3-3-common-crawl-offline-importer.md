# Task 007: E3.3 sealed Common Crawl offline importer

## Status

Implemented and used for the steward-approved RFC Editor pilot. Two exact
Common Crawl captures are admitted as historical archive evidence. The later
archive-to-packet and delta admission is tracked by Task 009; this task still
grants the offline importer no network authority.

## Objective

Turn separately acquired, bounded Common Crawl index and WARC-range artifacts
into an immutable, independently re-verifiable historical evidence spool. The
tool must accept only a sealed work order tied to completed human policy
review and must preserve raw evidence before any WARC or HTTP parsing.

## Required outputs

- strict `tw.common-crawl-work-order/0.2` validation with each approved
  capture's collection, route, timestamp, provider digest, WARC path and byte
  range bound before acquisition;
- exact official index/data URL and range derivation;
- bounded JSON-lines index parser with duplicate and ambiguity rejection;
- bounded single-member gzip, WARC and archived HTTP parsing;
- provider SHA-1 metadata verification while retaining SHA-256 for every
  TWIRX artifact identity;
- exact `206` and `Content-Range` evidence;
- raw-evidence-manifest-before-parsing and final-manifest-last publication;
- complete spool rehash, reparse and cross-artifact reconciliation;
- offline planner, inspection, import and verification CLI;
- malformed, adversarial and fuzz coverage;
- no network client, scheduler, arbitrary URL, policy approval or semantic
  canon admission.

## Acceptance

The final evidence spool is accepted only when its work-order authority,
selected index record, range-response evidence, compressed WARC record,
archived HTTP representation and derived metadata all reconcile. Any absent,
oversized, malformed, duplicated, ambiguous, redirected, cross-origin,
wrong-range, digest-mismatched or trailing input fails closed.

The Genesis steward supplied the required narrow authority on 2026-08-11.
That authority applies only to the exact sealed RFC Editor work order and does
not authorize another origin, route, collection, capture, range, or scheduler.
