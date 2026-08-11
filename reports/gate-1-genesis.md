# Gate 1 Genesis evidence report

**Status:** Local acceptance passed; implementation merged; remote CI startup blocked

**Evidence date:** 2026-08-10  
**Implementation commit tested:** `1d17d5b541176fdb6b742caa48f0f55a14dfa206`  
**Baseline commit:** `6df221bde6f2f7e0535df178104e8b1a93c11bb8`

This report is committed after the implementation it describes so its own
commit identifier is intentionally not the tested implementation identifier.

## Baseline audit

The bootstrap clean build, existing tests, and demo completed at the baseline
commit. The audit still found Gate 1 behavior gaps: Go and C applied different
text bounds, CBOR text was not validated as UTF-8, retrieval time and body
bounds were not a complete wire rule, JSON accepted duplicate keys without an
explicit policy, JSON resource bounds were absent, result publication was a
direct write, the Go/C corpus was not shared, fuzz targets were absent, and the
network regression matrix was incomplete.

No existing command failure was treated as evidence that these security and
interoperability properties were already satisfied.

## Toolchain and host

| Component | Version used |
|---|---|
| Host | Linux 7.1.3-arch1-1, x86_64 |
| Go | `go1.26.5-X:nodwarf5 linux/amd64` |
| GCC | `16.1.1 20260625` |
| Clang | `22.1.8` |
| Python | `3.14.6` |
| Bash | `5.3.15(1)-release` |
| GNU Make | `4.4.1` |
| ShellCheck | `0.11.0` |
| Git | `2.55.0` |

The repository has no `go.sum` and this gate adds no third-party Go runtime
dependency.

## Commands and outcomes

The full acceptance sequence was run from a clean generated-artifact state:

```bash
make clean && make build && make test && make demo
```

Outcome: pass. `make test` ran normal Go tests, all four Go fuzz targets, the
Clang ASan/UBSan C corpus, 5,000 libFuzzer executions, the offline end-to-end
test, and the documentation navigation check. `make demo` observed the local
fixture, verified it in Go and C, stopped the origin, and completed extraction
from CAS evidence.

Additional commands:

```bash
go test -count=1 -json ./...
go vet ./...
go test -race ./...
go test -cover ./...
make benchmark
test -z "$(gofmt -l cmd internal)"
shellcheck scripts/*.sh
git diff --check
./scripts/check-docs.sh

gcc -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic \
  -o /tmp/tw-verify-c-gcc \
  verifier/c/main.c verifier/c/observation.c verifier/c/sha256.c
clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic \
  -o /tmp/tw-verify-c-clang \
  verifier/c/main.c verifier/c/observation.c verifier/c/sha256.c
clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-c-main.plist verifier/c/main.c
clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-c-observation.plist verifier/c/observation.c
clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/tw-c-sha256.plist verifier/c/sha256.c
```

All passed. A repository scan for common private-key, GitHub token, AWS access
key, and Slack token patterns returned no match outside generated directories
and Git metadata.

## Test and conformance totals

| Evidence | Passed | Failed | Notes |
|---|---:|---:|---|
| Named Go test events | 94 | 0 | Seven tested packages; two command packages have no test files |
| Observation vectors in Go | 16 | 0 | 2 accepted, 12 envelope rejects, 2 evidence rejects |
| Observation vectors in C | 16 | 0 | Same committed inputs and expectations as Go |
| Corrupted CAS checks | 2 | 0 | One Go and one C check |
| Extraction vectors | 11 | 0 | 5 accepted, 6 rejected |
| Network-policy tests | 8 | 0 | Local fixtures only |
| Offline end-to-end test | 1 | 0 | Extraction succeeds after the origin stops |
| Documentation navigation | 1 | 0 | Configuration parsed and all navigation targets existed |

The final Go fuzz smoke run reported 138,620 observation-parser executions,
55,183 JSON Pointer executions, 83,640 manifest-decoder executions, and 72,160
bounded-extraction executions. These counts are host- and scheduling-dependent;
the gate is the absence of a panic or failing invariant, not a performance
claim.

## C sanitizer and fuzz results

The independent verifier was compiled with:

```text
-fsanitize=address,undefined -fno-omit-frame-pointer
```

It accepted both valid observation vectors, rejected all 14 invalid envelope
or evidence vectors, and rejected a post-validation CAS corruption. ASan and
UBSan emitted no findings.

The parser-only C harness was compiled with libFuzzer, ASan, and UBSan. It ran
5,000 mutations seeded by every shared observation vector without a crash or
sanitizer finding. The harness performs no filesystem or network operation.

## Remote CI evidence after publication

