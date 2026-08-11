# Local PostgreSQL development profile

Status: design for E3.3 S2 tests; not production admission

This profile is for a developer-controlled machine after S1 passes. It does
not authorize installation on Meridian and does not make local state durable
or public.

## Boundary

- PostgreSQL 18 only, pinned to an operator-reviewed patch release.
- Unix socket and loopback listeners only.
- No public firewall rule, Caddy route or LAN listener.
- Separate `twirx_dev` cluster/database/roles with disposable test data.
- No production, Object Storage, Storage Box or Meridian credentials.
- Required extensions: `pg_trgm` and `unaccent`; `pgvector` remains absent.
- Test fixtures and snapshot artifacts are public/controlled data only.

## Starting limits

```text
max_connections = 20
shared_buffers = 256MB
work_mem = 4MB
maintenance_work_mem = 128MB
temp_file_limit = 512MB
idle_in_transaction_session_timeout = 30s
lock_timeout = 5s
statement_timeout = 30s for request roles
```

The developer may use a native package or disposable rootless container, but
the exact image/package digest and commands must enter S2 evidence. A container
configuration is not a runtime dependency and must not silently download during
normal offline tests.

## Required S2 proof

1. Fresh migration from zero.
2. Idempotent schema-version check and explicit rollback boundary.
3. Runtime privilege-denial probes.
4. Transaction-kill atomicity across packet, head, delta and outbox changes.
5. Partition-boundary and global-digest uniqueness tests.
6. Snapshot import and full materialization rebuild.
7. Logical export plus isolated restore for development evidence.
8. Clean removal without touching another local database.

Local success is engineering evidence only. Production admission still requires
the independent durable-host and off-host recovery gates in ADR 009.
