# Object Storage release boundary

Status: operator plan; no bucket or credential has been created by this work

The canonical artifacts remain valid independently of any storage vendor.
Hetzner Object Storage is the planned Genesis off-host carrier, not protocol
authority.

## Layout

Use separate private build/evidence and public release policies:

```text
private
  cas/sha256/aa/bb/<digest>
  evidence/<origin-id>/<observation-digest>
  builds/<build-id>/<relative-artifact-path>
  quarantine/<build-id>/...

public
  snapshots/sha256/<snapshot-id>/manifest.cbor
  snapshots/sha256/<snapshot-id>/manifest.json
  snapshots/sha256/<snapshot-id>/artifacts/<relative-artifact-path>
  channels/genesis/current.cbor
```

The immutable release prefix MUST be create-only for the publisher. A later
object with the same key must have identical bytes and digest or publication
fails. The mutable channel pointer is versioned and only references an already
admitted snapshot; it does not define snapshot identity.

## Credentials

| Role | Private build | Public release | Delete/lifecycle | Meridian |
| --- | --- | --- | --- | --- |
| compiler | bounded read/write | none | none | absent |
| release publisher | one admitted build, read | create immutable release and channel pointer | none | absent |
| backup client | admitted objects, read | read | none | absent |
| lifecycle administrator | metadata only | metadata only | lifecycle/versioning administration | absent |
| edge updater/runtime | none | exact-object read only | none | present only if public anonymous read is not used |

No role may reuse another role's secret. Credentials are injected by the
operator and never stored in repository configuration, snapshot artifacts,
logs, command history or public reports.

## Versioning and lifecycle admission

Before first release, record independent evidence that:

1. versioning is enabled on each participating bucket;
2. public objects cannot read private prefixes;
3. the edge identity cannot list private objects, write or delete;
4. aborted multipart uploads and quarantined temporary objects have a bounded
   cleanup rule;
5. admitted evidence and immutable snapshot releases have no automatic expiry
   until a founder-approved retention policy exists;
6. channel-pointer history remains recoverable;
7. a byte-for-byte download reproduces the published snapshot ID.

Bucket versioning helps recover object revisions but is not the independent
backup. The encrypted Storage Box repository supplies the second failure
domain.
