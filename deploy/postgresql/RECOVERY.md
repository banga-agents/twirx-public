# PostgreSQL Genesis recovery plan

Status: required design; no backup target configured and no drill completed

## Recovery objectives

Initial objectives, subject to founder and operator approval:

```text
RPO: 15 minutes for admitted operational state
RTO: 4 hours for a single-node Genesis restore
```

These are targets, not demonstrated guarantees. Public status must not claim
them until an isolated restore drill meets both.

Canonical source representations, packet bytes, batch manifests and proof
bundles in the CAS have their own off-host backup and integrity inventory.
Database recovery without the corresponding canonical artifacts is
incomplete.

## Backup layers

### Continuous WAL archive

- Archive completed WAL segments to encrypted off-host object storage.
- Use credentials that can write only the configured backup namespace and
  cannot read application secrets or alter source state.
- Set an archive timeout consistent with the RPO after measuring WAL volume.
- Record archive success, lag, last durable segment and failure alarms.
- Object retention and deletion require an explicit approved policy.

### Physical base backup

- Take a regular `pg_basebackup` from the restricted backup role.
- Stream or immediately transfer the backup off-host; a copy that exists only
  on the VPS or its second local device is not a disaster-recovery backup.
- Encrypt in transit and at rest.
- Record PostgreSQL version, timeline, start/end LSN, manifest and SHA-256
  inventory.
- Verify the backup manifest after transfer.

### Logical and configuration backup

- Export schema-only SQL and globals/role definitions as reviewable recovery
  aids.
- Export bounded logical snapshots for portability and forensic comparison.
- Back up operator-approved PostgreSQL and systemd configuration separately.
- Never place cleartext role passwords, private keys or connection strings in
  the repository or report.

Logical dumps supplement physical/PITR recovery; they do not replace it.

## Isolated restore drill

The drill runs on a non-public, separately addressed target with no production
egress worker credentials:

1. Provision the exact supported PostgreSQL major version and approved
   extensions.
2. Restore the most recent verified base backup.
3. Replay archived WAL to a chosen timestamp and record the recovered LSN.
4. Start with all public endpoints and subscription delivery disabled.
5. Verify schema migrations, role ownership and forbidden privileges.
6. Rehash sampled and boundary canonical artifacts against the CAS inventory.
7. Check packet/delta identity uniqueness, batch completeness, head references,
   cursor monotonicity and outbox consistency.
8. Drop and rebuild all materialized semantic views from immutable history.
9. Execute the deterministic E2 reconciliation and E3.3 query corpus.
10. Compare rebuilt result/materialization digests with the pre-backup evidence.
11. Measure achieved data loss and elapsed recovery time.
12. Destroy or retain the isolated restore only under the approved evidence and
    credential-erasure policy.

The drill fails if a required CAS object is missing, an immutable digest
differs, materialized state cannot be rebuilt, public delivery starts during
recovery, or the measured RPO/RTO exceeds the stated objective without an
explicitly published limitation.

## Failure scenarios

| Scenario | Recovery path |
| --- | --- |
| PostgreSQL process/transaction failure | restart after log review; rely on WAL atomicity; verify outbox/head consistency |
| Corrupt materialized view | stop affected read route, rebuild from immutable packet/delta log |
| Database volume loss | restore off-host base backup and replay WAL |
| CAS loss with database intact | fail closed for affected proof reads; restore CAS off-host inventory before service |
| Accidental semantic mapping change | publish a semantic/canon delta or revert current projection; never rewrite packets |
| Origin statement retraction | append lifecycle/retraction state with provenance; do not erase historical evidence outside retention policy |
| Credential compromise | revoke/rotate role and backup credentials, stop delivery, audit outbox and database logs, restore only if integrity is uncertain |
| Complete VPS loss | provision a clean host, restore CAS, PostgreSQL, configs and Caddy in that order; re-enable delivery last |

## Drill cadence

- Complete one successful full drill before any production semantic admission.
- Repeat after PostgreSQL major changes, storage/topology changes or recovery
  procedure changes.
- Run at least quarterly while the Genesis service is public.
- Test a smaller materialization rebuild and backup-manifest verification at
  least monthly.

Each drill produces a commit-bound report with exact commands, backup and
target identifiers, observed RPO/RTO, integrity checks, exclusions and
unresolved limitations. Sensitive endpoints and credentials are redacted.

## References

- Continuous archiving and PITR:
  <https://www.postgresql.org/docs/18/continuous-archiving.html>
- `pg_basebackup`: <https://www.postgresql.org/docs/18/app-pgbasebackup.html>
- `pg_verifybackup`: <https://www.postgresql.org/docs/18/app-pgverifybackup.html>
- `pg_dump`: <https://www.postgresql.org/docs/18/app-pgdump.html>
