# Observatory local-fixture worker 0.1

**Authority:** Normative for the E3 process-boundary fixture profile

**Status:** Implemented local-fixture proof; public-origin retrieval is not
implemented or authorized

## Purpose

This profile proves that retrieval can occur outside the Atlas control plane,
that retrieved bytes become immutable evidence before an untrusted parser sees
them, and that the resulting decision can be replayed without a network
source. It does not define a general crawler or production egress sandbox.

The JSON schemas and invariants in this document are language-neutral. The
Genesis implementation is one implementation and is not protocol authority.

## Inputs and outputs

The worker consumes one `tw.observatory-job/0.1` document. The E3 fixture
profile accepts only:

```text
mode             local_fixture
artifact_kind    robots
host             literal 127.0.0.1
scheme           http
path             /robots.txt
redirects        zero
product token    TWIRXBot
body ceiling     500 KiB
```

The URL must contain an explicit valid TCP port. Credentials, query strings,
fragments, hostnames, other loopback addresses, private addresses, and public
addresses are rejected before a client is constructed.

A successful run publishes:

```text
job.json                    exact job authority
cas/sha256/...              immutable response body
evidence/observation.cbor   canonical observation envelope
evidence/observation.json   explanatory view
evidence/body.ref           CAS reference
result.json                 robots evaluation bound to job and evidence
```

## Publication order

The required order is:

```text
validate exact job
      ↓
preserve job authority
      ↓
retrieve bounded fixture representation
      ↓
write body to CAS
      ↓
write observation envelope and body reference
      ↓
parse robots bytes
      ↓
evaluate target
      ↓
publish result
```

If parsing or evaluation fails, the observation remains and no successful
result is published. The parser cannot promote the fixture or mutate Atlas
policy, registry, scheduler, or semantic state.

## Offline verification

Verification reads the preserved job, canonical observation, CAS body, and
result. It recomputes each digest, checks every shared field, parses the stored
body again, and requires the same robots decision. The verifier creates no
network client. Verification succeeds after the fixture origin is stopped.

## Failure behavior

The worker rejects non-regular or symlink job and verification artifacts,
duplicate JSON keys, unknown fields, trailing data, unsupported modes,
oversized bodies, invalid UTF-8 robots content, redirects, inconsistent
digests, changed evidence, and changed decisions. Output for a retrieval run
must be a new or empty directory so earlier evidence is not silently replaced.

## Security considerations

Application-layer URL and resolved-address validation is defense in depth, not
host-enforced production isolation. The included service template restricts a
future fixture process to loopback networking, but it is not activated by the
repository and is not evidence of VPS enforcement. Public-origin work still
requires a separate review for DNS behavior, private-range enforcement,
redirect policy, credentials, quotas, monitoring, revocation, and disposable
workers.

The local fixture does not alter `atlas/policies.json`. In particular, it is
not evidence for the live `https://twirx.org/robots.txt` representation.

## Conformance

```bash
make build
make demo-e3-worker
go test ./internal/observatoryworker ./cmd/twirx-observer-worker
```

A conforming fixture implementation must reject every non-literal-loopback
destination, publish evidence before parsing, emit no result after a parse
failure, and replay the successful decision without network access.
