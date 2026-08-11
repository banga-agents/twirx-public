# E3.2 Admission Factory and Secure Egress Pilot

Status: **IMPLEMENTATION PASS / GATE NOT ADMITTED**

Implementation commit: `90fefd7cca3ceea03725845d091b6e3410691955`

Evidence date: 2026-08-11

## Recommendation

Leave E3.2 as a stacked draft for founder review. The admission factory,
sealed worker, adversarial suite, deployment candidate, and 25-origin dossier
batch are implemented and locally validated. Do not merge, deploy, enable the
worker, or claim the first-origin pilot floors yet.

The remaining work requires human policy decisions and target-host authority,
not an implementation shortcut. No agent-prepared proposal was represented as
a human approval.

## Base and stack topology

- branch base: `e4dec2d011ce5e780889a8132025c27a24101304`;
- base branch: `agent/e3-orthogonal-origin-state` (PR #9);
- PR #8 is already merged at `ef7c74bcfe3f673b77d14e21047139e1a26217c7`;
- GitHub did not show PR #9 merged into `main` at evidence time;
- E3.2 therefore remains stacked on the exact orthogonal-state correction and
  does not duplicate those commits onto `main`.

## Implemented invariants

### Admission factory

- one public-origin directory per dossier;
- strict bounded JSON and regular-file validation;
- candidate identity and selected-family preservation;
- duplicate canonical identity, origin ID and alias rejection;
- local policy evidence rehashed before use;
- explicit pending versus completed admission review;
- only `reviewer_type: human` plus an approval reference can render canonical
  state;
- catalog admission remains orthogonal to policy review;
- pending policy remains `pending + uncertain` and never counts completed;
- deterministic policy set, registry, batch, dossier and review-queue output;
- no automatic decision, approval, canon promotion or network access;
- controlled fixtures loaded from a separate source directory and excluded
  from every Genesis-500 public-origin counter.

The two qualifying E2 public origins and the controlled E2 fixture are
reconciled into the single canonical vocabulary. Their technical, adapter,
mapping and artifact evidence is preserved. The policy-set digest changed
because the policies are now derived from per-origin digest-bound evidence;
the E2 deterministic result digest remains
`sha256:8bafb410dee23a3e6a5011f81b46531083d82c9395e0b2e9d134b702433ee972`.

### Secure egress boundary

- CLI accepts a sealed work-order ID and no URL;
- exact HTTPS route, host allowlist, authority class, policy/decision digests,
  human approval reference and validity interval are bound in the order;
- `policy_evidence_collection` is limited to a human-admitted origin's exact
  `/robots.txt` route under pending policy;
- broader retrieval requires a completed retrieval-permitting policy;
- `profile_only` cannot authorize observation;
- DNS is resolved and every answer is validated before each connection;
- a mixed public/private DNS answer fails entirely;
- redirects repeat URL, host, DNS, address and port validation;
- IPv4/IPv6 private, loopback, link-local, metadata, multicast, CGNAT,
  documentation, benchmarking, discard-only and reserved ranges are denied;
- embedded credentials, alternate numeric addresses, non-HTTP schemes,
  unexpected ports, redirect loops, TLS failures and oversized decompressed
  bodies fail closed;
- work order and representation evidence are written before any parsing;
- the worker contains no parser, adapter, registry/database/deployment/secret
  access or canonical write capability;
- the final manifest is written last and offline verification rehashes all six
  constituent artifacts;
- symlinked roots and artifacts, spool reuse and substituted work orders fail;
- global disable, emergency stop, order/origin revocation, one-worker lease,
  failure circuit and cooldown are enforced.

### Deployment candidate

The uninstalled systemd candidate uses a dedicated unprivileged account,
strict read/write paths, hidden repository/control-plane/secret paths,
cgroup-scoped denied IPv4/IPv6 ranges, no socket binding, a 256 MiB memory
ceiling, no swap, 50% CPU quota, 32-task ceiling, 64-file ceiling and 45-second
start timeout. Its committed control artifact is disabled with emergency stop
active. It has no install target.

Local `systemd-analyze security --offline=yes` reports exposure `1.2 OK`.

## Pilot evidence

| Requirement | Evidence | Result |
| --- | --- | --- |
| 25 Genesis origins | 25 separate dossiers | PASS |
| all domain families | 9/9 represented | PASS |
| at least ten countries/territories | 11 proposed catalog values | PASS as proposal; founder review pending |
| at least five languages | 6 proposed catalog values | PASS as proposal; founder review pending |
| 25 cataloged | 25 dossier-local catalog states; only 2 canonical admissions | PARTIAL |
| 25 completed policy decisions | 0 completed; 25 pending + uncertain | FAIL / BLOCKED ON HUMAN REVIEW |
| permitted, constrained, catalog-only, denied and uncertain | uncertain pending only | FAIL / BLOCKED ON EVIDENCE AND HUMAN REVIEW |
| at least 10 safely profiled | 2 imported E2 records at or beyond profile | FAIL (2/10) |
| at least 5 immutable observations | 2 imported E2 records at or beyond observation | FAIL (2/5) |

Twenty-three catalog admissions are in the generated human review queue. The
two canonical admissions reproduce already reviewed E2 identities; neither
claims a completed Atlas policy review.

New E3.2 public-origin network requests: **0**.

New E3.2 response bytes: **0**.

New E3.2 retained representation bytes: **0**.

## Operations evidence

Measured local host: Linux/amd64, Go `go1.26.5-X:nodwarf5`, AMD Ryzen 7 6800U.

- admission source: 101 files, 130,506 bytes;
- generated admission evidence: 28 files, 76,166 bytes;
- 25-dossier load and render: 15,039,904 ns/op, 2,138,442 B/op,
  47,553 allocs/op;
- complete 19-byte fixture spool verification: 443,062 ns/op, 59,678 B/op,
  953 allocs/op;
- human policy review time: not measured because none was performed;
- existing E2 catalog-admission review time: not recorded.

The 500-origin planning projection is explicitly non-evidentiary. A linear
static-artifact projection is 2,610,120 source bytes, 1,523,320 generated
bytes and 300.798 ms of cumulative render time. At the stated planning
assumption of 30–60 minutes of human review per origin, 500 reviews require
250–500 hours. The current 64-origin batch bound requires at least eight
reviewed batches. Live representation storage and network cost are excluded.

Machine-readable evidence is in
`generated/e3/operations-evidence.json`.

## Target-host read-only check

Commands changed no target state.

- kernel: Linux `6.12.90+deb13.1-amd64`;
- systemd: `257 (257.13-1~deb13u1)`;
- Caddy: active;
- E3.2 unit: absent;
- E3.2 state directory: absent;
- E3.2 cgroup firewall: not active;
- target worker CPU/RSS: not measured because no worker is deployed.

The target currently uses Tailscale MagicDNS addresses inside ranges denied by
the candidate. Installing it unchanged would fail DNS safely. No private-range
exception was added. Activation requires a reviewed public/isolated resolver
boundary or port-scoped DNS broker, followed by target DNS-rebinding,
private-range and control-plane reachability tests.

## Exact commands executed

```bash
make clean
make build
make test
make demo
make demo-e2
make demo-e3
make demo-e3-worker

go test -race ./...
go vet ./...
go test -coverprofile=/tmp/twirx-e3-2-cover.out ./...
go tool cover -func=/tmp/twirx-e3-2-cover.out
go test -json ./...

go test -run='^$' \
  -bench='BenchmarkAdmissionLoadAndRender25|BenchmarkVerifyEvidenceSpool' \
  -benchtime=2s -count=1 -benchmem \
  ./internal/admission ./internal/egressworker

go run ./cmd/twirx-admission validate \
  --root . --admissions atlas/admissions --fixtures atlas/fixtures
go run ./cmd/twirx-admission review-queue \
  --root . --admissions atlas/admissions --fixtures atlas/fixtures
go run ./cmd/twirx-admission check-canonical \
  --root . --admissions atlas/admissions --fixtures atlas/fixtures

systemd-analyze verify deploy/egress/twirx-egress-worker@.service
systemd-analyze security --offline=yes \
  deploy/egress/twirx-egress-worker@.service
shellcheck deploy/egress/verify-target.sh
bash -n deploy/egress/verify-target.sh
./scripts/check-docs.sh
```

GCC and Clang each built the observation, E2 result and E2 artifact C
verifiers with `-std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow
-Wpedantic`. `make test` additionally ran Clang ASan/UBSan verification and
both 5,000-run libFuzzer suites.

The target host was inspected read-only over the existing SSH agent for kernel,
systemd, Caddy, unit and state-directory status. No file was copied and no
service, firewall, DNS, repository or deployment state was changed.

## Local results

- 285 named Go test pass events;
- 14 Go fuzz targets passed;
- Go race detector passed;
- Go vet passed;
- aggregate statement coverage: 66.0%;
- GCC and Clang strict builds passed;
- ASan and UBSan passed;
- observation C vectors: 2 valid accepted, 14 invalid rejected, corruption
  rejected;
- E2 shared Go/C conformance passed;
- both C libFuzzer targets completed 5,000 runs without a crash;
- admission generation reproduced byte-for-byte with aggregate SHA-256
  `eef4299aaf4ba6105ce578b76736e1f19181515bdcbd8b34223580b6b719ddfb`;
- E1 demonstration passed;
- E2 deterministic transcript, Go verification, restricted-C verification,
  MCP invocation and offline replay passed;
- E3.1 selection, metrics and zero-job frontier passed;
- literal-loopback Observatory worker and offline replay passed;
- docs, JSON, shell syntax and local systemd analysis passed.

`systemd-analyze verify` produced the expected local warning that
`/srv/twirx/current/bin/twirx-egress-worker` does not exist on this development
host. No unit was installed to suppress that warning.

## Unresolved risks and deviations

1. Twenty-three founder catalog admissions and all 25 human policy decisions
   remain pending.
2. The outcome-distribution, 10-profile and 5-observation acceptance floors
   are not met.
3. Target-host isolation is a configuration candidate, not enforcement
   evidence; the DNS boundary must be resolved first.
4. Worker memory evidence is Go allocation data, not target-host peak RSS.
5. A host/root compromise can replace worker code and spool artifacts together;
   release signing and external transparency remain future controls.
6. The local circuit is not distributed fleet-wide abuse or health state.
7. No failure manifest is published for a failed retrieval; the failed sealed
   order remains non-reusable and its circuit state is preserved.
8. No live response-byte accounting exists because no live request was
   authorized.
9. No all-500 live processing, browser, model, write/payment action, automatic
   canon promotion or arbitrary public URL capability was added.

## Next recommended gate action

Founder review should proceed in this order:

1. review the 23 pending catalog dossiers and correct proposed jurisdiction,
   language, publisher and authority metadata;
2. explicitly admit or reject each catalog record;
3. approve exact robots evidence orders only for admitted origins;
4. complete policy evidence and record all five required outcome classes only
   where evidence supports them;
5. implement and verify the target DNS boundary without broad private egress;
6. activate a bounded pilot only after separate founder approval;
7. produce at least 10 safe profiles and 5 immutable observations;
8. rerun this complete suite and update this report with real request, byte,
   retained-size, CPU, peak-RSS and human-review measurements.

Until then, E3.2 is reviewable infrastructure—not an admitted live-origin
gate.
