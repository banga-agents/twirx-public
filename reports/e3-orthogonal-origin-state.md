# E3 orthogonal origin-state correction evidence

**Implementation commit:** `3a9f04992d44a83a00e9a519c8a9f114d0f47ba1`

**Tested:** 2026-08-11

**Recommendation:** PASS for founder review of the corrective E3 state model;
E3 remains unadmitted and public-origin retrieval remains disabled

## Root cause

The merged E3.1 model represented catalog review, policy assessment,
technical progress, publisher relationship and live status as a single A0–A9
sequence. That sequence encoded false implications between independent facts.
It also counted a `review_required` access value as a completed policy
assessment and left E2 execution identities outside the Atlas registry.

PR #8 had already merged before the correction request. This work therefore
corrects the merged tip forward rather than amending or rewriting published
history.

## Corrective result

The active model now derives and exposes these dimensions independently:

- `catalog.state`;
- `policy.review_state` and `policy.decision`;
- `technical.stage`;
- `publisher.status`;
- `health.status`;
- `adapter_trust.status`;
- `mapping_trust.status`.

Pending policy is fail-closed. It uses `decision: uncertain` but never counts
as a completed uncertain decision. A completed uncertain decision requires an
explicit reviewer and canonical review time.

`atlas/registry.json` is the canonical origin identity and state registry.
E2 execution entries bind a `registry_id`; E2 startup rejects host, publisher,
scope or alias disagreement. Exact repository evidence paths and SHA-256
digests are revalidated when the registry loads.

The qualifying E2 real origins are imported as `twirx-org` and
`api-worldbank-org`. Their independent technical evidence includes replay
representations, contracts, conformance vectors, generated proof metadata and
semantic-closure digests. Their Atlas policies remain pending and uncertain.

The controlled E2 origin has scope `test_fixture`. It is excluded from every
Genesis-500 public-origin state, coverage and semantic counter.

## Current derived state

```text
selected candidates                         500
cataloged public origins                       2
candidate public origins                     498
completed policy reviews                       0
pending policy reviews                       500
semantically linked technical records          2
publisher-approved public origins               1
healthy or degraded public origins              0
live public origins                             0
conformant public adapter records                2
reviewed public mapping records                  2
excluded test fixtures                          1
request budgets                                 0
storage budgets                                 0
frontier jobs                                   0
```

The generated metrics are
`generated/e3/atlas-metrics.json` (`tw.atlas-metrics/0.2`).

## PR #4 / PR #8 ancestry

PR #4 merged before PR #8. The exact admitted E2 head
`7ef731ce7d8b8c865f0aedf634fc407e93a5693c` is an ancestor of the PR #8 merge
`ef7c74bcfe3f673b77d14e21047139e1a26217c7` and of this correction. The PR #4
merge commit itself is not an ancestor because the later stack was based on
the admitted E2 head before GitHub created PR #4's merge commit. The E2
implementation is present once; no commit was duplicated, rewritten or
force-pushed.

## Complete commands executed

Primary clean acceptance sequence:

```bash
make clean && make build && make test && make demo && make demo-e2 && \
  make demo-e3 && make demo-e3-worker
```

Supplemental Go, generator and repository checks:

```bash
go test -race ./...
go vet ./...
go test -cover ./...
go test -count=1 -json ./... > /tmp/twirx-e3-state-go-test.json
rg -c '"Action":"pass".*"Test":' /tmp/twirx-e3-state-go-test.json

make generate-e2 generate-e3
gofmt -l cmd internal
bash -n scripts/*.sh
shellcheck scripts/*.sh
node --check lab/static/app.js
./scripts/check-docs.sh
git diff --check
jq empty atlas/*.json atlas/genesis-500/selection.json \
  schemas/json/*.json origins/catalog.json \
  generated/e3/atlas-metrics.json
```

The generated E2 and E3 trees were hashed before and after generation; both
were byte-identical.

Independent C production and static-analysis checks:

```bash
gcc -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/twirx-e3-state-gcc \
  verifier/c/e2_main.c verifier/c/e2.c \
  verifier/c/observation.c verifier/c/sha256.c

clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/twirx-e3-state-clang \
  verifier/c/e2_main.c verifier/c/e2.c \
  verifier/c/observation.c verifier/c/sha256.c

for source in verifier/c/e2_main.c verifier/c/e2_artifact_main.c \
  verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c; do
  output=/tmp/twirx-e3-state-$(basename "$source").plist
  clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion \
    -Wshadow -Wpedantic -o "$output" "$source"
done
```

