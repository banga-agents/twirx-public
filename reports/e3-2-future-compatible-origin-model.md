# E3.2 future-compatible origin model evidence

Date: 2026-08-11  
Implementation commit under test: `7ca65ad97af9360df7b45c9aa64b8bb73423ab02`  
Base: admitted E3.1 merge `6db9e0a7b925b0c2e6e1bff81dff5acb246b3335`  
Recommendation: **PASS for founder review of the compatibility amendment; E3.2 remains unadmitted and undeployed.**

This report supplements, and does not replace or edit, the exact evidence in
`reports/e3-2-admission-egress-pilot.md`. It records the application of
`E3_2_FUTURE_COMPATIBILITY_ADDENDUM.md`. The report commit itself changes only
documentation; the implementation commit above is the exact executable tree
used for the measurements below.

## Outcome

The Atlas origin model now records interface declarations, capability
candidates, effect classes, access and economic metadata, infrastructure-cost
estimates and evidence-bound publisher-readiness signals. These fields are
descriptive unless independently admitted. They create no new retrieval or
execution authority.

The following invariants passed:

- non-unknown declarations are bound to immutable evidence digests;
- unknown or unassessed state cannot claim evidence, an offer or a price;
- only an existing compiled, semantically linked, public-read E2 operation can
  be represented as admitted;
- an admitted interface must refer exactly to an existing E2 execution-catalog
  entry; WebMCP, MCP, browser, payment, authenticated, write and action
  capability execution remains forbidden by E3.2;
- access class, offer state, technical stage, policy, health, publisher status,
  adapter trust and mapping trust remain orthogonal dimensions;
- readiness evidence distinguishes direct origin statements, discovered
  machine declarations, inferred signals and project-generated declarations;
- machine-readable payment requirements remain declaration evidence, never
  payment authority;
- natural-language or commercial metadata cannot promote canonical state;
- the scheduler and origin budgets remain disabled and the frontier remains at
  zero jobs;
- the local Observatory proof remains literal-loopback-only and publishes
  immutable evidence before parsing;
- the 500-origin selection is unchanged and controlled fixtures remain excluded
  from public-origin counters.

## Canonical E2 reconciliation

The Atlas imports qualifying E2 evidence without duplicating execution state:

- `twirx-org`: REST and TWIRX-native interfaces plus three public-read
  operations; MCP and OpenAPI are descriptive declarations only;
- `api-worldbank-org`: one admitted REST interface and one public-read
  operation;
- `controlled-origin-lab-fixture`: imported as `test_fixture`, never counted as
  a Genesis public origin.

The canonical public metrics therefore expose four admitted public-read
operations across two real origins. They do not claim an E3 live-origin fetch.

## Pilot evidence

The deterministic admission batch contains:

- 25 origin dossiers across all nine domain families;
- 11 countries or territories and six languages;
- 23 pending catalog reviews and two imported canonical E2 admissions;
- zero completed Atlas policy decisions;
- two origins at profiled/observed-or-beyond from existing E2 evidence;
- five commercial candidates: `latimes-com`, `lemonde-fr`, `nytimes-com`,
  `reuters-com` and `washingtonpost-com`;
- five provisional `subscription` access classifications and five unverified
  offer candidates;
- zero machine-readable payment declarations;
- zero new requests, transferred bytes, retained evidence bytes, worker CPU,
  review time or infrastructure charges.

The five commercial notes are maintainer-directed catalog-review proposals.
They do not claim a current price, contractual term, machine payment protocol,
completed policy review or executable capability. Their policy state remains
`pending` + `uncertain` until a human decision is recorded.

The Atlas-500 runtime remains bounded at 500 selected origins, 25 prepared
dossiers, 475 unprepared origins, 500 frontier decisions and zero frontier
jobs.

## Commands executed

