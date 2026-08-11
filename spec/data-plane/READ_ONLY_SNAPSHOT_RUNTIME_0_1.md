# Read-only Semantic Snapshot runtime 0.1

**Authority:** Normative operational profile for the E3.3 grant demonstration

**Status:** Local implementation candidate; no public deployment admission

## Purpose

This profile turns the immutable Semantic Snapshot contract into an executable,
bounded demonstration without installing mutable production state. It compiles
already-admitted evidence offline, publishes a manifest last, and answers typed
queries from verified materialized state without origin access, a browser, a
model or a database.

The rules are language-neutral. `twirx-snapshot`, Go and the JSON carriage
containers are Genesis implementation choices, not the source of protocol
authority. The canonical packets, queries, query results and snapshot manifest
remain the deterministic-CBOR objects defined by this specification set.

## Claim boundary

The first snapshot contains two different populations:

1. the 500 candidate identities in the admitted Genesis selection; and
2. semantic packets compiled only from E2 operations with committed replay
   evidence.

An identity without admitted evidence produces no packet. A controlled fixture
is marked `test_fixture` and is excluded from public-origin counters, public
views, default queries and traces. The `origins` snapshot count means Atlas
identities carried by the snapshot; it does not mean 500 cataloged, policy
reviewed, observed, compiled or live origins.

E2 offline replay is labeled `recorded_offline_replay`, has stale freshness,
and does not become a current publisher statement. The build report records
targets and actual counts in separate structures. Target values MUST NOT be
rendered as achieved values.

## Deterministic build

The builder receives four explicit inputs:

```text
repository root
new output directory
exact source revision
canonical UTC creation time
```

It performs no network requests. It loads the canonical Atlas selection,
contracts and origin catalog; invokes each admitted E2 operation in replay
mode; verifies each manifest-last E2 proof bundle; and preserves the exact
native term, locator and lexical value before constructing a semantic packet.

Packet derivation binds the observation, transport, adapter, field extraction
plan, transformation sequence, reviewed mapping, semantic closure and compiler
contract. Integer-shaped lexical values declared by the broader E2 `decimal`
type are not silently rewritten to meet the narrower Semantic Packet decimal
profile; their typed packet value remains absent while the native lexical value
and semantic predicate remain available.

The output directory is reserved with an exclusive create before any artifact
is written. It is an incomplete staging area until the canonical manifest is
written last. The complete directory is then verified against its detached
identity. Existing output is never overwritten, including under a concurrent
creator race. A failed build may leave its exclusively reserved incomplete
directory, which has no manifest and therefore is not a snapshot.

## Snapshot carriage artifacts

The Genesis runtime uses bounded deterministic JSON carriage artifacts around
canonical protocol objects:

| Format | Purpose |
| --- | --- |
| `tw.snapshot-packet-segment/0.1` | globally sequenced packet digest and exact CBOR bytes |
| `tw.snapshot-concepts/0.1` | sorted concept and module identifiers used by the packet set |
| `tw.snapshot-mappings/0.1` | sorted reviewed native-to-semantic mapping records |
| `tw.snapshot-proof-index/0.1` | packet-to-E2-bundle evidence references |
| `tw.snapshot-view/0.1` | proof-linked materialized display rows |
| `tw.snapshot-build-report/0.1` | actual counts, separate targets and explicit limitations |

The packet segment is limited to 32,768 entries. Each entry contains a global
sequence, detached digest and exact canonical packet bytes. An importer MUST
verify sequence continuity, digest equality and the canonical packet before
admission. Multiple segments continue the global sequence; they do not restart
at one.

The independent restricted-C verifier validates canonical packet, query,
query-result and manifest bytes. The JSON carriage layer is operational index
data and does not replace independent constituent verification.

## Import reconciliation

Opening a snapshot performs the complete byte and path checks in
`SEMANTIC_SNAPSHOT_0_1.md`, followed by semantic reconciliation:

- exactly one selection, concepts, mappings and build-report artifact;
- packet counts and global sequence equal the manifest;
- packet identities are unique;
- every packet has one proof-index entry;
- every proof reference names a manifest-bound artifact with the same digest
  and size;
- public views cannot contain controlled-fixture packets;
- view rows, row counts and through-sequences match the manifest;
- build-report actual counts match the manifest;
- the build report declares zero retrieval requests and no current claims.

Failure at any stage rejects the entire snapshot.

## Query profile

The runtime accepts the canonical Semantic Query 0.1 object but implements
only the materialized exact-match subset:

- exact native or semantic predicate selection;
- exact native or canonical subject selection;
- admitted origin, authority, lane and mapping-status filters;
- current, as-of, between and history observation-time filters;
- explicit stale exclusion or explicit stale return;
- deterministic result and proof-byte limits;
- packet-level proof, source-preserving results and detached query, plan and
  result identities.

Live refresh, ontology expansion, dimension inequalities, economic filtering,
field/bundle proof expansion, conflict resolution, preference ranking, browser
execution and action capabilities fail with an explicit unsupported error. A
query cannot authorize new execution behavior.

The default public profile excludes controlled fixtures. Fixture visibility is
an explicit local conformance option and MUST NOT be enabled on the public
service.

## Service boundary

The Genesis command exposes the non-normative HTTP mapping:

```text
GET  /api/v1/status
GET  /api/v1/origins
GET  /api/v1/origins/{origin-id}
GET  /api/v1/concepts
GET  /api/v1/views
GET  /api/v1/snapshot/manifest.cbor
GET  /api/v1/packets/{packet-digest}
GET  /api/v1/trace/{packet-digest}
GET  /api/v1/proof/{packet-digest}/{artifact-name}
POST /api/v1/query
```

The process binds only to a literal IPv4 or IPv6 loopback address. It rejects
wildcard, public and hostname listen values. A separately reviewed edge proxy
may expose the bounded service after deployment admission. Request bodies,
headers, timeouts and concurrency are bounded; the runtime has no outbound
network path and no state-write endpoint.

The origin endpoints expose only the exact immutable Atlas selection carried
by the snapshot: canonical ID, HTTPS origin, host, domain family, selection
catalog state and the count of public packets in this snapshot. They do not
profile, retrieve or invoke an origin, and they do not infer policy review,
technical maturity, publisher status or health from selection or packet count.

Packet, manifest and proof-artifact downloads are selected only through an
already-admitted packet digest and exact proof-index artifact name. Visitor
input never becomes a filesystem path. The returned bytes are rehashed against
the in-memory admitted manifest immediately before serving.

## Admission boundary

This profile demonstrates deterministic packet compilation, materialization,
query and trace over a small admitted corpus. It does not pass E3.3 S3, S4, S5
or S6 as full subgates. In particular, it has no prior snapshot from which to
derive a genuine change event, no ontology expansion, no subscriptions, no
archive-assisted importer and no production activation evidence.
