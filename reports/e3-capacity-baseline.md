# E3 VPS capacity baseline

**Measured:** 2026-08-10

**Host role:** Shared VPS candidate for a bounded Atlas control plane

**Method:** Read-only inspection over the existing SSH-agent connection

## Measured resources

| Resource | Observed value |
|---|---:|
| Kernel / architecture | Linux 6.12.90, x86-64 |
| Logical CPUs | 16 |
| CPU model | Intel Xeon W-2145 at 3.70 GHz |
| Physical topology reported | 8 cores, 2 threads per core |
| RAM total | 134,733,492,224 bytes |
| RAM available at observation | 72,560,648,192 bytes |
| Swap total | 4,289,720,320 bytes |
| Swap used at observation | 4,183,543,808 bytes |
| Root filesystem | ext4 on `/dev/md2` |
| Root filesystem total | 938,524,024,832 bytes |
| Root filesystem available | 400,292,069,376 bytes |
| Root filesystem inode use | 4% |
| Public IPv4 / IPv6 | Present |

The provider-side bandwidth cap, sustained throughput, IOPS, backup capacity,
and failure-domain guarantees were not discoverable from the guest. They are
unmeasured and must not be inferred from interface presence.

## Boundary observations

- UFW was active with default deny for inbound and routed traffic.
- Only TCP 22, 80, and 443 were allowed by UFW for IPv4 and IPv6.
- Caddy was active and its installed configuration validated.
- The shared host has numerous unrelated listeners and workloads. Their
  presence makes raw free-resource numbers unsuitable as Atlas allocations.
- The Caddy unit reported `PrivateTmp=yes` and `ProtectSystem=full`, but also
  `NoNewPrivileges=no`, `ProtectHome=no`, and no service memory or CPU cap.
- Swap was almost fully occupied during the observation. This is a capacity
  warning, not proof of memory exhaustion; workload attribution was outside
  scope and no unrelated service was inspected.

No file, service, firewall rule, DNS record, repository, or deployment was
changed. The unrelated `meridian-velo` repository was not accessed.

## Conservative initial Atlas budget

These are admission ceilings, not claims about provider capacity:

| Control | Initial ceiling |
|---|---:|
| Atlas service CPU | 4 logical CPUs |
| Atlas service memory | 8 GiB |
| Raw retained evidence | 25 MiB per reviewed origin; 20 documents maximum |
| Atlas retained corpus | 25 GiB soft alert; 50 GiB hard stop before review |
| Response body | 2 MiB per representation |
| Expanded/decompressed body | 4 MiB |
| Direct-fetch concurrency | 4 global; 1 per origin |
| Redirects | 3, with policy and address revalidation on every hop |
| Default origin request rate | At most 1 request per minute, lower when policy requires |
| Worker wall time | 20 seconds per request |

The scheduler must begin in deterministic dry-run mode. Live egress remains
disabled until a dedicated user or rootless container, explicit destination
allowlist, private/metadata-range network blocking, DNS-rebinding tests,
redirect revalidation, writable-path restrictions, quotas, and incident
controls pass together.

## Commands executed

```bash
ssh -o BatchMode=yes -o ConnectTimeout=10 agent@116.202.50.220 \
  'set -eu; uname -srmo; printf "CPU_COUNT="; getconf _NPROCESSORS_ONLN; \
   lscpu | sed -n "/^Architecture:/p;/^Model name:/p;/^CPU(s):/p;/^Thread(s) per core:/p;/^Core(s) per socket:/p"; \
   free -b; df -B1 --output=source,fstype,size,used,avail,pcent,target / /var \
   2>/dev/null | sed -n "1p;2p;4p"; findmnt -no SOURCE,FSTYPE,OPTIONS /; \
   ip -brief address show scope global; ip route show default; \
   command -v tc >/dev/null && tc qdisc show || true; \
   systemctl is-active caddy 2>/dev/null || true; \
   systemctl is-active nftables 2>/dev/null || true; \
   systemctl is-active ufw 2>/dev/null || true; \
   ss -lntH | awk "{print \$4}" | sort -u'

ssh -o BatchMode=yes agent@116.202.50.220 \
  'set -eu; printf "%s\n" "--- firewall ---"; \
   sudo -n ufw status verbose 2>&1 || true; printf "%s\n" "--- link ---"; \
   (ethtool eno2 2>/dev/null | sed -n "/Speed:/p;/Duplex:/p;/Link detected:/p") || true; \
   printf "%s\n" "--- caddy sites ---"; \
   sudo -n caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1 || true; \
   printf "%s\n" "--- system limits ---"; \
   systemctl show caddy -p User -p Group -p NoNewPrivileges -p ProtectSystem \
   -p ProtectHome -p PrivateTmp -p MemoryMax -p CPUQuotaPerSecUSec -p TasksMax \
   --no-pager 2>/dev/null || true; printf "%s\n" "--- disk inodes ---"; \
   df -i / | sed -n "1,2p"'
```

## Capacity recommendation

**PASS for E3.0 control-plane development and offline catalog validation.**

**FAIL for continuous live ingestion or public fresh-origin execution** until
the egress worker and service limits are implemented, deployed, and tested.
