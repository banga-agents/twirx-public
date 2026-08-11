# E3.3 S1 Semantic Snapshot pre-implementation report

Status: **PASS for local S1 engineering; FAIL for Meridian production state or deployment**

Observed: 2026-08-11

Admitted public base before this work:
`669b2052e7586f2bc05b26ca37a22003a792dba9`

No VPS, DNS, firewall, RAID, service, package, Object Storage or Storage Box
state was changed during this assessment.

## Founder decision applied

ADR 010 separates two admissions:

- **Engineering admission:** deterministic packet, batch, delta, query,
  result, materialization and snapshot work may proceed locally.
- **Production-state admission:** authoritative mutable PostgreSQL remains
  forbidden on Meridian until independent durable infrastructure and tested
  recovery exist.

Meridian may later serve one founder-admitted immutable snapshot through a
read-only runtime. This report does not authorize that deployment.

## Exact target-host resources

| Resource | Read-only observation |
| --- | --- |
| CPU | Intel Xeon W-2145; 8 physical cores / 16 logical CPUs |
| Memory | 134,733,492,224 bytes total; 73,239,023,616 bytes available |
| Swap | 4,289,720,320 bytes total; 4,144,222,208 bytes used; 145,498,112 bytes free |
| Root | `/dev/md2`, ext4; 938,524,024,832 bytes total; 384,965,894,144 bytes available; 57% used |
| Secondary data filesystem | `/dev/nvme1n1p3`; 938,656,133,120 bytes total; 512,307,449,856 bytes available; 43% used |
| Root RAID | degraded RAID1, one active member, state `[U_]` |
| Existing TWIRX files | 2,477,405 bytes under `/srv/twirx` |
| Init | systemd 257 |
| Existing public edge | Caddy active; 187,899,904 bytes current and 293,773,312 bytes peak memory at observation |
| Database | system PostgreSQL inactive; no production database admission |
| Public firewall | SSH, HTTP and HTTPS only through existing UFW policy |

Only aggregate service and resource facts were retained. Unrelated listeners,
repositories and application metadata were not inspected or changed.

## Proposed Meridian limits

These are initial ceilings, not demonstrated capacity claims:

```text
canonical snapshot                         <= 8 GiB
active + previous + staging snapshots      <= 24 GiB
all TWIRX state, cache and bounded logs     stop at 32 GiB
Meridian free-space floor                  >= 20 GiB
CPUQuota                                   200% (two logical CPUs)
MemoryHigh                                 1536 MiB
MemoryMax                                  2048 MiB
MemorySwapMax                              0
IOWeight                                   1
TasksMax                                   64
LimitNOFILE                                256
public query concurrency                   8
```

Systemd 257 cannot enforce the project-directory quota directives introduced
in systemd 258, and changing Meridian's filesystem or RAID is forbidden. The
32 GiB stop is therefore enforced structurally by a maximum 8 GiB manifest,
one active release, one previous release, one staging release, bounded logs,
and a free-space preflight. A future deployment must prove these checks and
must not claim a kernel disk quota.

## Object Storage layout and credential boundaries

Two security surfaces are required. They may be separate buckets or policies
whose non-overlap is proven:

```text
private
  cas/sha256/<prefix>/<digest>
  builds/<build-id>/...
  evidence/<origin-id>/...
  quarantine/<build-id>/...

public release
  snapshots/sha256/<snapshot-id>/manifest.cbor
  snapshots/sha256/<snapshot-id>/artifacts/...
  snapshots/sha256/<snapshot-id>/manifest.json
  channels/genesis/current.cbor
```

Credential roles are separate:

- local compiler: write private build/CAS objects; no public publish;
- release publisher: read one admitted build and create one public immutable
  release; no private-origin discovery beyond that build;
- Meridian runtime/updater: public release read only; no listing requirement,
  private read, write, delete, lifecycle, backup or signing privilege;
- lifecycle administrator: bucket policy/versioning only; no compiler use;
- backup client: read admitted private/public objects into the encrypted cold
  repository; no Meridian access.

No credential, endpoint secret or encryption key belongs in Git. Object
Storage versioning must be enabled before first release. Genesis lifecycle may
remove incomplete multipart uploads and explicitly quarantined temporary
objects, but must not expire admitted evidence or immutable snapshot releases
without a later retention decision.

## Independent Storage Box backup and restore

The 20 TB Storage Box holds a Borg repository encrypted client-side using a
secret absent from Meridian, Object Storage and this repository. A restricted
append-only backup identity writes new archives; a separately held recovery
identity performs prune or destructive maintenance. Storage Box snapshots are
operational conveniences, not independent full backups.

Before any public snapshot admission, the following restore proof must pass in
an isolated temporary directory:

1. list and verify the selected Borg archive;
2. extract its manifest and every referenced artifact;
3. run the independent snapshot verifier with network disabled;
4. reproduce `snapshot_id` from the exact canonical manifest bytes;
5. open the snapshot in the read-only runtime and execute the fixed smoke
   queries;
6. compare packet, delta, view and artifact counts with the build report;
7. record elapsed time, restored bytes and tool versions;
8. destroy only the explicitly created temporary restore directory.

The first successful drill is an admission prerequisite. Each admitted
snapshot triggers a new archive; a monthly isolated restore is the Genesis
minimum after launch.

