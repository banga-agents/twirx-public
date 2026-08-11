# E3 Atlas-500 full-catalog runtime evidence

Status: **IMPLEMENTATION PASS / E3 GATE NOT ADMITTED**

Implementation commit: `39afd1bd5b26a45c40a9bfd17409ac0a9bb35f8c`

Evidence date: 2026-08-11

## Outcome

The real read-only Atlas control plane now operates at the complete selected
catalog breadth instead of treating 500 as a static-file count:

- one derived admission work item exists for every selected origin;
- one deterministic frontier outcome exists for every selected origin;
- the actual loopback HTTP service enumerates and directly describes all 500;
- every domain-family quota and the current orthogonal-state filters are
  checked against the service;
- 100 complete concurrent catalog rounds produced 50,000 successful lookups;
- eight malformed or unauthorized requests were rejected and a subsequent
  valid request succeeded;
- a service restart reproduced byte-identical status;
- the stable 500-response set digest is
  `sha256:cda36528db7b9e303505a28dbf5b6e91fdf7b77384d4efe3d1f5ee9306831817`.

This proves the current control plane at 500-origin scale. It does not claim
that 500 origins have completed policy review, have been fetched, compiled, or
made live.

## Admission and frontier truth

The generated admission queue reports:

```text
500 selected work items
25 prepared dossiers
475 not-prepared dossiers
23 pending human catalog reviews
2 previously completed catalog admissions
500 pending policy reviews
0 completed policy reviews
```

The deterministic frontier reports:

```text
500 decisions
498 blocked: catalog_review_pending
2 blocked: policy_review_pending
0 retrieval jobs
```

The work queue never manufactures an evidence dossier for a missing origin.
The frontier no longer omits the 498 selected origins absent from the
canonical registry. Missing state becomes an explicit fail-closed outcome.

## Runtime results

Measured host: Linux/amd64, Go `go1.26.5-X:nodwarf5`, AMD Ryzen 7 6800U.

| Measurement | Result |
| --- | ---: |
| HTTP requests | 50,000 |
| Successful / failed | 50,000 / 0 |
| Concurrent workers | 32 |
| Response bytes | 32,076,700 |
| Wall time | 2.218219 s |
| Throughput | 22,540.61 requests/s |
| Mean request latency | 1.333 ms |
| p50 / p95 / p99 | 1.007 / 3.591 / 5.232 ms |
| Maximum request latency | 12.158 ms |
| Atlas-service resource samples | 32 |
| Maximum Atlas-service RSS | 23,460 KiB |
| Maximum threads / file descriptors | 24 / 42 |
| Final sampled CPU ticks | 866 |
| Stress-client cumulative allocation | 695,633,192 bytes |
| Stress-client ending heap | 3,282,648 bytes |

These are local literal-loopback measurements. They are not public TLS, VPS,
live-origin, database, storage, or multi-tenant capacity claims.

## Artifact reproducibility

```text
selection.json       sha256:3b07836c4b490e34a6285e7886a5d8589b79ed54ae906f0741ea976e33d2ef7f
atlas-queue.json     sha256:9302c75071b98ed4addb547ae9cb3e8910ffca09a2a7ca4c60b63c691705c385
atlas-metrics.json   sha256:a8c6057e07969131b7923d133a9a9ce255d445d50818088004c28e0cf7708a88
```

E2, E3 metrics, and E3 admission artifacts regenerated without a tracked
diff.

## Implemented invariants

- The admission queue covers exactly 500 unique selected identities.
- A missing dossier is `not_prepared`, never cataloged or approved.
- Pending policy remains `pending + uncertain` and cannot create a job.
- The dry-run frontier covers exactly 500 selected identities once each.
- Candidate-only and cataloged-but-policy-pending reasons remain distinct.
- The frontier carries origin IDs rather than retrieval URLs and declares
  network access disabled.
- The Atlas stress client accepts only an exact literal-loopback HTTP origin
  with an explicit port, disables proxies and redirects, and pins the dialed
  address.
- The public HTTP surface remains GET-only and has no origin-fetch client.
- The work queue is generated from selected and digest-bound per-origin
  artifacts; the canonical registry is not a manual 500-entry editing surface.
- E1/E2 evidence, native-first semantics, read-only behavior, replay-only
  public invocation, and independent C verification remain unchanged.

## Exact commands executed

The complete validation sequence against the implementation commit was:

