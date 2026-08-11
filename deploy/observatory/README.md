# Observatory worker deployment candidate

This directory contains an unactivated systemd template for the E3
local-fixture worker proof. It is not installed by repository tests and is not
a production public-origin egress policy.

Before activation, an operator must create the dedicated `twirx-observer`
account, install an immutable release under `/srv/twirx/current`, create
`/var/lib/twirx-observer`, and place fixed `TWIRX_JOB` and `TWIRX_OUTPUT`
values in `/etc/twirx/observer-worker`. The job validator still accepts only
literal `127.0.0.1` fixture URLs.

The service applies `IPAddressDeny=any` and `IPAddressAllow=localhost` in
addition to application checks. That combination is suitable only for this
fixture proof. A later public-origin worker needs a separately reviewed egress
architecture; removing the address denial is not an acceptable migration.

Validate the unit on the target host before use:

```bash
systemd-analyze verify deploy/observatory/twirx-observer-fixture.service
systemd-analyze security deploy/observatory/twirx-observer-fixture.service
```

Those commands validate a configuration. They do not prove that it is active.
