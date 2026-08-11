# E3.1 policy and dry-run frontier evidence

**Candidate commit:** `6210e6be1133e6676f70578a4629a0f7b5d6158d`

**Recorded:** 2026-08-10

**Implementation status:** PASS for the offline E3.1 policy, robots, and
frontier control candidate

**Gate status:** FAIL for E3.1 admission and FAIL for E3 admission; the
required human reviews, controlled egress boundary, and all later maturity
floors remain absent

## Outcome

The repository now contains the controls needed to begin E3.1 review without
turning the A0 selection into network authority:

- a separate, bounded, duplicate-rejecting policy artifact set;
- exact SHA-256 policy-set binding from A2 registry records;
- field agreement checks between policy and registry access claims;
- explicit `allow`, `deny`, and `review_required` outcomes;
- an offline RFC 9309 parser/evaluator with shared conformance fixtures;
- explicit per-origin scheduler state, budgets, cooldown, failures, and
  bounded priority factors;
- a deterministic dry-run frontier containing origin IDs but no destination
  URLs;
- generated metrics that expose policy-record breadth separately from A2
  maturity.

No origin was fetched, profiled, observed, or promoted. `atlas/policies.json`
and `atlas/registry.json` both remain empty. The derived evidence remains:

```text
500 A0 candidates
0 reviewed policy records
0 A1–A9 origins
0 dry-run jobs
network_access: disabled
```

## Artifact identities

```text
selection
sha256:209c3629ad58c60ea4477436332ec4847b3f0ecaf7b091046923c95c87f53fc5

policy set
sha256:1752838b00fba147abc0c1d383e8acd6692f00eaa4b662c3b84c468c884ea566
```

The policy digest is over the exact committed JSON bytes. It is an artifact
identity, not a claim that policy review has occurred.

## Invariants implemented

1. **Selection is not admission.** Policy records must refer to an exact
   selected identity, and an A2 registry entry must bind a policy that exists.
2. **Exact policy evidence.** The registry carries the SHA-256 identity of the
   whole reviewed policy set and must reproduce the applicable decision,
   review time, terms, attribution, authentication, rate, retention, risk,
   notes, and robots digest.
3. **Unknown fails closed.** Policy absence grants nothing. Unreachable,
   redirect-limit, and not-observed robots states cannot support `allow`.
4. **Genesis stays public and read-only.** An allowed E3 policy requires
   `none_required` authentication and accepted risk review. No write or payment
   path exists.
5. **Robots is not authority.** Robots results inform crawler behavior but do
   not replace terms, risk, or maintainer review.
6. **Bounded untrusted parsing.** Robots input is limited to 500 KiB of valid
   UTF-8; target strings are limited to 16 KiB; malformed lines do not erase
   parseable rules; oversized or invalid documents fail closed.
7. **Deterministic matching.** Exact user-agent groups are combined
   case-insensitively, wildcard is fallback only, paths match case-sensitively,
   longest specificity wins, and `allow` wins an equivalent tie.
8. **Scheduler state is explicit.** Refresh class, request/storage ceilings,
   failures, cooldown, and each priority factor are bounded and validated.
9. **Dry run is not egress.** Plans require an explicit canonical UTC time,
   contain no destination URL, mutate no state, and declare network access
   disabled.
10. **Protocol remains language-neutral.** The specification and JSON
    artifacts define the control behavior; the Go packages are an
    implementation, not normative authority.

## RFC basis

The parser follows RFC 9309 group selection, longest-match, parseable-rule,
access-result, caching-boundary, and 500 KiB requirements. The corpus also
supports a leading `*` pattern as the interoperability behavior in section 5.1
and reported erratum 7995. Reported errata are identified as such and are not
represented as accepted changes to the RFC.

## Commands executed

Focused implementation and artifact validation:

```bash
gofmt -w internal/atlas/*.go internal/atlasapi/*.go internal/robotstxt/*.go cmd/twirx-atlas/*.go
python3 -m json.tool atlas/policies.json >/dev/null
python3 -m json.tool schemas/json/atlas-registry.schema.json >/dev/null
python3 -m json.tool schemas/json/atlas-policy-set.schema.json >/dev/null
python3 -m json.tool schemas/json/atlas-frontier-plan.schema.json >/dev/null
python3 -m json.tool conformance/robots/v1/cases.json >/dev/null
go test ./internal/atlas ./internal/atlasapi ./internal/robotstxt ./cmd/twirx-atlas
go vet ./internal/atlas ./internal/atlasapi ./internal/robotstxt ./cmd/twirx-atlas
go test ./...
go vet ./...
make build
make generate-e3
bin/twirx-atlas validate --root .
bin/twirx-atlas plan --root . --at 2026-08-10T00:00:00Z
git diff --check
```

