# ADR 010: Meridian as a stateless Semantic Snapshot edge

Status: accepted by founder; deployment remains subject to review

Date: 2026-08-11

## Context

ADR 009 correctly blocks authoritative PostgreSQL state on a host with a
degraded root storage array and no tested off-host point-in-time recovery. That
production-state prohibition must not be mistaken for a prohibition on local
E3.3 engineering.

The current public host is owned by Meridian. It runs unrelated services, its
root RAID1 reports only one active member, and swap is under severe pressure.
TWIRX has separate local-development capacity, Hetzner Object Storage and a
20 TB Hetzner Storage Box. The public proof can therefore use immutable state
without placing a mutable semantic database on Meridian.

## Decision

Engineering admission and production-state admission are separate gates.

Local development may implement and validate E3.3 S1 contracts, deterministic
CBOR, Go and restricted-C verification, packet compilation, deltas,
materialized-view logic, snapshot construction and local-only PostgreSQL test
profiles. This work does not authorize production state.

Meridian is a stateless public edge and replaceable read-only cache:

```text
controlled local compiler
    -> immutable evidence and semantic artifacts
    -> canonical snapshot manifest published last
    -> private/public Object Storage boundaries
    -> independently encrypted Storage Box backup
    -> verified atomic download
    -> read-only Meridian snapshot runtime
```

The edge exposes bounded `twirx.query`, `twirx.compare`, `twirx.trace`,
`twirx.explain`, `twirx.resolve`, and `twirx.snapshot.describe` operations over
one admitted snapshot. It performs no semantic-state writes and holds no
compiler, database, Object Storage write, Storage Box, signing or backup
credentials.

## Meridian prohibitions

- no authoritative PostgreSQL or WAL-dependent database;
- no local compiler, crawler frontier, browser worker or model training;
- no large corpus retention or arbitrary URL capability;
- no authenticated, payment, write or material-action execution;
- no RAID, filesystem-topology, unrelated service, data or repository changes;
- no automatic download or activation of an unadmitted snapshot;
- no TWIRX process permitted to consume swap.

## Resource envelope

The absolute founder ceiling remains 80 GiB of local TWIRX data with at least
20 GiB free for Meridian. The initial demo is tighter:

```text
one canonical snapshot:        <= 8 GiB
active + previous + staging:   <= 24 GiB
all TWIRX state/cache/logs:     <= 32 GiB operational stop
runtime CPUQuota:              200% (2 of 16 logical CPUs; 12.5% host)
runtime MemoryHigh:            1536 MiB
runtime MemoryMax:             2048 MiB
runtime MemorySwapMax:         0
runtime IOWeight:              1
runtime TasksMax:              64
runtime LimitNOFILE:           256
public query concurrency:      8
```

`CPUQuota=200%` is two logical CPUs in systemd notation, not 200% of the
machine. Cgroup limits enforce CPU, memory, swap, I/O priority, tasks and file
descriptors. Snapshot byte totals, retained release count and a pre-activation
free-space floor enforce the initial disk envelope. Meridian systemd 257 does
not support the project-quota directives added in systemd 258, and changing the
root filesystem to enable project quotas is forbidden. Deployment evidence
must state this residual difference rather than claiming a kernel disk quota.

The service receives a read-only release tree. A separate founder-invoked
updater may write only to staging, refuses a manifest above the snapshot/file
budgets, verifies every artifact offline, and switches one release symlink
atomically. It retains at most one previous snapshot and never activates on
verification or free-space failure.

## Off-host authority

Hetzner Object Storage holds private immutable evidence/build artifacts and a
separate public snapshot-release surface. Bucket versioning and lifecycle are
operator-verified. The public release credential cannot read private evidence;
Meridian receives public read-only access only.

The Storage Box holds an independently encrypted Borg repository and cold
archive. Its credentials and encryption secret never reach Meridian or the
repository. Storage Box snapshots may aid recovery of that box but are not an
independent backup.

## Promotion to mutable production state

A future funded host may run authoritative PostgreSQL only after separate
storage, recovery, monitoring and founder admission. Snapshot identity and
canonical packet/delta artifacts remain valid during that migration; the
database imports them and reconstructs current state rather than redefining
their protocol meaning.

## Consequences

- The public proof can proceed without risking Meridian's mutable state.
- The snapshot runtime becomes the first edge/materialization implementation,
  not throwaway demo code.
- The degraded RAID remains a hard database blocker but no longer freezes S1.
- Public availability depends on immutable release freshness; stale state is
  explicit and never disguised as current publisher data.
- Object Storage and Storage Box credentials become operational prerequisites,
  not repository dependencies.

## Rejected alternatives

- Installing PostgreSQL on Meridian for speed: rejected because it converts a
  known single-device failure path into authoritative project state.
- Serving a local compiler on Meridian: rejected because it introduces writes,
  egress and resource contention on a shared host.
- Treating Storage Box snapshots as backup independence: rejected because they
  share the same Storage Box failure domain.
- Pausing all engineering until new hosting is funded: rejected because local
  S1 and immutable snapshot development do not depend on production storage.