PR #1's reachable merge commit is
`ac4f9948ad21b319b11f9caeee9bd4e472c39780`. Before public release, its
author metadata was remediated to the maintainer's verified GitHub noreply
identity. The file tree, topology, names, message, timestamps, and
implementation behavior were preserved.
The merge-triggered GitHub Actions run
[`31363566671`](https://github.com/banga-agents/TWIRX/actions/runs/31363566671)
ended with `startup_failure` before GitHub created a job. The latest PR #1 run
was also a startup failure and GitHub refused an explicit `gh run rerun` with
`This workflow run cannot be retried`.

GitHub's workflow API reports the intended `ci` workflow at
`.github/workflows/ci.yml` as active, but it has zero associated runs. Every
repository event is instead recorded against a separate deleted workflow whose
path is `BuildFailed`, with an empty workflow name, no jobs, and no downloadable
logs. This means no intended CI command reached a runner.

A CI-only registration test renamed `ci.yml` to `ci.yaml` with byte-for-byte
identical content. The Git diff was a 100% rename with zero insertions and zero
deletions, and actionlint 1.7.12 passed. Fresh push and pull-request events still
failed at startup; see run
[`31365647287`](https://github.com/banga-agents/TWIRX/actions/runs/31365647287).
Disabling and re-enabling the active workflow and then closing and reopening the
unmerged CI-only PR produced the same result; see run
[`31365777371`](https://github.com/banga-agents/TWIRX/actions/runs/31365777371).
The original workflow path was restored in a new commit without rewriting
history, leaving no net workflow diff from merged `main`.

The remaining limitation is external to the workflow document: GitHub Actions
availability, budget, or account eligibility for GitHub-hosted runners in this
private repository must be resolved in the repository owner's Actions and
billing settings. Until then, remote job results do not exist; the local Gate 1
evidence in this report remains the only executed validation evidence.

## Coverage and benchmark record

Statement coverage from `go test -cover ./...`:

| Package | Coverage |
|---|---:|
| `internal/adapter` | 71.3% |
| `internal/atomicfile` | 56.8% |
| `internal/cas` | 63.0% |
| `internal/cborlite` | 61.3% |
| `internal/jsonbounded` | 76.5% |
| `internal/observation` | 56.1% |
| `internal/safefetch` | 76.7% |

The command packages report 0% because they do not have direct Go tests; their
current path is exercised by the shell end-to-end test and demo.

Benchmark on an AMD Ryzen 7 6800U, Linux/amd64, 16 logical workers:

```text
BenchmarkResolveJSONPointer-16  4492222  271.7 ns/op  128 B/op  8 allocs/op
```

This is a host-specific baseline record, not a general performance guarantee.

## Invariants evidenced

- The observation v1 field order and byte bounds agree across CDDL, Go, and C.
- Deterministic CBOR rejects non-shortest integers, indefinite forms, wrong
  field and digest lengths, invalid UTF-8, U+0000, non-canonical UTC times,
  trailing bytes, and bodies over 2 MiB.
- Evidence is read from the digest-derived CAS path and its size and SHA-256
  digest are rechecked before extraction.
- Manifest and source JSON reject duplicate keys, trailing values, lone
  surrogate escapes, and inputs beyond declared byte, nesting, scalar,
  container, and token limits.
- Required source fields fail closed. Optional missing fields are explicit
  `unresolved` results with provenance and without fabricated lexical values.
- A resolved empty string remains distinct from `unresolved`.
- Native lexical values are retained before declared ASCII-only transforms;
  transformed semantic values do not overwrite them.
- Every emitted field carries request URL, final URL, retrieval time, body and
  observation digests, adapter identity and digest, extraction method,
  locator, transform chain, and mapping relation.
- Observation metadata, body references, and results use bounded temporary
  files and atomic rename. The trusted Go extraction path does not execute a
  shell or contact the network.
- The C parser and verifier have no network path and do not write canonical
  state.

## What Gate 1 proves

For the controlled source representation and the committed adversarial corpus,
the implementation can bind retrieved bytes to a canonical observation,
independently verify those bytes in Go and C, and deterministically extract a
source-native statement plus an explicit semantic interpretation and complete
derivation record while offline.

It proves rejection behavior for the bounded classes represented by the
public vectors. It also proves that the local demo does not need the origin to
remain available after observation.

## What Gate 1 does not prove

- It does not establish that provider content, a semantic mapping, or a
  retrieval timestamp is objectively true.
- It does not establish publisher identity or authority.
- It does not cover arbitrary websites, HTML, JavaScript, browser execution,
  model output, writes, payments, registry operation, or multi-tenant service.
- It does not establish production DNS-rebinding resistance, egress isolation,
  or safe public arbitrary-URL deployment.
- It does not provide signatures, release provenance, external transparency,
  or protection against a host compromise that replaces code and evidence.
- Finite tests and short fuzz runs do not prove the absence of parser defects.

## Residual risks

1. URL and resolved-address checks are application-layer controls. Production
   deployment still needs separate worker/control networks, DNS and egress
   enforcement, metadata isolation, and quotas.
2. Observation v1 records request and final URL but not the redirect chain or
   selected transport headers. A later immutable transport-evidence artifact
   is needed without mutating v1.
3. The adapter runtime has no independent second implementation. Its manifest
   and result formats remain pre-stable and unsigned.
4. Atomic publication is per file, not a transaction spanning every file in
   an observation bundle. Destination directories and CAS roots are assumed
   to be trusted configuration, and filesystem durability semantics vary.
5. The corpus is intentionally small and the fuzz durations in normal tests
   are smoke tests. Longer scheduled fuzzing and external review remain
   necessary.
6. Direct command-package coverage is absent even though the shell workflow
   exercises their current happy path. CLI error-path tests should be added at
   the next maintenance gate.
7. Local filesystem compromise can replace implementation and evidence
   together. Signed releases and external transparency are out of Gate 1.

## Deviations and next gate

There were no scope-expanding deviations from Work Order 001. The C fuzz
harness was implemented because the installed Clang toolchain supported
libFuzzer without a new runtime dependency. The validation ADR additionally
closes lone-surrogate and timestamp-canonicalization ambiguity before release.

The next recommended gate is independent review and CI reproduction of this
report, followed by Gate 2 only if no Gate 1 invariant regresses. Public
deployment is not the next gate.