Deployment-candidate configuration checks:

```bash
systemd-analyze verify deploy/observatory/twirx-observer-fixture.service
systemd-analyze security --offline=yes \
  deploy/observatory/twirx-observer-fixture.service
```

## Results

| Check | Result |
|---|---:|
| Named Go test pass events | 248 |
| Go race detector | PASS |
| Go vet | PASS |
| Go fuzz targets | 12 PASS |
| C observation vectors | 2 valid accepted; 14 invalid rejected |
| E2 shared Go/C vectors | PASS |
| C observation libFuzzer | 5,000 runs, no crash |
| C E2 libFuzzer | 5,000 runs, no crash |
| Clang ASan/UBSan | PASS |
| GCC and Clang production builds | PASS |
| Clang static analyzer | PASS |
| E1 demonstration | PASS |
| E2 deterministic agent transcript | PASS; result digest unchanged |
| E3 state/frontier demonstration | PASS; zero jobs |
| Literal-loopback worker and stopped-origin replay | PASS |
| Generator reproducibility | PASS |
| Docs, shell, JavaScript, JSON and diff checks | PASS |
| Offline systemd security analysis | Exposure 1.1, `OK` |
| Target release-path verification | NOT RUN on target; local path absent |

Toolchains:

```text
Go go1.26.5-X:nodwarf5 linux/amd64
GCC 16.1.1 20260625
Clang 22.1.8
```

## Files changed

- State contracts and decisions: ADR 006, Atlas selection/policy/registry JSON,
  all three Atlas JSON schemas, and the language-neutral control-plane spec.
- Implementation: Atlas state validation, metrics, frontier, read-only API,
  canonical E2 registry binding, and exact evidence-file verification.
- Conformance: orthogonal-state, pending-policy, fixture exclusion, alias,
  policy binding, API filter, malformed input and fuzz tests.
- Generated evidence and documentation: E3 metrics, architecture, security,
  threat model, status, roadmap, origin-catalog and task documentation.

No workflow, runtime dependency, C verifier behavior, adapter extraction,
production deployment, VPS service or repository visibility changed.

## Preserved invariants

- E1 and E2 remain read-only.
- Native source terms and lexical values remain preserved before mapping.
- Missing evidence fails closed; optional missing values remain unresolved.
- The C verifier and extraction path retain no network access.
- The Atlas control plane has no HTTP client.
- The scheduler and every committed request/storage budget remain disabled or
  zero, and the plan contains zero jobs.
- Literal-loopback fixture policy, evidence-before-parsing and stopped-origin
  replay continue to pass.
- No user-supplied destination, browser, model, write action, payment action or
  automatic canon promotion was added.

## E1/E2 behavior confirmation

E1 and E2 operation contracts, result encoding, proof-bundle topology,
adapters, extraction, Go/C verification, generated bindings and deterministic
E2 transcript are unchanged. `make generate-e2` produced no diff, and the E2
controlled result remains
`sha256:8bafb410dee23a3e6a5011f81b46531083d82c9395e0b2e9d134b702433ee972`.

The only E2 path change is a startup-time fail-closed identity/evidence check:
an execution catalog whose canonical registry binding or evidence digest is
invalid can no longer start. Successful operation behavior did not change.

## Unresolved risks and limitations

1. The other 498 selected origins are not cataloged.
2. None of the 500 public-origin policy reviews is completed.
3. No public-origin egress boundary, health monitor or live Atlas scheduler is
   admitted.
4. Historical E3.0/E3.1 reports describe the state at their tested commits;
   ADR 006 and this report supersede their linear interpretation for current
   code without rewriting historical evidence.
5. The systemd unit parses and scores 1.1/OK offline, but target-host
   verification remains outstanding because the production release path does
   not exist on this development host. No deployment was authorized.
6. A report cannot contain the SHA of the commit that contains itself. The
   implementation SHA above is the exact code/docs/artifact tree subjected to
   the complete matrix; the following report-only commit must be rechecked
   before publication.

## Deviations

PR #8 could not be amended because it had already merged. The correction is a
non-rewriting follow-up from the exact merged tip. No other requested behavior
or safety constraint was relaxed.

## Next recommended gate

Founder review and merge of this correction, followed by the stacked draft
`E3.2 — Atlas Admission Factory and Secure Egress Pilot`. That subgate must not
perform broad 500-origin retrieval or deploy before founder review.
