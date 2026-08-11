# FUTO off-host durability gate

**Status:** BLOCKED — storage boundaries are designed, but no TWIRX-specific
credential or independent restore target has been authorized.

**Audit date:** 2026-08-11

## Protected existing resources

The operator identified these existing resources:

- Object Storage bucket `meridianv2raw` at
  `hel1.your-objectstorage.com`;
- Storage Box service `quantlab-archive-bx41` at
  `u623297.your-storagebox.de`.

Neither resource was read, listed, written, reconfigured or used by this gate.
Their names and existing usage indicate Meridian/Quantlab ownership, and the
founder expressly required TWIRX not to touch unrelated Meridian data or
repositories.

An existing host credential file was also left unread and unchanged. TWIRX
will not reuse unrelated object-storage identities, backup keys, passphrases
or archive paths.

## Required operator inputs

The durability gate requires deliberately isolated resources:

1. a new private TWIRX Object Storage bucket, or an explicitly empty and
   dedicated TWIRX prefix whose policy cannot access `meridianv2raw`;
2. a TWIRX release-publisher S3 identity scoped to create/read the admitted
   snapshot prefix and unable to access unrelated buckets;
3. bucket versioning enabled and lifecycle evidence recorded;
4. a new encrypted Borg repository path on the Storage Box, separate from all
   current archives;
5. the exact Storage Box SSH port and a TWIRX-specific SSH key or subaccount;
6. a new Borg passphrase held outside the repository, logs and chat.

Credentials must be injected interactively or through a root-owned secret
file. They must not be pasted into Git, reports, command arguments preserved
in shell history, or this conversation.

## Planned admission commands

After isolated credentials exist, the operator procedure in
`deploy/snapshot/OBJECT_STORAGE.md` and
`deploy/snapshot/STORAGE_BOX_BACKUP.md` will:

```text
verify versioning and least-privilege boundaries
publish the exact admitted immutable snapshot
download it into a new temporary directory
verify every digest and recompute snapshot_id
create a client-side encrypted Borg archive
restore the archive into a second new temporary directory
verify the snapshot with networking disabled
open the read-only runtime and run fixed query/trace smoke vectors
record restored bytes, elapsed time and tool versions
```

No production PostgreSQL, mutable semantic state, broad corpus or Meridian
RAID change is part of this gate.

## Current evidence

- canonical snapshot ID:
  `sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`;
- local snapshot verification: PASS;
- immutable loopback runtime: PASS;
- Object Storage versioning/upload/download: NOT RUN;
- independent Borg backup/check/restore: NOT RUN;
- byte-identical off-host restore: NOT RUN.

## Unresolved risk

The snapshot currently lacks demonstrated durability across two independent
off-host storage paths. This remains a FUTO send-gate blocker, but protecting
unrelated Meridian/Quantlab data takes precedence over reusing the available
accounts without explicit isolation.

## Next recommended gate

Create the isolated TWIRX bucket and backup repository, inject scoped
credentials outside Git, and execute one recorded byte-identical restore
drill. Do not use or alter the existing `meridianv2raw` objects or current
`quantlab-archive-bx41` archive paths.
