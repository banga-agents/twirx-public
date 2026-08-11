# Bounded runtime stress evidence

The E2 stress workload exercises all five admitted read operations in offline
replay mode. It does not contact an origin, accept a destination URL, weaken a
catalog policy, or make a production-capacity claim.

Run:

```bash
make stress-e2
```

The harness starts the Lab on literal loopback, sends 100 mixed invocations
from 100 explicitly simulated clients at the admitted concurrency ceiling of
8, validates every
typed result, downloads and rehashes every distinct proof bundle, and then
verifies the published directories with both the primary and independent
restricted verifier. A second fresh process confirms that the configured
per-client burst rejects overload with HTTP 429.

The workload weights stay within the catalog's initial per-origin bursts:

```text
controlled fixture  60
TWIRX project        30 across three operations
World Bank replay    10
```

Simulated clients use documentation-only IP addresses in `X-Forwarded-For`.
The harness can set that header only when its target is a literal loopback
address, matching the service's Caddy trust boundary. Its client refuses every
target except literal loopback and `https://lab.twirx.org`.

The separate overload phase sends concurrency 8 beyond the per-client burst;
requests rejected by admission are not counted as execution capacity.

This answers a narrow question: whether the admitted deterministic runtime
survives concurrent use while preserving its result and proof invariants. It
does not measure public TLS, fresh-origin reliability, continuous ingestion,
or the usefulness of 500 compiled origins.
