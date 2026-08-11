# E3.1 TWIRX admission and local-fixture worker evidence

**Base commit:** `baa42cf81ad69837e605acb0b4b987154082a457`

**Recorded:** 2026-08-10

**Implementation status:** PASS for the founder-reviewed TWIRX A1/A2 record
and the process-separated local-fixture retrieval proof

**Gate status:** FAIL for complete E3.1 admission and FAIL for E3 admission;
499 identity/policy reviews, public-origin egress admission, and every later
maturity floor remain outstanding

## Outcome

The publisher-authored `https://twirx.org` origin is the first reviewed Atlas
record. Its evidence supports A1 identity/catalog review and A2 policy
assessment. The access outcome is deliberately `review_required`:

```text
selected candidates                 500
reviewed policy records               1
A1 or higher                          1
A2 or higher                          1
highest exactly A0                  499
highest exactly A2                    1
A3 through A9                         0
TWIRX access state       review_required
TWIRX scheduler                 disabled
dry-run jobs                           0
```

The A2 record binds the exact policy-set artifact:

```text
selection
sha256:209c3629ad58c60ea4477436332ec4847b3f0ecaf7b091046923c95c87f53fc5

policy set
sha256:856399a039cddcfa882555965da45b98c082077e5647182745d7bff166533a83
```

The live `https://twirx.org/robots.txt` representation remains `not_observed`.
No selected origin was contacted.

Separately, `twirx-observer-worker` proves the intended retrieval ordering
against a controlled loopback fixture only:

```text
validate and preserve exact job
retrieve literal 127.0.0.1 robots representation
write response body to CAS
write canonical observation and body reference
parse and evaluate robots
publish result
stop fixture
recompute digests and decision offline
```

The committed job digest is
`sha256:8616ae226268c6113ab50263f3542f4f81365952daec064509edc7e4e8221f56`.
The deterministic fixture body is 55 bytes with digest
`sha256:8dd624300420b840db6c1d508cfff98b8e04c48a42713c1fd5024e5d5feac39d`.
Its `/private/report` evaluation is denied by the `/private` rule. Observation
identity changes with the recorded retrieval time and is not represented as a
fixed corpus digest.

## Invariants implemented

1. **Selection does not grant access.** The TWIRX selection entry remains A0;
   maturity lives in the separate attested registry.
2. **Policy evidence is exact.** The A2 registry record binds the SHA-256 of
   the complete policy-set bytes and reproduces every reviewed access field.
3. **Review is not authorization.** `review_required`, zero budgets, disabled
   refresh, and disabled scheduling are mutually enforced. The frontier emits
   `access_not_allowed` and no job.
4. **Unknown robots state fails closed.** The live robots record carries no
   time or artifact digest and cannot support `allow`.
5. **No false jurisdiction coverage.** The explicit `not_established`
   jurisdiction does not increase the country/territory metric.
6. **Worker authority is finite.** The executable job profile accepts only
   plain HTTP `/robots.txt` on literal `127.0.0.1` with an explicit port, the
   `TWIRXBot` token, no redirect, and at most 500 KiB.
7. **Evidence precedes interpretation.** The CAS body and observation are
   committed before the robots parser runs. Invalid UTF-8 leaves evidence and
   produces no successful result.
8. **Offline replay is evidence-bound.** Verification checks the exact job,
   observation, CAS body, shared fields, digests, parser output, and decision
   without creating a network client.
9. **Untrusted output cannot promote itself.** The worker has no policy,
   registry, scheduler, adapter, semantic-canon, or publisher-verification
   write path.
10. **Protocol authority remains language-neutral.** The JSON schemas and
    specification define the fixture profile. The Genesis implementation is
    not normative protocol authority.
11. **Earlier gates remain intact.** E1 observation, E2 typed result, Go/C
    agreement, read-only effects, native-value preservation, and proof-bundle
    behavior pass unchanged.

## Commands executed

Artifact and focused validation:

```bash
sha256sum atlas/policies.json
go run ./cmd/twirx-atlas validate --root .
go run ./cmd/twirx-atlas metrics --root .
go run ./cmd/twirx-atlas plan --root . --at 2026-08-10T16:01:26Z
gofmt -w cmd/tw-test-origin/main.go internal/observatoryworker/*.go cmd/twirx-observer-worker/*.go
python3 -m json.tool atlas/policies.json >/dev/null
python3 -m json.tool atlas/registry.json >/dev/null
python3 -m json.tool schemas/json/observatory-job.schema.json >/dev/null
python3 -m json.tool schemas/json/observatory-result.schema.json >/dev/null
python3 -m json.tool conformance/observatory/v1/local-robots-job.json >/dev/null
go test ./internal/observatoryworker ./cmd/twirx-observer-worker ./cmd/tw-test-origin ./internal/atlas ./internal/atlasapi ./cmd/twirx-atlas
make build
make demo-e3-worker
make generate-e3
./scripts/check-docs.sh
bash -n scripts/demo-e3-worker.sh
git diff --check
```

