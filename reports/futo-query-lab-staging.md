# FUTO Query Lab staging report

**Status:** STAGED — refreshed immutable runtime and Atlas-500 explorer pass on
literal loopback; public activation awaits authoritative DNS.

**Staging date:** 2026-08-11

**Source revision:**
`95a537a0a2d54e160a4b67643be2f637f5a7bea5`

**Snapshot ID:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`

**Runtime release:**
`/srv/twirx-snapshot/releases/20260811T153300Z-95a537a`

**UI release:**
`/srv/twirx-snapshot-ui/releases/20260811T153300Z-95a537a`

**Runtime binary SHA-256:**
`cbb102bedbbcc914a974f39b723c4a795fed6a13629c53b7de0e6a352ba36a9a`

## Result

The immutable Semantic Snapshot Query Lab is installed on the Meridian host
and runs as a dedicated unprivileged service bound only to literal loopback at
`127.0.0.1:8092`. Its Caddy virtual host is installed and validates, but it is
not imported into the active Caddy configuration. No public Lab execution is
claimed yet.

The refreshed service exposes one exact admitted snapshot and the complete
500-origin identity catalog. It can search and inspect all selected origins,
run the committed cross-origin and archive-history queries, return packet
traces, and serve packet, delta and snapshot-manifest CBOR. It cannot retrieve
an origin, refresh the snapshot, accept a URL, execute an adapter, browser,
model, payment or action, or write semantic state.

The Atlas endpoint reports exactly:

```text
500 selected origin identities
500 origin records returned across the complete catalog
3 origins with admitted public packets
15 public packets total
5 controlled fixture packets excluded by default
```

The difference is intentional and visible: the 500 are selected catalog
identities, while only the three explicitly policy-approved scopes currently
have admitted source-derived packets. This is a human admission boundary, not
a technical maximum.

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

The service is active and enabled. Its refreshed runtime uses approximately
12 MiB at idle and retains the same hard systemd limits.

## Publication profile

Raw third-party archive bodies and WARC records are not exposed. The Caddy
site returns `404` for `/api/v1/proof/*`, while packet, delta, snapshot
manifest and trace endpoints remain available. This publishes TWIRX-authored
derivations, digests and bounded reproduction metadata while retaining raw
third-party representations privately pending a later rights review.

Archive-derived packets describe historical Common Crawl captures, not a
current publisher statement and not objective truth. This is an engineering
publication boundary, not a legal opinion.

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
  'http://127.0.0.1:8092/api/v1/origins?offset=0&limit=500'
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

The following literal-loopback behaviors passed:

- status returns the exact snapshot ID and `execution` value
  `immutable_materialized_snapshot_only`;
- origin pagination returns all 500 selected identities and exact public
  packet counts;
- unknown origin identifiers return HTTP `404`;
- unknown pagination parameters fail closed;
- cross-origin query returns four rows from TWIRX and World Bank, with zero
  origin calls and all five fixtures excluded;
- archive-history query returns the two exact RFC Editor source-native values,
  with zero origin calls and all five fixtures excluded;
- a query containing an unknown `url` field fails closed with HTTP `400`;
- a connection to public `116.202.50.220:8092` times out.

## Remaining activation work

1. Publish `CNAME lab twirx.org.` at the authoritative Namecheap DNS service.
2. Wait until both authoritative Namecheap nameservers return the record.
3. Import `/etc/caddy/twirx-snapshot-lab.caddy` into the active Caddyfile,
   validate the complete configuration and reload Caddy without restarting it.
4. Verify TLS, exact preset query results, headers, body/method limits, raw
   proof denial, browser rendering, zero cookies and zero origin calls over the
   public hostname.
5. Deploy the release-bound website that links the verified public Lab.

At the report time, direct queries to `dns1.registrar-servers.com` still return
no CNAME, A or AAAA response for `lab.twirx.org`; therefore Caddy activation
would fail certificate issuance and is intentionally deferred.

No apex, `www`, `docs`, Proton Mail, DKIM, SPF, DMARC, MX or Mintlify record was
modified by this staging gate.

## Unresolved risks

- The Lab is not publicly reachable until DNS and edge activation pass.
- Off-host snapshot durability and a clean independent restore remain
  unverified.
- The current measurement is a small immutable snapshot on a shared host; it
  is not a production-capacity claim.

## Next recommended gate

Activate and wire-verify `lab.twirx.org`, then deploy the synchronized website
and complete the off-host restore test using TWIRX-specific credentials and a
new encrypted repository path.