```bash
make clean
make build
make test
make demo
make demo-e2
make demo-e3
make demo-e3-worker
make stress-e2
make stress-e3-500

go test -race ./...
go vet ./...
go test -coverprofile=/tmp/twirx-e3-2-future-cover.out ./...
go tool cover -func=/tmp/twirx-e3-2-future-cover.out
go test -json ./...

go run ./cmd/twirx-admission validate \
  --root . --admissions atlas/admissions --fixtures atlas/fixtures
go run ./cmd/twirx-admission review-queue \
  --root . --admissions atlas/admissions --fixtures atlas/fixtures
go run ./cmd/twirx-admission check-canonical \
  --root . --admissions atlas/admissions --fixtures atlas/fixtures

go test -run='^$' \
  -bench='BenchmarkAdmissionLoadAndRender25|BenchmarkVerifyEvidenceSpool' \
  -benchtime=2s -count=1 -benchmem \
  ./internal/admission ./internal/egressworker

make generate-e2 generate-e3 generate-e3-admission
rg --files generated/e3/admission | sort | xargs sha256sum | sha256sum
make generate-e2 generate-e3 generate-e3-admission
rg --files generated/e3/admission | sort | xargs sha256sum | sha256sum
git diff --exit-code -- generated/e2 generated/e3 atlas/registry.json

systemd-analyze verify deploy/egress/twirx-egress-worker@.service
systemd-analyze security --offline=yes \
  deploy/egress/twirx-egress-worker@.service
shellcheck deploy/egress/verify-target.sh
bash -n deploy/egress/verify-target.sh
./scripts/check-docs.sh
```

GCC and Clang each built the observation, E2 result and E2 artifact C
verifiers with:

```text
-std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic
```

No public origin was contacted by these commands.

## Results

- 304 named Go test pass events;
- 14 Go fuzz targets passed;
- Go race detector and vet passed;
- aggregate Go statement coverage: 66.7%;
- strict GCC and Clang builds passed;
- Clang ASan and UBSan passed;
- observation C vectors accepted two valid cases, rejected 14 invalid cases
  and rejected corruption;
- shared E2 Go/C conformance passed;
- both C libFuzzer targets completed 5,000 runs without a crash;
- E1, E2, E3.1 and literal-loopback worker demonstrations passed;
- E2 stress: 100 deterministic invocations and five verified bundles; HTTP
  load yielded 20 successes and 30 expected rate-limit responses at concurrency
  eight, 17.667 ms mean latency, 105.892 ms p95 and 17,772 KiB peak RSS;
- Atlas stress: 50,000 requests at 32 workers, 8,183.653 requests/second,
  10,934 microseconds p95 and 23,732 KiB peak RSS; zero frontier jobs;
- admission render benchmark: 16.989 ms/op, 2,557,005 B/op and 57,699
  allocations/op;
- immutable evidence-spool verification: 405.914 microseconds/op, 59,678 B/op
  and 953 allocations/op;
- generated admission artifacts reproduced byte-for-byte twice with aggregate
  SHA-256 `cbe20286abb231b520d8d1c976b8d717d86de323a0d45e69cf0687502681a1d2`;
- systemd offline security exposure: `1.2 OK`;
- docs, JSON, shell syntax and deterministic generation checks passed.

`systemd-analyze verify` emitted the expected development-host warning that
`/srv/twirx/current/bin/twirx-egress-worker` is not installed. No unit was
installed or activated to suppress it.

## Files changed

The implementation commit changes 94 tracked files: the Atlas state and
validation packages; API and admission derivation; the registry JSON Schema;
25 admission records and their generated dossiers; E2 registry reconciliation;
tests; generated metrics; and the corresponding architecture, threat-model,
protocol and task documentation. It adds no dependency and changes no E1 or E2
execution behavior.

## Unresolved risks

1. Twenty-three catalog reviews and all 25 explicit Atlas policy decisions
   still require human action. The five commercial candidates are not completed
   policy reviews.
2. The E3.2 acceptance floors for policy outcomes, ten safe profiles and five
   new immutable observations are not met.
3. The target-host DNS and network boundary is still an unactivated deployment
   candidate. The host's current private MagicDNS resolvers fail closed under
   the candidate policy.
4. The worker has not produced target-host CPU, RSS, byte or retained-evidence
   measurements because no live retrieval was authorized.
5. Capability presence is not semantic equivalence, rank, publisher approval,
   freshness or execution authority.
6. Access and economic observations can drift and must retain time, freshness,
   source and evidence class.
7. Host/root compromise, distributed abuse state, external release
   transparency and a failure-manifest design remain future controls.

## Deviations

- No live commercial site was fetched. The five requested commercial
  candidates were added as explicit, evidence-bound review proposals so the
  project does not invent current access terms.
- No E3.3 semantic resolver, ranking, compact meta-interface or execution path
  was implemented in this amendment.
- No repository, DNS, VPS, service, firewall or deployment state was changed.

## Next recommended gate

Founder review should first admit or reject the 23 pending catalog dossiers,
authorize exact evidence work orders, complete the 25 policy decisions and
approve a target DNS-isolation design. Only then should the bounded E3.2 live
pilot run. E3.3 may prepare language-neutral request, plan, capability and offer
contracts separately, but semantic execution must remain blocked until E3.2 is
admitted.
