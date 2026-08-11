# FUTO off-host durability gate

**Status:** PASS_WITH_CONDITIONS — an isolated encrypted Storage Box archive
passed full data checking and a byte-identical independent restore; the
versioned Object Storage replica remains pending.

**Audit date:** 2026-08-11

## Protected existing resources

The operator identified these existing resources:

- Object Storage bucket `meridianv2raw` at
  `hel1.your-objectstorage.com`;
- Storage Box service `quantlab-archive-bx41` at
  `u623297.your-storagebox.de`.

The existing `meridianv2raw` bucket was not read, listed, written or
reconfigured. On the Storage Box, no existing archive or directory was listed,
read, modified or reused. The gate created only the new TWIRX-owned parent
`twirx-backups` and the isolated Borg repository
`futo-semantic-snapshot-v1`.

An existing object-storage credential file was left unread and unchanged.
TWIRX did not reuse unrelated object-storage identities, passphrases or archive
paths. The already-authorized Storage Box SSH key was used only for the new
TWIRX path.

## Remaining operator input

The durability gate requires deliberately isolated resources:

1. a new private TWIRX Object Storage bucket, or an explicitly empty and
   dedicated TWIRX prefix whose policy cannot access `meridianv2raw`;
2. a TWIRX release-publisher S3 identity scoped to create/read the admitted
   snapshot prefix and unable to access unrelated buckets;
3. bucket versioning enabled and lifecycle evidence recorded;
4. optionally replace the current authorized SSH identity with a dedicated
   TWIRX Storage Box subaccount for stricter least privilege.

Credentials must be injected interactively or through a root-owned secret
file. They must not be pasted into Git, reports, command arguments preserved
in shell history, or this conversation.

## Commands executed

The Storage Box proof used Borg 1.4.4 locally and Borg 1.2.9 remotely:

```bash
ssh -p 23 -o BatchMode=yes \
  u623297@u623297.your-storagebox.de pwd

borg init --encryption=repokey-blake2 \
  ssh://u623297@u623297.your-storagebox.de:23/./twirx-backups/futo-semantic-snapshot-v1

borg create --stats \
  ssh://u623297@u623297.your-storagebox.de:23/./twirx-backups/futo-semantic-snapshot-v1::futo-snapshot-54739822-20260811 \
  var/futo-public-snapshot-d13c0bf-rebuilt

borg check --verify-data \
  ssh://u623297@u623297.your-storagebox.de:23/./twirx-backups/futo-semantic-snapshot-v1

borg extract \
  ssh://u623297@u623297.your-storagebox.de:23/./twirx-backups/futo-semantic-snapshot-v1::futo-snapshot-54739822-20260811

diff -u /tmp/twirx-original-snapshot.sha256 \
  /tmp/twirx-restored-snapshot.sha256

bin/twirx-snapshot verify \
  --snapshot "$restore_root/var/futo-public-snapshot-d13c0bf-rebuilt" \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5
```

No production PostgreSQL, mutable semantic state, broad corpus or Meridian
RAID change is part of this gate.

## Current evidence

- canonical snapshot ID:
  `sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`;
- local snapshot verification: PASS;
- immutable loopback runtime: PASS;
- Object Storage versioning/upload/download: NOT RUN;
- independent encrypted Borg backup/check/restore: PASS;
- restored files: 85;
- restored tree hash-list SHA-256:
  `eb5f394ae3b82abb6dd5959cf4891b6ee2f9191587203f8d14d7f52b5f993a7a`;
- original tree hash-list SHA-256: the same value;
- restored snapshot semantic verification: PASS;
- Borg archive fingerprint:
  `c0ed6502862470166bfe365d68d174886cfcaa242f713f596519bfca68ef965e`;
- archive original/compressed/deduplicated sizes:
  676.05 kB / 260.75 kB / 202.57 kB.

The locally held passphrase and exported encrypted repository key are mode
`0600` at:

```text
/home/shiva/.config/twirx/futo-storagebox-borg-passphrase
/home/shiva/.config/twirx/futo-storagebox-borg-key-export
```

Their contents were not printed, committed or transmitted through chat.

## Unresolved risk

The snapshot now has one demonstrated encrypted off-host backup and clean
restore path. It does not yet have the planned versioned Object Storage copy,
and the local passphrase/key export still requires an additional offline
operator-controlled copy. The existing `meridianv2raw` bucket remains outside
TWIRX scope.

## Next recommended gate

Create an isolated TWIRX Object Storage bucket and least-privilege publisher
identity, enable versioning and lifecycle policy, and run the same download and
verification drill. Preserve an offline copy of the Borg passphrase and key
export. Do not use or alter existing `meridianv2raw` objects or any pre-existing
Storage Box archive path.