```bash
make clean && make build && make test && make demo && make demo-e2 && \
  make demo-e3 && make demo-e3-worker && make stress-e2 && \
  make stress-e3-500 && go test -race ./... && go vet ./...
```

Supplemental evidence commands were:

```bash
go test -count=1 -json ./... > /tmp/twirx-e3-atlas-500-tests.json
go test -coverprofile=/tmp/twirx-e3-atlas-500-cover.out ./...
go tool cover -func=/tmp/twirx-e3-atlas-500-cover.out

make generate-e2
make generate-e3
make generate-e3-admission
git diff --exit-code -- generated/e2 generated/e3

sha256sum atlas/genesis-500/selection.json \
  generated/e3/admission/atlas-queue.json \
  generated/e3/atlas-metrics.json
```

## Validation results

- 300 named Go test pass events, zero failures, zero skips;
- aggregate Go statement coverage: 66.3%;
- 14 Go fuzz targets passed;
- Go race detector passed;
- Go vet passed;
- GCC 16.1.1 and Clang 22.1.8 strict builds passed;
- ASan and UBSan passed;
- E1 restricted-C vectors: two valid accepted and 14 invalid rejected;
- E2 shared Go/C conformance passed;
- both C libFuzzer targets completed 5,000 runs without a crash;
- E1, E2, E3, and E3 worker demonstrations passed;
- E2 replay stress passed with 100 invocations and five independently
  verified bundles;
- Atlas-500 stress passed with 50,000/50,000 successful lookups;
- generated artifacts reproduced without a tracked diff;
- documentation, JSON syntax, shell syntax, and formatting checks passed.

## Files changed

- full-catalog work queue: `internal/admission`, `cmd/twirx-admission`,
  `generated/e3/admission/atlas-queue.json`;
- full-selection frontier: `internal/atlas`;
- bounded full-catalog stress runtime: `internal/atlasstress`,
  `cmd/twirx-atlas`, `scripts/stress-e3-500.sh`, `stress/e3-atlas-500.md`;
- language-neutral schemas: Atlas work queue and corrected frontier reason
  vocabulary under `schemas/json`;
- build, architecture, security, task, protocol, README, and Mintlify
  documentation.

The preserved untracked implementation packs, website work, VPS reports, and
the unrelated `meridian-velo` repository were not modified or staged.

## Unresolved risks

1. Four hundred seventy-five selected origins still need evidence-backed
   dossier preparation; 23 prepared proposals need founder catalog review.
2. All 500 policy reviews remain pending. No direct public-origin retrieval is
   authorized by the Atlas.
3. The E3 depth floors remain unmet: 300 profiles, 100 observations, 50 native
   schemas, 25 compiled adapters, 12 semantic operation families, eight live
   origins, and an operating scheduler.
4. The target-host sealed worker remains disabled and undeployed; its DNS
   isolation conflict with Tailscale MagicDNS remains unresolved.
5. The stress result covers the stateless read-only control plane. It does not
   measure real-origin latency, evidence-spool I/O, parsing, compilation,
   semantic invocation, public TLS, or multi-tenant abuse controls.
6. The stress client's cumulative allocation is workload churn, not peak RSS;
   the separately sampled service RSS is the process-capacity measurement.
7. Atlas validation still has one implementation. Its schemas are
   language-neutral, but an independent Atlas verifier is future work.

## Deviations

- No selected public origin was fetched. This preserves the prior explicit
  prohibition on processing all 500 through live retrieval before human
  policy decisions and target-host egress admission.
- No candidate was promoted, no policy was approved, no scheduler was enabled,
  and no adapter or mapping entered canon.
- No merge or deployment was performed.

## Next recommended gate

Proceed with **E3.3 — Genesis-500 dossier completion and reviewed retrieval
waves**:

1. prepare the remaining 475 bounded dossiers in deterministic batches;
2. expose a founder review surface for identity and policy artifacts;
3. record real human decisions, including deny, catalog-only, profile-only,
   constrained, permitted, and uncertain outcomes where evidence supports
   them;
4. resolve and verify the VPS DNS/egress boundary;
5. execute retrieval only for explicitly admitted work orders;
6. advance the first evidence-backed waves toward 300 profiles, 100
   observations, 50 native schemas, 25 adapters, 12 semantic families, and
   eight live origins while the complete 500-policy review proceeds.

The 500-origin breadth test now passes. E3 admission still correctly fails on
the real depth and human-review work rather than on missing control-plane scale.
