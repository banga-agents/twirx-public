# FUTO Query Lab staging report

**Status:** STAGED — private loopback validation complete; public activation
awaits the `lab.twirx.org` DNS record.

**Staging date:** 2026-08-11

**Source revision:**
`212e6ce2256b`

**Snapshot ID:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`

**Runtime release:**
`/srv/twirx-snapshot/releases/20260811T141700Z-212e6ce2256b`

**UI release:**
`/srv/twirx-snapshot-ui/releases/20260811T141700Z-212e6ce2256b`

**Runtime binary SHA-256:**
`e627ecea95e1e2d99493896553add83eee83a55b0a79af4e1820381877d79a11`

## Result

The immutable Semantic Snapshot Query Lab is installed on the Meridian host
and runs as a dedicated unprivileged service bound only to literal loopback at
`127.0.0.1:8092`. Its Caddy virtual host is installed and validates, but it is
not imported into the active Caddy configuration. No public Lab execution is
claimed.

The service exposes one exact admitted snapshot. It can run the committed
cross-origin and archive-history queries, return packet traces, and serve
packet, delta and snapshot-manifest CBOR. It cannot retrieve an origin,
refresh the snapshot, accept a URL, execute an adapter, browser, model,
payment or action, or write semantic state.

## Isolation and resource invariants

- service user and group: `twirx-snapshot` with no login shell;
- runtime and snapshot: read-only to the service and unreadable by Caddy;
- static UI: separate Caddy-readable release and unreadable by the runtime;
- listener: literal `127.0.0.1:8092` only;
- public firewall: no rule for port `8092`;
- `NoNewPrivileges=yes` and empty capability sets;
- `ProtectSystem=strict`, private devices/tmp and restricted namespaces;
- address families limited to Unix, IPv4 and IPv6;
- all service IP traffic denied except loopback;
- `MemoryMax=256M` and `MemorySwapMax=0`;
- `CPUQuota=25%`, `IOWeight=10`, `Nice=10`;
- `TasksMax=32` and `LimitNOFILE=1024`;
- query concurrency cap: eight;
- Caddy request-body cap: 64 KiB;
- no Caddy access log for the Lab virtual host.

Observed after activation:

```text
ActiveState=active
SubState=running
MemoryCurrent=13,643,776 bytes
MemoryPeak=17,174,528 bytes
TasksCurrent=13
```

## Publication profile

Raw third-party archive bodies and WARC records are not exposed. The Caddy
site returns `404` for `/api/v1/proof/*`, while packet, delta, snapshot
manifest and trace endpoints remain available. This implements the
conservative public-release treatment: publish TWIRX-authored derivations,
digests and bounded reproduction metadata, but keep retained third-party raw
representations private unless a later rights review approves redistribution.

The treatment follows two verified boundaries:

1. Common Crawl's Terms of Use permit access and use of the service but state
   that crawled content may be subject to the source owner's separate terms
   and rights.
2. The RFC Editor expressly permits reuse of RFC documents; that permission
   does not by itself establish reuse rights for archived homepage HTML.

This is an engineering publication boundary, not a legal opinion.

## Commands executed

Local build and validation:

```bash
cmp snapshotlab/static/queries/cross-origin.json \
  examples/semantic-query-two-origins.json
cmp snapshotlab/static/queries/archive-history.json \
  examples/semantic-query-rfc-editor-history.json
node --check snapshotlab/static/app.js
GOMAXPROCS=2 go test ./cmd/twirx-snapshot ./internal/snapshotruntime
GOMAXPROCS=2 go vet ./cmd/twirx-snapshot ./internal/snapshotruntime
make bin/twirx-snapshot
bin/twirx-snapshot verify \
  --snapshot var/futo-public-snapshot-d13c0bf-rebuilt \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5
git diff --check
```

Target-host validation:

```bash
systemctl is-active twirx-snapshot-lab.service
systemctl is-enabled twirx-snapshot-lab.service
ss -ltnp
curl --fail --silent --show-error \
  http://127.0.0.1:8092/api/v1/status
systemctl show twirx-snapshot-lab.service \
  -p User -p Group -p MemoryMax -p MemorySwapMax \
  -p CPUQuotaPerSecUSec -p IOWeight -p TasksMax \
  -p NoNewPrivileges -p IPAddressDeny -p IPAddressAllow
sudo caddy validate \
  --config /etc/caddy/twirx-snapshot-lab.caddy \
  --adapter caddyfile
sudo -u caddy test ! -r /srv/twirx-snapshot/current/snapshot
sudo -u twirx-snapshot \
  test ! -r /srv/twirx-snapshot-ui/current/index.html
```

The following loopback behaviors also passed:

- status returns the exact snapshot ID and `execution` value
  `immutable_materialized_snapshot_only`;
- cross-origin query returns four rows from TWIRX and World Bank, with zero
  origin calls and all five fixtures excluded;
- archive-history query returns the two exact RFC Editor source-native values,
  with zero origin calls and all five fixtures excluded;
- a query containing an unknown `url` field fails closed with HTTP `400`;
- a connection to public `116.202.50.220:8092` times out.

## Remaining activation work

1. Add the single DNS record `CNAME lab twirx.org.`.
2. Wait for public resolution.
3. Import `/etc/caddy/twirx-snapshot-lab.caddy` into the active Caddyfile,
   validate the complete configuration and reload Caddy without restarting it.
4. Verify TLS, exact preset query results, headers, body/method limits, raw
   proof denial, browser rendering, zero cookies and zero origin calls over the
   public hostname.
5. Bind the website and final FUTO readiness report to the public Lab result.

No apex, `www`, `docs`, Proton Mail, DKIM, SPF, DMARC, MX or Mintlify record was
modified by this staging gate.

## Unresolved risks

- The Lab is not publicly reachable until DNS and edge activation pass.
- Off-host snapshot durability and a clean independent restore remain
  unverified.
- The current measurement is a small immutable snapshot on a shared host; it
  is not a production-capacity claim.

## Next recommended gate

Activate and wire-verify `lab.twirx.org`, then update the public website to
link the live Lab and complete the off-host restore test.
