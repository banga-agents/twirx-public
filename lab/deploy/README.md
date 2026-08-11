# E2 VPS deployment

Do not activate this service or add `lab.twirx.org` DNS until its release is
reviewed. The HTTP service is replay-only: fresh-origin execution is disabled
and no request field can select a URL or destination. The Lab process binds
only to `127.0.0.1:8090`; Caddy owns the public edge and TLS. Port 8090 must
remain closed in the host firewall.

## Filesystem and service ownership

```text
/srv/twirx-lab/releases/<release-id>  immutable release
/srv/twirx-lab/current               atomic symlink to one release
/var/lib/twirx-lab/results           only writable application path
```

Create a dedicated system user with no login shell or home directory. Install
`twirx-lab.service` under systemd and adapt only the explicit release paths if
the VPS layout differs. Do not give this user access to the main website
release, repository credentials, PostgreSQL, Proton Mail records, or any other
repository on the VPS.

The committed unit applies a read-only application filesystem, an explicit
writable result directory, empty Linux capabilities, namespace and kernel
protections, a syscall allowlist, task/file/memory limits, and automatic
restart. Fresh external-origin access remains disabled until it is moved
behind the separately admitted egress worker and host-level controls; the
application URL policy is not a substitute for those controls.

## Activation order

1. Build and run the complete offline E2 suite on the proposed release.
2. Copy only the reviewed release into a new immutable directory.
3. Verify its checksum and switch `/srv/twirx-lab/current` atomically.
4. Start the systemd unit and verify `127.0.0.1:8090/api/v1/status` locally.
5. Import the Caddy site block and validate Caddy configuration.
6. Verify public HTTP/TLS on a temporary host override.
7. Add `CNAME lab twirx.org.` only after the service is healthy.
8. Verify every denial in `scripts/check-lab-surface.sh`.

Do not modify the existing apex, `www`, `docs`, Proton Mail, DKIM, SPF, DMARC,
MX, or Mintlify verification records during this activation.