Complete regression and demonstration matrix:

```bash
make generate-e3
go test -race ./...
make test
make demo
make demo-e2
make demo-e3
```

Independent compiler build and intended-diff checks:

```bash
make -B CC=gcc bin/tw-verify-c bin/tw-verify-result-c bin/tw-verify-e2-artifact-c
go test -json ./... | rg -c '"Action":"pass".*"Test":'
git diff --cached --check
```

The staged intended diff was also checked for common private-key, GitHub token,
AWS key, OpenAI-style key, password-assignment, seed/mnemonic, Gmail, identity
document, and street-address patterns. No match was found. This was a targeted
change scan, not a replacement for the previously recorded full-history public
readiness audit.

## Results

| Check | Result |
|---|---:|
| Named Go test pass events | 227 |
| Go race run | PASS |
| Go vet | PASS |
| Go fuzz targets | 11 PASS |
| New policy-set fuzz target | PASS |
| New robots parse/evaluate fuzz target | PASS |
| Robots conformance cases | 16 PASS |
| C observation vectors | 2 valid accepted; 14 invalid rejected |
| E2 shared Go/C vectors | PASS |
| C observation libFuzzer | 5,000 runs, no crash |
| C E2 libFuzzer | 5,000 runs, no crash |
| GCC C2x builds | PASS |
| Clang ASan/UBSan builds and tests | PASS |
| E1 end-to-end demonstration | PASS |
| E2 deterministic agent transcript | PASS |
| E3.1 offline policy/frontier demonstration | PASS |
| Documentation navigation check | PASS |
| Intended-diff secret/personal-data pattern scan | PASS |

Toolchains observed:

```text
Go go1.26.5-X:nodwarf5 linux/amd64
GCC 16.1.1 20260625
Clang 22.1.8
```

## Files changed

- Policy and frontier implementation: `internal/atlas`, `atlas/policies.json`,
  `cmd/twirx-atlas`, and `internal/atlasapi`.
- Robots implementation and evidence: `internal/robotstxt` and
  `conformance/robots/v1/cases.json`.
- Language-neutral formats: `schemas/json/atlas-policy-set.schema.json`,
  `schemas/json/atlas-frontier-plan.schema.json`, and the updated registry
  schema.
- Generated evidence and demonstration: `generated/e3/atlas-metrics.json`,
  `scripts/demo-e3.sh`, and `Makefile`.
- Architecture, security, task, protocol, README, and Mintlify documentation.

The Makefile correction forces Go command targets to rebuild while retaining
the Go build cache. Without that correction, an existing binary could be
mistakenly reused after its source changed, causing generated evidence to come
from stale code.

## Unresolved risks and exclusions

1. None of the 500 candidates has an admitted identity or policy review.
2. The parser is not connected to live retrieval; no cache, conditional
   request, redirect, sitemap, feed, or observatory workflow is operating.
3. The dry-run frontier is not a daemon, does not persist state transitions,
   and has never operated a daily schedule.
4. There is no independent second implementation of robots matching.
5. A human reviewer can still misinterpret robots, terms, identity, authority,
   retention, attribution, or risk.
6. The separate host-enforced egress worker, private-range controls, disposable
   isolation, monitoring, and revocation procedures remain unimplemented.
7. E2 hardened public deployment evidence remains outstanding.
8. E3.2–E3.5 and every E3 acceptance floor remain outstanding.
9. No live claim, publisher verification, semantic admission, model training,
   browser execution, write action, or arbitrary-URL fetch is authorized.

## Deviations

No requested scope was weakened. The implementation deliberately stopped
before human policy decisions, origin contact, deployment, or maturity
promotion. No third-party runtime dependency was added. The website, VPS,
repository visibility, and unrelated repositories were not changed.

## Recommendation

**PASS the E3.1 offline control implementation candidate. Do not admit E3.1 or
activate egress.** After the integration and E3.1 PRs are reviewed, the next
gate is a founder-approved identity and policy review workflow beginning with
TWIRX's publisher-authored origin, followed by a dedicated isolated retrieval
worker and local-fixture proof of robots retrieval. The first real external
origin must remain blocked until both the policy artifact and the egress
boundary pass review.
