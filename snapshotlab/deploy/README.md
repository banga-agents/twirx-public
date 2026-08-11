# Immutable Semantic Snapshot Query Lab deployment

This release exposes one verified read-only snapshot. It does not retrieve an
origin, run an adapter, accept a URL, use a browser or model, authenticate,
execute a payment, or write semantic state.

## Release layout

```text
/srv/twirx-snapshot/releases/<release-id>/
  bin/twirx-snapshot
  snapshot/
/srv/twirx-snapshot/current -> one immutable release

/srv/twirx-snapshot-ui/releases/<release-id>/
  index.html
  app.js
  styles.css
  queries/
/srv/twirx-snapshot-ui/current -> one immutable UI release
```

The committed systemd unit binds the runtime to literal loopback
`127.0.0.1:8092`. Caddy is the only public edge. Raw proof-artifact downloads
are denied at the edge while the third-party archive-redistribution treatment
is unresolved; packet, delta and snapshot-manifest CBOR remain downloadable.
The static UI is in a separate Caddy-readable release tree, so Caddy receives
no filesystem access to the snapshot's retained raw evidence.

## Host invariants

- dedicated `twirx-snapshot` user with no shell and no home;
- no repository, deployment, database, object-storage or origin credentials;
- application release and snapshot read-only;
- `MemoryMax=256M`, `MemorySwapMax=0`, `CPUQuota=25%`, `IOWeight=10`;
- only loopback IP traffic allowed by the service unit;
- port 8092 absent from public firewall rules;
- Caddy request body limited to 64 KiB;
- runtime concurrency limited to eight queries;
- no access log enabled for this virtual host.

## Admission order

1. Build `cmd/twirx-snapshot` at the reviewed source revision.
2. Verify the snapshot and exact expected ID locally.
3. Create two immutable releases: a private runtime release containing the
   binary and snapshot, and a separate Caddy-readable static UI release.
4. Compare local and remote SHA-256 manifests.
5. Install the systemd unit but do not start it yet.
6. Verify the release is readable only by root and `twirx-snapshot`.
7. Start the service and query `127.0.0.1:8092/api/v1/status` and the bounded
   `127.0.0.1:8092/api/v1/origins?limit=500` Atlas view locally.
8. Confirm the process has the expected user, limits, address and no public socket.
9. Install and validate the Caddy site file.
10. Add `CNAME lab twirx.org.` only after service and edge configuration are healthy.
11. Verify HTTPS, headers, preset queries, trace, packet/delta/manifest downloads,
    raw-proof denial, method/body limits and zero origin calls.

Do not modify the apex, `www`, `docs`, Proton Mail, DKIM, SPF, DMARC, MX or
Mintlify verification records during this activation.