Complete regression and evidence matrix:

```bash
go test -race ./...
go vet ./...
make test
make demo
make demo-e2
make demo-e3
make demo-e3-worker
go test -json ./... | rg -c '"Action":"pass".*"Test":'
go version
gcc --version | head -1
clang --version | head -1
systemd-analyze verify deploy/observatory/twirx-observer-fixture.service
systemd-analyze security --offline=yes deploy/observatory/twirx-observer-fixture.service
```

## Results

| Check | Result |
|---|---:|
| Named Go test pass events | 246 |
| Go race run | PASS |
| Go vet | PASS |
| Go fuzz targets | 12 PASS |
| New worker-job fuzz target | PASS |
| Evidence-before-malformed-parse test | PASS |
| Non-loopback, credential, query, redirect, duplicate, trailing, unknown, and symlink tests | PASS |
| C observation vectors | 2 valid accepted; 14 invalid rejected |
| E2 shared Go/C vectors | PASS |
| C observation libFuzzer | 5,000 runs, no crash |
| C E2 libFuzzer | 5,000 runs, no crash |
| Clang ASan/UBSan builds and tests | PASS |
| E1 end-to-end demonstration | PASS |
| E2 deterministic agent transcript | PASS |
| E3 policy/frontier demonstration | PASS; zero jobs |
| E3 fixture-worker stopped-origin replay | PASS |
| Documentation navigation and JSON syntax checks | PASS |
| Offline systemd security analysis | Exposure 1.1, `OK` |
| Target-host systemd verification | NOT RUN; installed release path is absent locally |

Toolchains observed:

```text
Go go1.26.5-X:nodwarf5 linux/amd64
GCC 16.1.1 20260625
Clang 22.1.8
```

`systemd-analyze verify` parsed the unit but correctly reported that
`/srv/twirx/current/bin/twirx-observer-worker` does not exist on this local
development host. The service is an unactivated deployment candidate; this is
recorded as a limitation rather than a deployment claim.

## Files changed

- Admission decision and evidence: `decisions/005-twirx-origin-admission.md`,
  `atlas/policies.json`, `atlas/registry.json`, and generated E3 metrics.
- Atlas honesty and public view: maturity/coverage tests, the read-only API's
  explicit `access_state`, and current-status documentation.
- Worker implementation: `internal/observatoryworker` and
  `cmd/twirx-observer-worker`.
- Fixture and demonstration: the controlled origin's `/robots.txt` route,
  `conformance/observatory/v1/local-robots-job.json`, the Makefile, and
  `scripts/demo-e3-worker.sh`.
- Language-neutral contracts: Observatory job/result JSON schemas and
  `spec/atlas/OBSERVATORY_WORKER_0_1.md`.
- Unactivated operations candidate: `deploy/observatory`.
- Architecture, access, security, task, README, and Mintlify documentation.

## Unresolved risks and exclusions

1. The other 499 selected origins have no A1 or A2 evidence.
2. TWIRX terms and risk review remain incomplete; its robots representation is
   not observed and network access is not allowed.
3. The worker supports only a literal-loopback fixture. It has no admitted
   public DNS, redirect, conditional-request, caching, sitemap, feed, or
   continuous-scheduler path.
4. The systemd unit is not installed or active. Network namespace behavior,
   filesystem ownership, logging, restart, quota, and revocation behavior need
   target-host tests.
5. The worker relies on application checks in local development. Production
   still needs network-level egress enforcement and controlled DNS.
6. The worker result has one implementation and no independent C verifier.
7. The operator-provided output parent is trusted. A hostile local operator or
   compromised host remains outside the fixture proof.
8. No selected origin is A3-profiled or A4-observed. No adapter, semantic
   mapping, live operation, publisher verification, or model corpus is added.
9. E3.1 and E3 acceptance floors remain unmet.

## Deviations

No external-origin request, VPS change, DNS change, deployment, repository
visibility change, runtime dependency, production/test refactor, browser,
model, write action, or E1/E2 behavioral change was introduced. The fixture
proof intentionally stops before live TWIRX robots retrieval. Untracked Claude
packs, website work, migration reports, and unrelated VPS repositories were
not modified or included.

## Recommendation

**PASS this bounded implementation slice. Do not pass E3.1 or activate
public-origin egress.** The next engineering gate should install and test the
worker boundary in a disposable staging environment using only adversarial
local fixtures, including redirect, DNS/private-range, resource, lifecycle,
quota, logging, and revocation tests. A later maintainer decision may change
TWIRX from `review_required` to `allow` only after that boundary and the exact
live robots/terms/risk evidence pass review.
