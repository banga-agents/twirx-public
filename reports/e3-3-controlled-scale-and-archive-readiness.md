# E3.3 controlled scale and archive-readiness evidence

**Controlled-capacity recommendation:** PASS

**Archive/public-corpus recommendation:** FAIL — policy admission incomplete

**Production-state recommendation:** NOT ADMITTED

**Implementation commit:**
`c433ad33d5ba76f5ae0112fd3b58f89cf293e6ca`

**Evidence date:** 2026-08-11

## Result

The implementation commit compiles, verifies and queries an immutable
25,018-packet Semantic Snapshot. The complete runtime path handled 5,000
concurrent-loopback HTTP queries without failure or origin-network access. A
second build from the same inputs was byte-identical.

This result proves controlled packet-scale behavior. It does not satisfy the
public corpus target: only 13 packets derive from two existing public E2 replay
origins, while 25,005 packets are explicitly controlled fixtures. Fixtures are
separately counted and are absent from public views, queries, traces and HTTP
runtime defaults. No archive profile, current observation or new semantic
mapping is claimed.

## Exact snapshot

```text
source revision: c433ad33d5ba76f5ae0112fd3b58f89cf293e6ca
created_at:       2026-08-11T00:00:00Z
snapshot_id:      sha256:0677c3e37b20a1fd39c7010433e5cb81d92118f6b99f41b14ad4c3f8deed3f82
manifest sha256:  0677c3e37b20a1fd39c7010433e5cb81d92118f6b99f41b14ad4c3f8deed3f82
snapshot bytes:   52536074
snapshot files:   74
build duration:   17569814 microseconds
```

The exact rebuild had the same snapshot ID, manifest bytes and complete
directory contents.

### Actual content

| Measure | Actual | Publicly claimable |
| --- | ---: | ---: |
| Atlas identities carried | 500 | 500 catalog identities |
| Archive/index profiles | 0 | 0 |
| Public origins with packets | 2 | 2 |
| Controlled fixture origins with packets | 2 | 0 |
| Operations represented | 6 | 5 |
| Total packets | 25,018 | 13 |
| Controlled fixture packets | 25,005 | 0 |
| Resolved packets | 25,018 | 13 |
| Semantic deltas | 0 | 0 |
| Materialized views | 2 | 2 small E2 replay views |
| Embedded proof artifacts | 53 | 50 E2 artifacts |

The scale corpus consists of source-native JSON fields named `field_00000`
through `field_24999`, with lexical values `value_00000` through
`value_24999`. They use the `observed_native` lane and `none` mapping status.
No generated field is aligned to a shared semantic concept or admitted to the
canon.

## Implemented invariants

- The scale representation and extraction plans are deterministic and bounded.
- Every scale packet preserves its native term, exact lexical value, JSON
  locator, representation digest, observation digest and extraction-plan
  digest.
- The controlled observation envelope is compatible with the existing
  independent observation profile and binds the complete representation body.
- Packet batches and proof indexes split at 4,096 entries and remain below the
  16 MiB local artifact ceiling.
- Proof-index parsing rejects unknown fields, invalid proof types, malformed
  identities, unsafe paths, zero-size constituents, duplicates and bad order.
- Runtime admission rehashes all artifacts, parses every canonical packet,
  reconstructs every controlled field and extraction plan, and reconciles all
  25,018 packet/proof relationships before serving.
- Controlled fixtures are excluded unless an operator uses the explicit
  `--include-fixtures` loopback-conformance flag. The candidate systemd service
  does not set that flag.
- Public and fixture packet counters are independently reconciled on import.
- Public materialized views continue to contain only recorded E2 replay
  evidence.
- There is no origin fetch, live refresh, semantic-state write, browser, model,
  action, payment, authentication or arbitrary-URL path.
- E1, E2, E3.2 and E3.3 S1 conformance behavior remains unchanged.
- No third-party runtime dependency was added.

## Scale stress result

```text
requests:                         5000
concurrency:                      8
successes:                        5000
failures:                         0
runtime origin-network requests: 0
duration:                         2341957 microseconds
throughput:                       2134.966611 requests/second
p50:                              3516 microseconds
p95:                              4417 microseconds
p99:                              4827 microseconds
maximum RSS:                      250340 KiB
maximum threads:                  21
maximum file descriptors:         14
```

The stable controlled query selected `field_24999`. Its query identity was
`sha256:f2000351b99715140eff464590992a95463fa71c903c01461cbfa28766239850`;
its result identity was
`sha256:02c39ef6f1076bf1331b84ce35f3059d1f3e0d20095f2f6ea74b028248aedda3`.
The public-default invocation returned `unresolved`; the explicit local
fixture invocation returned the exact native value `value_24999`.

The restricted-C verifier accepted the snapshot manifest and 14 canonical
packet samples spanning the first and last entries of all seven packet
segments. Go admission verified every packet and every proof relationship.

This is local evidence on the recorded host and controlled corpus. It is not a
Meridian benchmark, public-origin latency claim or proof of ontology quality.

## Atlas policy blocker

The 25 admission dossiers currently contain:

```text
catalog review:  2 completed, 23 pending
policy review:   0 completed, 25 pending
policy decision: 25 uncertain
```

The bounded Common Crawl design requires a reviewed work order bound to a
completed policy-decision evidence digest. Creating archive work orders now
would bypass that invariant. Therefore the exact honest archive count is zero,
and no network importer was executed.

This is the remaining human-admission dependency, not a packet-runtime
capacity failure.

## Meridian read-only preflight

A read-only SSH measurement was performed; no host state was changed.

