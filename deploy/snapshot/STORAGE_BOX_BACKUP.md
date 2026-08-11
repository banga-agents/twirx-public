# Encrypted Storage Box backup and restore

Status: operator plan; no backup credential or repository is configured here

## Boundary

Use a client-side encrypted Borg repository on the 20 TB Storage Box. The
backup client runs away from Meridian and receives read-only access to admitted
Object Storage artifacts. The encryption secret is held separately from the
Storage Box account, Object Storage credentials, Meridian and Git.

A restricted append-only identity performs backups. A separate recovery
identity is required for prune or destructive maintenance. Storage Box
snapshots may help recover that service, but they are not a separate full
backup.

## Archive contents

Each admitted snapshot archive contains:

- canonical snapshot manifest;
- every constituent artifact;
- detached display manifest when published;
- build and admission reports;
- the exact release-pointer version used;
- tool-version and backup-inventory metadata that contains no credential.

Mutable local development databases are not substituted for these immutable
artifacts. Future PostgreSQL base backups and WAL use separate archive classes
and recovery procedures.

## Admission drill

No public release is called durable until a restore has passed:

```text
borg check <repository>
borg extract --dry-run <repository>::<archive>
restore into a new mktemp directory
verify manifest and every artifact with networking disabled
recompute snapshot_id
open read-only runtime
run fixed query/trace smoke vectors
compare counts and digests with build report
record elapsed time, restored bytes and exact tool versions
```

The real command record MUST redact repository user/host/path when it is
sensitive and MUST never include passphrases or environment dumps. Cleanup may
remove only the validated temporary restore directory.

Genesis cadence is one archive for every admitted snapshot and an isolated
restore drill at least monthly. Any failed backup or restore blocks promotion
of a new channel pointer.
