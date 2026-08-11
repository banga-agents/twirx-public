# Secure public-origin egress pilot 0.1

**Authority:** Normative for the E3.2 sealed retrieval profile

**Status:** Implemented and adversarially tested offline; disabled,
unactivated, and not admitted for real-origin retrieval

## Purpose

This profile defines how an already reviewed, read-only public-origin route
may be retrieved outside the Atlas control plane without creating an
arbitrary-URL capability. The data model and failure behavior are
language-neutral. A particular worker or service manager is an implementation,
not protocol authority.

## Authority topology

```text
human catalog admission
        +
completed retrieval-permitting policy review
        +
exact policy and decision digests
        ↓
sealed work order
        ↓
isolated egress worker
        ↓
immutable evidence spool
        ↓
manifest-last admission
```

Neither origin content, a model, a browser, a parser, nor the egress worker
can issue a work order or promote state.

## Work order

`tw.egress-work-order/0.1` binds:

- a unique order and canonical origin ID;
- purpose `robots`, `profile`, or `observation`;
- authority class `policy_evidence_collection` or `reviewed_policy`;
- credential-free HTTPS `GET` route and exact host allowlist;
- retrieval-permitting policy decision;
- policy-evidence and human-decision SHA-256 digests;
- approval reference and canonical validity interval;
- redirect, decompressed-byte, request, connect, and header-time limits;
- consecutive-failure threshold and circuit cooldown.

Only the ID crosses the execution interface. A URL supplied by the caller is
not an input. `policy_evidence_collection` exists to avoid a review cycle: it
allows only an explicitly human-approved `/robots.txt` request for a
canonically admitted origin while policy is `pending + uncertain`. It cannot
authorize a profile or observation. `reviewed_policy` requires a completed
retrieval-permitting decision, and `profile_only` cannot authorize
`observation`.

## Destination validation

The initial URL and every redirect require the admitted hostname, HTTPS,
standard port, no credentials, and canonical syntax. Numeric IP literals and
alternate numeric host encodings are forbidden.

DNS is resolved for every connection. Every returned address is normalized
and checked before dialing. Private, loopback, link-local, metadata,
multicast, unspecified, carrier-grade NAT, documentation, benchmarking,
discard-only, and reserved IPv4 and IPv6 ranges are denied. No decision from
an earlier DNS answer can authorize a later connection.

TLS certificate and host verification remain enabled. Response bytes are
limited after decompression. Redirect count and wall time are bounded.

## Publication

The worker preserves the exact work order before retrieval. A successful run
then publishes:

```text
work-order.json
cas/sha256/.../BODY_DIGEST
evidence/body.ref
evidence/observation.cbor
evidence/observation.json
result.json
manifest.json                  written last
```

No parsing or adapter extraction occurs in this worker. `manifest.json`
contains a sorted list of exact artifact digests. A spool is admitted only
when the final manifest exists, every entry rehashes, the result and canonical
observation agree, and the CAS body matches its digest and declared size.

## Revocation and resource control

A root-owned control artifact provides global disable, emergency stop, and
sorted origin/order revocation. Invalid, expired, revoked, disabled, or
circuit-open orders fail before retrieval. A process lease sets pilot global
concurrency to one. A deployment must also bound memory, CPU, task count, file
descriptors, and execution time and must provide network-level denied-range
enforcement independent of application checks.

The worker must not see registry write paths, databases, deployment state,
home directories, credentials, or secrets. It may write only immutable spool
artifacts and bounded circuit/concurrency state.

## Failure behavior

Unknown fields, duplicate keys, trailing JSON, symlinks, substituted work
orders, unsafe paths, invalid validity intervals, policy mismatch, forbidden
destinations, DNS rebinding, denied redirects, TLS failure, redirect loops,
oversized decompressed bodies, timeouts, concurrent execution, revocation,
open circuits, missing manifests, missing artifacts, and digest disagreement
fail closed.

## Conformance

```bash
go test ./internal/safefetch ./internal/egressworker ./cmd/twirx-egress-worker
go test -run='^$' -fuzz='^FuzzWorkOrderJSON$' -fuzztime=1s ./internal/egressworker
systemd-analyze verify deploy/egress/twirx-egress-worker@.service
```

Conformance requires offline adversarial coverage for both IPv4 and IPv6
denied ranges, DNS rebinding, redirects to denied addresses, numeric address
forms, credentials, schemes, ports, decompression limits, redirect loops,
TLS verification, control-plane destinations, revocation, global disable,
circuit breaking, immutable publication, and evidence tampering.
