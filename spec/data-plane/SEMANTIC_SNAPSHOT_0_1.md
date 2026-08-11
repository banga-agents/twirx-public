# Semantic Snapshot 0.1

**Authority:** Normative Genesis contract

**Status:** Offline manifest codec and constituent verifier implemented as an
S1 review candidate; a bounded local builder and read-only runtime are an
implementation candidate under `READ_ONLY_SNAPSHOT_RUNTIME_0_1.md`; activation
remains pending a separate gate

## Purpose

A Semantic Snapshot is an immutable, bounded and independently verifiable
release of TWIRX semantic state. It permits a read-only edge to answer admitted
queries without a mutable database, compiler, browser, model or origin network
access.

The protocol is language-neutral. Go and restricted C are independent Genesis
implementations, not normative authority.

## Identity and publication topology

All constituent artifacts are complete before the manifest exists:

```text
artifact bytes
    -> SHA-256 artifact digests and exact byte sizes
    -> deterministic-CBOR semantic-snapshot-manifest
    -> snapshot_id = SHA-256(exact manifest.cbor bytes)
```

The manifest is self-digest-free. It MUST NOT list its own bytes, a detached
JSON rendering or a mutable channel pointer as an artifact. A JSON rendering
MAY display the detached `snapshot_id`, but it is explanatory and cannot be
used as the canonical manifest.

The canonical manifest is published last. A snapshot does not exist unless
the manifest and every artifact it references are present and valid.

## Bounds

| Item | Limit |
| --- | ---: |
| Canonical manifest | 4 MiB |
| Artifacts per snapshot | 16,384 |
| Individual artifact | 4 GiB |
| Total constituent bytes | 8 GiB |
| Materialized views | 32 |
| Evidence classes | 16 |
| Path length | 255 UTF-8 bytes |

Implementations MAY use lower operational limits but MUST NOT accept above
these protocol bounds.

## Artifact roles

The closed Genesis roles are:

```text
origin_catalog
concepts
mappings
packet_batch
delta_batch
materialized_view
search_index
proof_index
economic_summary
build_report
```

Every snapshot MUST contain at least one `origin_catalog`, `concepts`,
`mappings`, `packet_batch`, `proof_index` and `build_report` artifact. Empty
semantic collections use canonical empty collection artifacts rather than
omitting required roles.

## Ordering and uniqueness

- artifact entries are strictly increasing by UTF-8 path bytes;
- artifact paths are unique;
- view entries are strictly increasing by view identifier bytes;
- evidence classes are strictly increasing and unique;
- no two artifact paths may reference different bytes with the same role and
  declared logical identity;
- extension arrays are empty in 0.1.

## Safe relative paths

An artifact path:

- is relative and uses `/` as its only separator;
- contains only non-empty UTF-8 segments;
- contains no `.` or `..` segment;
- contains no NUL, backslash, control character or percent-encoded separator;
- does not begin or end with `/`;
- is not `manifest.cbor`, `manifest.json` or a path beneath `channels/`.

Importers MUST resolve the destination beneath an already-created staging
directory and reject any path that escapes it. Symlinks, hard links, devices,
FIFOs and sockets are forbidden snapshot artifacts.

## Required manifest reconciliation

The verifier recomputes and compares:

- `total-artifact-bytes` as the exact sum of artifact sizes without overflow;
- count fields against the declared artifact contents or their independently
  verified indexes;
- every artifact digest from the exact stored bytes;
- every required role;
- each view's definition, artifact digest, row count and through-sequence;
- the detached snapshot identifier from exact manifest bytes.

Missing required evidence fails closed. Optional content remains explicit in
its source artifact and never becomes an inferred value during import.

## Build authority

The manifest records:

- exact compiler contract digest and implementation version;
- source revision without treating a Git implementation as normative;
- Atlas selection and canon module-set digests;
- prior snapshot identity when one exists;
- archive/live/fixture evidence classes used;
- build report digest;
- highest packet and delta sequence included;
- generated counts and total bytes.

A prior snapshot reference establishes lineage only. It does not make an
otherwise invalid snapshot acceptable.

## Import and activation

Import is offline and fail closed:

1. read at most 4 MiB of `manifest.cbor`;
2. verify canonical encoding and all semantic constraints;
3. compute and compare the expected detached snapshot ID;
4. preflight artifact count, per-file, total-byte and free-space limits;
5. retrieve into a new staging directory without following links;
6. verify every byte size and digest before parsing higher-level content;
7. verify indexes, views and reconciled counts;
8. open the staged snapshot in a read-only runtime and run fixed smoke queries;
9. rename staging to the content-addressed release directory;
10. atomically switch the active release reference.

Any error leaves the existing release active. Automatic activation from an
unreviewed remote `latest` pointer is forbidden.

## Runtime boundary

The edge runtime receives the release tree read only. It MAY expose bounded
query, compare, trace, explain, resolve and snapshot-description operations.
It MUST NOT:

- modify semantic state or the active snapshot;
- contact source origins;
- accept arbitrary URLs, SQL, filesystem paths or remote tool servers;
- execute browsers, models, payments, authentication, writes or material
  actions;
- possess compiler, publication, Object Storage write, backup or signing
  credentials.

## Migration

A mutable store imports exact artifact identities and rebuilds derived current
state. It may add operational indexes and sequence assignments, but MUST NOT
alter packet, delta, observation, mapping, batch or snapshot identity. A full
rebuild MUST reproduce the admitted materialized-view outputs before cutover.