```text
systemd:              257 (257.13-1~deb13u1)
logical CPUs:         16
CPU:                  Intel Xeon W-2145 @ 3.70 GHz
RAM total:            131575676 KiB
RAM available:        74257156 KiB
swap total:           4189180 KiB
swap free:            592 KiB
root filesystem:      938524024832 bytes
root used:            507938521088 bytes
root available:       382834192384 bytes
/srv/twirx usage:     2477405 bytes
RAID md2:             degraded RAID1 [2/1] [U_]
RAID md0/md1:         healthy RAID1 [UU]
```

The host has ample CPU, RAM and disk for a replaceable 52.5 MB snapshot, but
the degraded data RAID and near-exhausted swap preserve the founder block on
authoritative PostgreSQL, compiler state or a broad corpus on Meridian. No
service, RAID, firewall, Caddy, repository or file was modified.

The repository candidate remains limited to an 80 GB disk ceiling, 25% CPU,
20% RAM, zero process swap, low I/O priority, bounded logs and loopback-only
query service. Target-host enforcement is a separate production-admission
gate.

## Commands executed

The complete local suite and scale proof passed on the implementation commit:

```bash
gofmt -d cmd/twirx-snapshot/*.go \
  internal/scalefixture/*.go \
  internal/snapshotartifact/*.go \
  internal/snapshotbuild/*.go \
  internal/snapshotruntime/*.go

git diff --check
go vet ./...
GOMAXPROCS=2 go test -race ./...
GOMAXPROCS=2 make test
make stress-semantic-snapshot-scale

bin/twirx-snapshot build \
  --root . \
  --out /tmp/twirx-scale-rebuild.EIiAqH/snapshot \
  --source-revision c433ad33d5ba76f5ae0112fd3b58f89cf293e6ca \
  --created-at 2026-08-11T00:00:00Z \
  --scale-fixture-packets 25000

cmp var/e3-semantic-snapshot-stress/snapshot/manifest.cbor \
  /tmp/twirx-scale-rebuild.EIiAqH/snapshot/manifest.cbor

diff -qr var/e3-semantic-snapshot-stress/snapshot \
  /tmp/twirx-scale-rebuild.EIiAqH/snapshot

sha256sum var/e3-semantic-snapshot-stress/snapshot/manifest.cbor
```

The full `make test` run included 18 Go fuzz targets, all Go package tests, the
E1 and E2 restricted-C suites, 56 E3.3 S1 C vectors (16 valid accepted and 40
invalid rejected), three 5,000-run restricted-C libFuzzer campaigns under ASan
and UBSan, E2 end-to-end replay, the baseline Semantic Snapshot integration and
documentation navigation validation.

The Meridian preflight command was read-only:

```bash
ssh agent 'set -eu
printf "systemd="
systemctl --version | sed -n "1p"
printf "cpu_count="
getconf _NPROCESSORS_ONLN
awk -F: "/model name/ {
  gsub(/^[ \\t]+/, \"\", \$2)
  print \"cpu_model=\" \$2
  exit
}" /proc/cpuinfo
awk "/MemTotal:/ {print \"mem_total_kib=\" \$2}
     /MemAvailable:/ {print \"mem_available_kib=\" \$2}
     /SwapTotal:/ {print \"swap_total_kib=\" \$2}
     /SwapFree:/ {print \"swap_free_kib=\" \$2}" /proc/meminfo
df -B1 --output=size,used,avail,pcent,target / /srv/twirx 2>/dev/null
du -sb /srv/twirx 2>/dev/null
cat /proc/mdstat
uptime'
```

## Toolchain and local host

```text
Go:     go1.26.5-X:nodwarf5 linux/amd64
GCC:    16.1.1
Clang:  22.1.8
Kernel: Linux 7.1.3-arch1-1 x86_64
CPU:    AMD Ryzen 7 6800U with Radeon Graphics
```

## Unresolved risks and limitations

1. Zero Atlas policy reviews are completed, so zero Common Crawl profiles are
   admitted and the real public corpus remains 13 packets from two E2 origins.
2. The scale corpus tests capacity, not semantic diversity, mapping quality,
   ontology traversal, publisher authority or current freshness.
3. Query execution is a bounded linear in-memory scan. It passed at 25,018
   packets but needs measured indexing before substantially larger snapshots.
4. The JSON proof carriage is intentionally explicit and makes this snapshot
   52.5 MB. Compression and compact indexes require a separately specified,
   independently verifiable format rather than an unreviewed optimization.
5. Restricted C sampled 14 scale packets; it did not iterate all 25,018 packet
   objects. Complete Go admission and shared C conformance still passed.
6. There is one snapshot and therefore no genuine origin, semantic or canon
   delta stream.
7. Object Storage versioning, immutable upload, Storage Box encrypted backup
   and isolated restore proof remain unexecuted.
8. Meridian production limits, filesystem ownership, atomic activation, Caddy
   route and rollback remain unverified; the degraded RAID and swap pressure
   remain external host risks.

## Deviations

No Common Crawl or live-origin request was performed because the required
human policy decisions do not exist. A controlled scale corpus was implemented
instead to answer the independent question of whether the actual snapshot
compiler and query runtime can operate at 25,000 packets. It is not counted
toward public funding-demo packet or origin targets.

No Object Storage, Storage Box, PostgreSQL, Meridian service, DNS, Caddy,
firewall or deployment state was changed. No merge or deployment was
performed.

## Next recommended gate

Complete explicit human policy review for an initial archive pilot, then
implement the sealed Common Crawl importer and its full offline adversarial
suite from `deploy/snapshot/COMMON_CRAWL_IMPORT.md`. Execute only admitted work
orders, preserve archive capture metadata and historical classification, and
compile the first real archive-derived packets into a new deterministic
snapshot. Increase to 100 archive profiles only after the pilot proves budgets,
failure handling and review effort. Production edge activation remains a
separate founder-reviewed gate.
