# E3.3 VPS capacity and deployment baseline

Recommendation: **FAIL for database deployment; PASS for continued offline design**

Observed: 2026-08-11

Target: TWIRX VPS, accessed through an existing operator account

Repository base: `9aebad7ffaa888c070352bf8247b21e24e6b5213`

No host state was changed during this audit.

## Measured capacity

| Resource | Observation |
| --- | --- |
| Operating system | Debian GNU/Linux 13.5 |
| Kernel | Linux 6.12.90 |
| Init | systemd 257 |
| CPU | Intel Xeon W-2145, 8 physical cores / 16 logical CPUs |
| Memory | 134,733,492,224 bytes total; approximately 72.6 GB available at observation |
| Swap | 4,289,720,320 bytes total; approximately 4.14 GB used |
| Root filesystem | ext4 on `/dev/md2`, approximately 938.5 GB total, 386.2 GB available, 57% used |
| Root inodes | approximately 4% used |
| Secondary filesystem | `/dev/nvme1n1p3` mounted at `/data` and `/var/lib/docker` |
| PostgreSQL | not installed; no `psql`, `pg_config`, service or port 5432 listener |
| Public edge | Caddy active |
| Firewall | UFW active; public rules only for 22, 80 and 443 on IPv4 and IPv6 |
| Other workload | Docker and containerd active; multiple bridge networks; 24 running services observed |
| Resolver | Tailscale MagicDNS addresses, requiring explicit compatibility work for sealed egress |

## Critical storage finding

`/dev/md2` identifies itself as RAID1 but reported:

```text
active devices: 1
working devices: 1
array state: [U_]
```

Only `/dev/nvme0n1p3` was an active member. The nominal second device,
`/dev/nvme1n1p3`, is formatted and mounted separately for `/data` and Docker;
it is not an active mirror member. The root filesystem therefore has a known
single-device failure path despite the RAID1 configuration label.

The swap array (`md0`) and boot array (`md1`) reported healthy `[UU]` state.

## Deployment decision

Do not install PostgreSQL, initialize a cluster, change firewall rules, alter
RAID membership, move Docker data, or admit semantic state on this host yet.
The combination of degraded root storage, almost fully consumed swap and an
unprofiled shared workload makes the current durability and contention risk
unacceptable for authoritative operational state.

## Required remediation evidence

1. A host administrator identifies why `md2` is degraded and either restores
   redundancy or documents a founder-approved replacement durability design.
2. SMART/NVMe health and provider hardware status are captured for both
   devices without exposing unrelated data.
3. Swap consumers and peak memory behavior are measured; unexplained sustained
   swap pressure is resolved.
4. Docker and other service ownership, resource ceilings and disk growth are
   inventoried without changing the unrelated `meridian-velo` repository.
5. PostgreSQL storage path and its failure domain are selected explicitly.
6. Encrypted off-host base backup and WAL archive destinations are provisioned.
7. A full point-in-time restore drill passes before production admission.

## Initial capacity envelope after remediation

The following is a conservative starting allocation to validate, not a claim
of supported throughput:

| Component | Initial ceiling |
| --- | ---: |
| PostgreSQL service memory | 4 GiB |
| PostgreSQL connections | 40 |
| Semantic database soft disk budget | 50 GiB |
| Semantic database alert threshold | 75 GiB |
| Compiler/query service memory | 2 GiB |
| External-origin worker memory | existing E3.2 per-job limits; separate from database |
| Live external-origin concurrency | zero until separately activated |

The server has ample nominal CPU and memory for the initial design, but
nominal capacity does not cure a known storage failure path or prove safety
under the existing shared workload.

## Commands executed

The read-only audit used the following command families over SSH. The public
form redacts the administrative account and host endpoint because they are not
protocol evidence. Values that could reveal unrelated application metadata
were not copied into this report.

```bash
ssh <operator>@<meridian-host> 'uname -a'
ssh <operator>@<meridian-host> 'cat /etc/os-release'
ssh <operator>@<meridian-host> 'systemd --version'
ssh <operator>@<meridian-host> 'lscpu'
ssh <operator>@<meridian-host> 'free -b'
ssh <operator>@<meridian-host> 'df -B1 -T / /data /var/lib/docker'
ssh <operator>@<meridian-host> 'df -i / /data /var/lib/docker'
ssh <operator>@<meridian-host> 'cat /proc/mdstat'
ssh <operator>@<meridian-host> 'sudo mdadm --detail /dev/md0'
ssh <operator>@<meridian-host> 'sudo mdadm --detail /dev/md1'
ssh <operator>@<meridian-host> 'sudo mdadm --detail /dev/md2'
ssh <operator>@<meridian-host> 'lsblk -o NAME,TYPE,FSTYPE,SIZE,MOUNTPOINTS'
ssh <operator>@<meridian-host> 'systemctl is-active caddy postgresql docker containerd'
ssh <operator>@<meridian-host> 'systemctl show caddy -p MemoryCurrent -p MemoryPeak -p MemoryMax -p CPUQuotaPerSecUSec'
ssh <operator>@<meridian-host> 'command -v psql; command -v pg_config'
ssh <operator>@<meridian-host> 'ss -lntup'
ssh <operator>@<meridian-host> 'sudo ufw status verbose'
ssh <operator>@<meridian-host> 'cat /etc/resolv.conf'
ssh <operator>@<meridian-host> 'apt-cache policy postgresql-18'
```

## Unresolved risks

- Root storage is degraded and not presently redundant.
- The cause of near-total swap use is unknown.
- The resource behavior and growth of unrelated shared services are not yet
  bounded for coexistence with PostgreSQL.
- No off-host backup target or recovery evidence exists yet.
- Tailscale DNS and network-level E3.2 egress enforcement require a deliberate
  target-host design; neither should be changed casually.
