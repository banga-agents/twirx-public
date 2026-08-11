# Immutable snapshot edge candidate

Status: repository configuration candidate only. Nothing in this directory has
been installed, enabled, started, published to Object Storage or activated on
Meridian.

The service opens one founder-admitted Semantic Snapshot and serves the bounded
read-only API on literal loopback port 8091. Caddy may reverse proxy a reviewed
public route after separate deployment admission. The process cannot fetch an
origin, refresh semantic state or write a release.

## Files

- `twirx-snapshot.service` applies the ADR 010 CPU, memory, swap, I/O, task,
  descriptor, filesystem, syscall and loopback-network boundary.
- `snapshot-runtime.env.example` documents the two non-secret release values;
  an operator creates the real root-owned file outside Git.
- `verify-runtime-unit.sh` performs read-only static checks and confirms that
  no candidate service is enabled or running.
- `OBJECT_STORAGE.md` and `STORAGE_BOX_BACKUP.md` define later durability
  admission.
- `COMMON_CRAWL_IMPORT.md` defines a later offline build boundary and is not
  part of the edge service.

## Required pre-activation evidence

Founder review must identify the exact implementation commit, snapshot ID and
snapshot build report. Before activation an operator must demonstrate:

1. byte-identical deterministic rebuild from the recorded inputs;
2. Go verification of every snapshot artifact;
3. restricted-C verification of the manifest and every canonical packet;
4. fixed query and trace smoke results;
5. an unprivileged dedicated account and root-owned immutable release tree;
6. no TWIRX write path, Object Storage write credential or backup credential;
7. `systemd-analyze verify` and security output for the installed unit;
8. actual cgroup CPU, memory, swap, I/O, task and descriptor enforcement;
9. loopback-only listener and Caddy route limits;
10. disk ceilings, free-space reserve, rollback and off-host restore proof;
11. unchanged Meridian services and no RAID, repository or unrelated data
    mutation.

The environment file pins both the content-addressed path and detached ID. A
mutable `latest` value is never runtime authority. The service fails before
listening if either the snapshot or expected identity is invalid.
