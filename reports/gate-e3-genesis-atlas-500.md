# Engineering Gate E3 — Genesis Atlas 500 evidence

**Implementation status:** E3.0 offline control-plane candidate passed locally

**Gate admission:** FAIL; E3.1–E3.5 and E2 production admission remain outstanding

**Release label:** Genesis Preview

**Evidence date:** 2026-08-10

**Implementation commit tested:** `14faeacc947a583d858a67b8ec2cec8a56406e8c`

**Stacked E2 baseline:** `7ef731ce7d8b8c865f0aedf634fc407e93a5693c`

This report is committed after the implementation it describes, so its own
commit identifier is intentionally not the tested implementation identifier.

## Outcome

E3.0 now has a real, bounded control plane:

- accepted ADR 004 and five independently testable E3 subgates;
- a read-only measurement of the actual shared VPS;
- one exact, digest-bound selection of 500 unique HTTPS A0 candidates under
  the required 100/80/60/60/50/50/50/30/20 family quotas;
- a separate empty A1–A9 registry rather than automatic candidate promotion;
- language-neutral JSON schemas and a normative control-plane specification;
- sequential maturity attestations that reject skipped levels;
- deterministic exact-highest and at-or-above metrics;
- a loopback-only GET API for bounded status, list, filter, pagination, and
  origin description;
- no HTTP client, crawler, browser, model, adapter execution, or candidate
  network access in the E3.0 process.

Generated evidence truthfully reports:

```text
500 A0 candidates
0 A1 cataloged
0 A2 policy assessed
0 A3 profiled
0 A4 observed
0 A5 native schema
0 A6 compiled
0 A7 semantically linked
0 A8 live
0 A9 publisher verified
WSIM Seed ready: false
```

The selection digest is
`sha256:209c3629ad58c60ea4477436332ec4847b3f0ecaf7b091046923c95c87f53fc5`.
TWIRX itself is included in the 500 but remains A0 under the same rules as
every other candidate.

## E2 prerequisite result

The Genesis steward accepted ADR 003's acyclic publication graph. Commit
`88431ffcaf3d3f37a85eaed3dddbf7129ba00b27` makes `bundle_id` the detached
SHA-256 digest of the exact canonical manifest and adds shared Go/C rejection
cases for missing, cyclic, substituted, malformed, symlinked, and trailing
artifacts. The complete E1+E2 suite passed after that change.

E2 is still not deployed or admitted. The VPS inspection found a valid UFW
default-deny boundary and Caddy configuration, but also a busy shared host,
nearly full swap, many unrelated listeners, no dedicated Atlas/Lab service
limits, and insufficient egress isolation. No deployment was attempted.

## Acceptance-floor accounting

| Evidence | Required for E3 | Current | Result |
|---|---:|---:|---|
| Exact selected candidates | 500 | 500 A0 | PASS for E3.0 only |
| A1 cataloged | 500 | 0 | FAIL |
| A2 policy assessed | 500 | 0 | FAIL |
| A3 profiled | 300 | 0 | FAIL |
| A4 observed | 100 | 0 | FAIL |
| A5 native schemas | 50 | 0 | FAIL |
| A6 compiled adapters | 25 | 0 | FAIL |
| A7 cross-origin semantic operation families | 12 | 0 | FAIL |
| A8 live origins | 8 | 0 | FAIL |
| A9 publisher-authored or verified | 1 | 0 | FAIL |
| Daily bounded scheduler | operating | disabled | FAIL |
| Semantic discovery and invocation | complete | not implemented | FAIL |
| Controlled browser comparisons | 3 | 0 | FAIL |

## Main validation command

The following complete offline sequence passed from a clean generated-runtime
state at the tested implementation commit:

```bash
make clean && make build && make test && make demo && make demo-e2 && make demo-e3
```

It ran all E1 and E2 validation plus E3 selection/registry parser tests and two
new Go fuzz targets.

## Supplemental validation

```bash
go test -count=1 -json ./... > /tmp/twirx-e3-go-tests.json
jq -n --slurpfile tests /tmp/twirx-e3-go-tests.json \
  '{named_passes:([$tests[]|select(.Action=="pass" and .Test!=null)]|length),
    named_failures:([$tests[]|select(.Action=="fail" and .Test!=null)]|length),
    named_skips:([$tests[]|select(.Action=="skip" and .Test!=null)]|length),
    package_failures:([$tests[]|select(.Action=="fail" and .Test==null)]|length)}'
go test -race ./...
go test -cover ./...
go vet ./...
test -z "$(gofmt -l cmd internal)"

make generate-e2
git diff --exit-code -- generated/e2
make generate-e3
git diff --exit-code -- generated/e3
./scripts/check-docs.sh
jq empty schemas/json/*.json atlas/genesis-500/selection.json \
  atlas/registry.json generated/e3/atlas-metrics.json
shellcheck scripts/*.sh
node --check lab/static/app.js
git diff --check
```

All passed. The uncached JSON test stream contained 192 named passes, zero
named failures, zero named skips, and zero package failures. The race detector
passed. Measured statement coverage was 67.9% for `internal/atlas`, 69.5% for
`internal/atlasapi`, and 52.3% for `cmd/twirx-atlas`.

The main suite ran nine Go fuzz targets for one second each, 5,000 E1 C
libFuzzer executions, 5,000 E2 C libFuzzer executions, GCC and Clang builds,
ASan, UBSan, 16 shared E1 observation vectors, and 13 shared E2 artifact/bundle
vectors. Supplemental production-warning and static-analyzer commands also
passed:

