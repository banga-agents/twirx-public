# Secure egress pilot deployment candidate

Status: implemented configuration candidate; not installed, activated, or
deployed by E3.2 repository commands.

The public-origin worker executes one sealed, read-only work order by ID. The
client cannot supply a URL. A root-owned work-order artifact binds the exact
HTTPS URL, admitted host names, policy and decision digests, approval
reference, validity interval, redirect limit, byte limit, timeouts, and
circuit-breaker thresholds.

## Isolation boundaries

- `twirx-egress` is a dedicated unprivileged account.
- Work orders and the kill-switch/revocation file are root-owned and read-only
  to the worker.
- Only the evidence spool and bounded circuit/concurrency state are writable.
- The repository Atlas, reports, `.git`, deployment configuration, home
  directories, PostgreSQL state, credentials, and secret paths are hidden.
- Application checks reject non-HTTPS schemes, credentials, numeric address
  forms, non-standard ports, unadmitted redirects, and every resolved private,
  loopback, link-local, multicast, metadata, documentation, benchmark, or
  reserved address.
- `IPAddressDeny=` provides a second, cgroup-scoped network firewall for the
  denied IPv4 and IPv6 ranges. It does not alter networking for unrelated VPS
  services.
- A process lock caps global worker concurrency at one for this pilot.
- `MemoryMax`, `TimeoutStartSec`, `CPUQuota`, `TasksMax`, and `LimitNOFILE` bound
  resource use.
- The final manifest is written last. Downstream code must call `verify`
  before parsing an observation.

## Deliberately absent capabilities

There is no arbitrary URL argument, scheduler, browser, model, adapter,
database client, registry mutation, deployment access, credential loading,
write action, or payment action. The service template has no install target
and the committed control file is disabled with its emergency stop active.

## Operator-only pilot sequence

Founder review is required before any of these installation steps. An
operator must separately create the account and directories, install an
immutable reviewed binary, copy the unit, and copy a reviewed control file.
Work orders must be generated from completed human admission decisions by the
control plane; they must never be composed from visitor input.

Before activation, run on the target host:

```bash
deploy/egress/verify-target.sh /etc/systemd/system/twirx-egress-worker@.service
systemd-analyze security --offline=yes /etc/systemd/system/twirx-egress-worker@.service
```

The first command also proves that no instance is running. Neither command
installs, enables, starts, or changes the unit or host firewall.

## Current target-host blocker

The 2026-08-11 read-only check found systemd 257 and active Caddy, but no
egress unit or state directory. The target's current resolvers are Tailscale
MagicDNS addresses inside ranges denied by this unit. Installing the candidate
unchanged would therefore fail DNS resolution safely.

Do not solve this by broadly allowing private egress. Before activation, the
operator must provide a reviewed resolver boundary—such as a dedicated
port-scoped DNS broker or isolated worker network—and rerun DNS rebinding,
private-range, and control-plane reachability tests on the target. This
repository does not claim that target-host firewall enforcement is active.
