# Origin Access Policy — Genesis

Genesis supports public, read-only HTTP `GET` observations.

## Allowed

- `https` public origins on standard ports;
- `http` public origins where the user deliberately requests them;
- explicitly controlled loopback fixtures when `--allow-loopback` is supplied locally;
- bounded response bodies;
- bounded redirects, with every redirected destination revalidated;
- public JSON, JSON-LD, HTML, and other declared response representations preserved as evidence.

## Denied

- embedded URL credentials;
- schemes other than HTTP and HTTPS;
- private, loopback, link-local, multicast, reserved, documentation, and metadata-service network ranges under public policy;
- non-standard ports under public policy;
- unbounded bodies, decompression, redirects, or time;
- credentialed browsing;
- CAPTCHA or anti-bot circumvention;
- paywall bypass;
- browser automation in the Genesis slice;
- write actions.

## Important limitation

The Genesis safe-fetch package is an application-layer defense for local development. A public multi-tenant deployment additionally requires network-level egress isolation, separate worker credentials, controlled DNS, process or VM isolation, monitoring, and independent security review.

## E2 catalog-only public mode

The deployable Live Provenance Lab accepts `origin_id`, `operation_id`, and
contract-typed input and executes admitted offline replay only. `mode: fresh`
fails closed before quota or execution admission. User input cannot add or
override a URL, scheme, host, port, path template, redirect policy, or response
limit.

Each origin has an explicit request budget, deadline, response limit,
attribution, policy reference, terms reference, and recorded offline fixture.
The controlled fixture is produced locally under its explicit fixture policy.
Explicit local CLI fresh mode remains subject to application URL policy.
Public fresh mode stays disabled until it uses the separately admitted worker
and network-level egress controls. Application checks alone are not sufficient
for submitted or arbitrary origins.

## E3 local-fixture worker

The Observatory process-boundary proof accepts a committed job for
`http://127.0.0.1:<explicit-port>/robots.txt` only. It rejects every hostname,
other address, redirect, credential, query, and fragment. The response body
and canonical observation are stored before robots parsing, and verification
replays from those artifacts after the fixture stops.

This profile does not contact or authorize an Atlas-selected origin. The
included loopback-only systemd unit is an unactivated review candidate, not
evidence of production egress enforcement.