```bash
clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-e3-final-clang verifier/c/e2_main.c \
  verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c
gcc -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-e3-final-gcc verifier/c/e2_main.c \
  verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c
for source in verifier/c/e2_main.c verifier/c/e2_artifact_main.c \
  verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c; do
  output=/tmp/e3-$(basename "$source").plist
  clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion \
    -Wshadow -Wpedantic -o "$output" "$source"
done
```

## Local HTTP evidence

`bin/twirx-atlas serve --root . --listen 127.0.0.1:18092` was exercised with
`curl` and `jq`. Status returned 500 candidates and zero cataloged origins; a
government-family page returned 10 of 100 A0 records; the TWIRX detail remained
A0 with a null publisher; and POST returned HTTP 405. The server was then
stopped. The command rejects non-loopback listeners in both code and tests.

## Secret scans

```bash
gitleaks git . --redact --report-format json \
  --report-path /tmp/twirx-e3-gitleaks-history.json
gitleaks dir . --redact --report-format json \
  --report-path /tmp/twirx-e3-gitleaks-tree.json
GIT_ALLOW_PROTOCOL=file trufflehog git \
  file:///home/shiva/typed-web-genesis \
  --branch agent/e3-genesis-atlas-500 \
  --no-verification --no-update --json \
  --results=verified,unknown,unverified
```

Gitleaks 8.30.1 found zero items in 26 reachable commits and zero in the full
working tree, including preserved untracked website/planning work. TruffleHog
3.96.0 found zero verified items and the same one unverified URI heuristic in
the intentional embedded-credential rejection fixture at
`internal/safefetch/safefetch_test.go:52`.

## VPS evidence

`reports/e3-capacity-baseline.md` records the exact read-only SSH commands,
measured CPU, memory, swap, disk, interfaces, firewall, Caddy validation, and
service-limit state. No VPS file, service, firewall rule, DNS record,
repository, or deployment changed. The unrelated `meridian-velo` repository
was not accessed.

## Invariants implemented

- Candidate breadth cannot inflate maturity depth.
- Selection identity is unique, exact-origin HTTPS, credential-free, and
  quota-bound.
- A registry origin must be selected and preserve its selected identity.
- Maturity is sequential and evidence-attested; A9 cannot bypass A8.
- Metrics are deterministic and generated from validated repository artifacts.
- WSIM readiness fails closed below every corpus threshold.
- The E3.0 process has no outbound-network or shell-execution capability.
- The HTTP API is bounded, GET-only, loopback-only, and has no URL input.
- E1/E2 evidence, provenance, native-first mapping, read-only behavior, and
  independent C verification remain intact.

## Files changed relative to the stacked E2 branch

- Atlas selection/registry: `atlas/`;
- control-plane implementation: `cmd/twirx-atlas`, `internal/atlas`,
  `internal/atlasapi`;
- schemas/specification: `schemas/json`, `spec/atlas`;
- generated evidence: `generated/e3/atlas-metrics.json`;
- decisions/tasks/reports: ADR 004, Task 003, capacity and selection reports;
- architecture, threat model, security policy, README, Makefile, demo, and
  Mintlify documentation.

The copied implementation packs and Claude's untracked `web/` work were not
staged or modified.

## Unresolved risks and limitations

1. Every selected external origin is A0 and unreviewed. Some may be obsolete,
   redirected, misclassified, controlled by an unexpected publisher, or
   inappropriate for later access.
2. No publisher, jurisdiction, language, authority, robots, terms,
   attribution, authentication, rate, retention, or risk review exists yet.
3. The overlapping diversity and representation-surface targets have zero
   evidence-backed coverage; selection-family quotas alone do not prove them.
4. No E3 crawler, robots parser, sitemap/feed/XML parser, archive importer,
   scheduler, evidence store, native schema compiler, adapter, semantic graph,
   Atlas agent, publisher kit, learning ledger, or model is admitted.
5. E2 lacks the dedicated production egress boundary and public deployment
   evidence on which live E3 work depends.
6. The VPS is shared and had almost fully used swap at measurement time.
   Provider bandwidth, IOPS, backups, and failure-domain guarantees remain
   unmeasured.
7. The control plane has one implementation. The JSON schemas and fixtures are
   language-neutral, but independent Atlas validation remains future work.
8. Metrics currently contain null runtime measurements rather than invented
   production statistics.
9. Hosted GitHub Actions still has no executed runner evidence; local evidence
   is not described as hosted CI.
10. Repository public-readiness still has the immutable private-PR metadata
    limitation recorded in `reports/public-readiness.md`.

## Deviations from the complete E3 work order

- Only the required E3.0 prerequisite/control-plane subgate was implemented.
  Completing later floors would require 500 real human policy decisions and a
  production egress boundary; fabricating either would violate the gate.
- No candidate was contacted. Continuous ingestion did not start because its
  control-plane, policy, egress, and founder-review prerequisites do not yet
  pass together.
- No deployment or merge occurred, as instructed.
- No WSIM model was trained because all four minimum corpus counts are zero.

## Recommendation and next gate

**PASS E3.0 as an offline control-plane implementation candidate. FAIL E3
admission and any Atlas deployment.**

The next recommended action is founder review of the stacked E2/E3 draft. On
approval, complete the dedicated E2 service/egress boundary and public Lab
evidence, then begin E3.1 with identity review, policy records, a dry-run
per-origin frontier, and RFC 9309 conformance. Do not fetch any of the 500 from
the A0 selection.
