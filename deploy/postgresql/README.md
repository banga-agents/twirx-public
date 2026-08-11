# PostgreSQL Genesis deployment candidate

Status: design only; not installed or activated

This directory describes the target-host boundary for the E3.3 Genesis state
store. It is not a deployment script. Nothing here authorizes installation,
cluster initialization, RAID changes, firewall changes, database creation or
public exposure.

The current target fails its storage gate. See
`reports/e3-3-vps-capacity-baseline.md`.

## Preconditions

All of these must pass before an operator installs PostgreSQL:

- E3.2 is admitted and merged.
- E3.3 S1 contracts and adversarial conformance are admitted.
- Root storage redundancy or an approved replacement durability design is
  healthy and evidenced.
- Swap pressure and shared-service resource ownership are understood.
- An encrypted off-host base-backup and WAL archive destination exists.
- The founder authorizes target-host activation.

PostgreSQL 18 packages are not currently available from the host's configured
APT sources. Adding the PostgreSQL Apt repository is a separate operator
change: verify its Debian 13 codename, signing key, package pinning and exact
package versions before approval. Do not pipe a remote script into a shell.

## Intended topology

```text
Caddy :443
    -> TWIRX API on loopback/Unix socket
        -> least-privilege PostgreSQL application role

compiler/admission service
    -> CAS verification
    -> PostgreSQL compiler role

E3.2 egress worker
    - no PostgreSQL credentials
    - no database socket visibility
    - no registry or deployment writes
```

PostgreSQL listens only on a Unix socket and, if operationally required,
`127.0.0.1` and `::1`. Port 5432 remains absent from UFW's public rules. Caddy
never proxies PostgreSQL.

## Role model

| Role | Ownership and grants |
| --- | --- |
| `twirx_migrator` | owns schemas and migrations; unavailable to runtime services |
| `twirx_compiler` | verifies and inserts identities, packets, deltas, heads and outbox rows through bounded procedures |
| `twirx_query` | read-only semantic/query views; creates bounded query-run state through specific procedures |
| `twirx_subscriber` | reads admitted outbox/delta cursors; no packet or canon writes |
| `twirx_analytics_export` | read-only, rate/statement bounded export views |
| backup role | replication/backup grants only; credentials restricted to backup unit |

The database owner is never an application login. Runtime roles receive no
`CREATE`, schema ownership, superuser, replication, filesystem, program
execution or extension-install privilege. The public API cannot choose a SQL
role or submit SQL.

The packet and delta base tables are append-only for runtime roles. Current
head changes occur only in the same transaction as their delta and outbox
records. A separate migration role is required for retention detach/drop
operations after policy approval.

## Initial bounded profile

The exact configuration is generated only after the host gate. Starting
limits for measurement are:

```text
max_connections = 40
shared_buffers = 1GB
work_mem = 8MB
maintenance_work_mem = 256MB
temp_file_limit = 1GB
idle_in_transaction_session_timeout = 30s
lock_timeout = 5s
```

Statement timeouts are assigned by role and endpoint. Bulk compiler work uses
a separate bounded role; an HTTP request cannot disable its timeout.
Autovacuum remains enabled and is observed per partition. The service receives
an initial 4 GiB systemd memory ceiling and explicit file/task limits, then is
tuned from stress evidence.

## Extensions

Genesis-required extensions:

```text
pg_trgm
unaccent
```

They are installed by the migration owner from operator-approved packages.
`pgvector` is optional and disabled. Enabling it requires a later ADR update,
pinned package/source provenance, offline conformance, a capacity comparison,
and confirmation that deterministic exact/lexical/graph retrieval still works
without it.

## Admission transaction

One compiler transaction:

1. verifies the canonical artifact already exists in the CAS;
2. inserts or confirms the global packet identity;
3. inserts the immutable partitioned packet row;
4. compares the existing current head under a deterministic lock order;
5. inserts the appropriate origin, semantic or canon delta;
6. updates a materialized head if and only if the ordered state changes;
7. inserts an economic event where measurements exist;
8. inserts the public/internal outbox record;
9. commits all changes together.

A digest collision with unequal bytes, missing required evidence, unexpected
duplicate, invalid trust transition or unsupported canon version aborts the
entire transaction.

## Verification before activation

- `ss` shows no non-loopback PostgreSQL listener.
- UFW has no public PostgreSQL rule.
- the egress worker cannot see the database socket, credentials or service;
- each runtime role fails prohibited SQL privilege probes;
- packet and delta update/delete attempts fail;
- a killed compiler transaction leaves no partial head or outbox change;
- timeouts, temp-file ceilings, connection caps and service memory caps fire;
- base backup, WAL archive and isolated point-in-time restore pass;
- materialized-state rebuild produces the same deterministic digests;
- target-host verification does not inspect or modify unrelated repositories.

## References

- PostgreSQL 18 documentation: <https://www.postgresql.org/docs/18/>
- Declarative partitioning: <https://www.postgresql.org/docs/18/ddl-partitioning.html>
- Client authentication: <https://www.postgresql.org/docs/18/client-authentication.html>
- Resource consumption: <https://www.postgresql.org/docs/18/runtime-config-resource.html>
- Server signaling and `LISTEN`: <https://www.postgresql.org/docs/18/sql-listen.html>