## Snapshot contract summary

The snapshot is acyclic and manifest-last:

```text
canonical constituent artifacts
    -> digest and size table
    -> canonical manifest.cbor
    -> snapshot_id = SHA-256(manifest.cbor)
    -> optional detached manifest.json display wrapper
    -> founder-approved mutable channel pointer
```

The canonical manifest never contains its own digest. It also does not list
`manifest.cbor`, the detached JSON wrapper or the mutable channel pointer as a
constituent. Import verifies the manifest, path safety, ordering, every size
and digest, count reconciliation, snapshot budget and required roles entirely
offline before the staging directory can be renamed and activated.

## Common Crawl archive-assisted data budget

The importer is a controlled build tool, not a public URL service. It accepts
only admitted origin/representative-route work orders and the official Common
Crawl index and data hosts. At build time it selects and records an explicitly
allowed crawl index; it does not hard-code a claim that June is still latest.
On 2026-08-11 the Common Crawl collection list identified
`CC-MAIN-2026-30` (July 2026) as the latest collection.

Per origin hard bounds:

```text
crawl periods                         <= 2
representative captures per period    <= 3
index requests                         <= 4
index response                         <= 256 KiB each
compressed WARC range                  <= 2 MiB each
decompressed WARC record               <= 8 MiB
retained representation body           <= 5 MiB
concurrency                             <= 2 globally
```

Full 500-origin theoretical network ceiling:

```text
index requests            <= 2,000
index response bytes      <= 500 MiB
WARC records              <= 3,000
compressed WARC bytes     <= 6,000 MiB
total run network stop    8 GiB
```

The funding demonstration begins with 100 or fewer archive profiles, a
2 GiB retained-evidence stop and an actual-count report. Bytes are stored and
hashed before WARC parsing. Archive-derived packets must state
`archive_observation`, `historical`, `current_publisher_statement=false`, the
Common Crawl collection/capture metadata and `observed_by=common_crawl`.
Archive observations never imply current publisher state.

## Public proof acceptance criteria

The first read-only demonstration is admitted only when it can show actual,
not target, counts for:

- Atlas identities and archive profiles;
- immutable observations and semantic packet batches;
- two cross-origin materialized views;
- one historical semantic delta stream;
- one downloadable, offline-verifiable snapshot proof;
- fixed semantic queries with native terms, lexical values, mappings and
  source evidence preserved;
- query latency, transferred bytes, proof bytes and interpretation reuse;
- zero semantic-state writes and zero arbitrary-origin execution on Meridian.

The aspirational 500 origins, 25,000 packets and 100,000-packet stretch goal
remain labeled as targets until generated evidence proves their exact counts.

## Snapshot-to-PostgreSQL migration

The immutable snapshot is not a competing authority. A future durable
PostgreSQL deployment imports the same canonical packet and delta bytes:

1. verify the snapshot and all existing artifact identities offline;
2. import origin, contract, canon and mapping identities;
3. admit packet/delta digests transactionally and preserve sequence data;
4. rebuild current heads and materializations from the immutable log;
5. compare rebuilt view digests and fixed query results with the snapshot;
6. keep the snapshot runtime available as a rollback read path;
7. cut over mutable production reads only after recovery and founder admission.

PostgreSQL may assign operational sequence numbers, but cannot redefine packet,
delta, batch, snapshot, evidence or mapping identity.

## Commands executed

The public form preserves command arguments while redacting the SSH account
and host endpoint; those identifiers are not protocol evidence.

```bash
git status --short
git branch --show-current
ssh <operator>@<meridian-host> 'uname -a; systemd --version; lscpu'
ssh <operator>@<meridian-host> 'free -b'
ssh <operator>@<meridian-host> 'df -B1 -T / /data; df -i / /data'
ssh <operator>@<meridian-host> 'cat /proc/mdstat'
ssh <operator>@<meridian-host> 'du -sb /srv/twirx'
ssh <operator>@<meridian-host> 'systemctl is-active caddy docker containerd postgresql'
ssh <operator>@<meridian-host> 'systemctl show caddy -p MemoryCurrent -p MemoryPeak -p MemoryMax'
ssh <operator>@<meridian-host> 'sudo ufw status verbose'
```

## References

- Common Crawl collection list: <https://index.commoncrawl.org/collinfo.json>
- Common Crawl CDXJ index and byte-range retrieval: <https://commoncrawl.org/cdxj-index>
- Hetzner Object Storage versioning and lifecycle: <https://docs.hetzner.com/storage/object-storage/howto-protect-objects/protect-versioning/>
- Hetzner Storage Box Borg access: <https://docs.hetzner.com/storage/storage-box/access/access-ssh-rsync-borg/>
- Hetzner Storage Box snapshot limitation: <https://docs.hetzner.com/storage/storage-box/snapshots/>

## Unresolved risks

- Meridian root RAID remains degraded and swap pressure is unexplained.
- No Object Storage bucket/policy/versioning evidence has yet been captured.
- No encrypted Storage Box repository or restore drill has yet passed.
- Systemd 257 cannot enforce the intended directory-level disk quota.
- Snapshot codecs and independent verification are not yet implemented.
- The archive importer and its network adversarial suite are not yet built.
- Public demo counts and performance remain acceptance targets, not results.
