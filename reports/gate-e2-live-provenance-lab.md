# Engineering Gate E2 — Live Provenance Lab evidence

**Implementation status:** Local acceptance passed

**Gate admission:** Blocked pending hardened public deployment and live-surface evidence

**Release label:** Genesis Preview

**Evidence date:** 2026-08-10

**Implementation commit tested:** `88431ffcaf3d3f37a85eaed3dddbf7129ba00b27`

**Baseline commit:** `c85030dd4537f2a34cfa07f938826337c3a40c40`

This report is committed after the implementation it describes, so its own
commit identifier is intentionally not the tested implementation identifier.

## Outcome

The E2 implementation candidate provides three reviewed catalog entries, five
read-only operations, canonical result and manifest-last proof formats, fresh
and offline replay, field-level native and semantic views, generated CLI/JSON
Schema/OpenAPI/local-MCP bindings, a browser Lab instrument, and independent
Go and restricted-C verification.

Local implementation acceptance passed. E2 is not admitted and the public
service is not deployed. The work order requested a canonical result that
binds both its own digest and the digest of a manifest that hashes that result.
Those requirements form a cryptographic cycle. The Genesis steward accepted
ADR 003 and its acyclic graph:

```text
content artifacts → result-core bytes → result digest → final manifest
                  → bundle ID → API publication record
```

The result-core contains no self-digest. The bundle ID is the SHA-256 digest of
the exact canonical manifest bytes. API, CLI, MCP, and UI wrappers expose both
detached identifiers. Shared Go/C vectors now explicitly reject missing,
cyclic, substituted, malformed, symlinked, and trailing-byte cases. The
remaining admission blocker is deployment behind the required VPS boundary
and production evidence from that surface.

## Delivered surface

- TWIR Core 0.1 contract source with primitive, record, optional, list,
  resource, operation, typed error, read effect, evidence, native/semantic
  references, closure, and resolved/unresolved state.
- Three reviewed catalog entries: the TWIRX publisher interface, the
  Controlled Origin Lab, and the official World Bank Indicators JSON API.
- Five operations generated into CLI reference, JSON Schema, OpenAPI 3.1, and
  MCP tool definitions from `contracts/e2/contracts.json`.
- Canonical `tw.result/0.2`, `tw.transport/0.2`, semantic closure, canonical
  input, adapter, contract, transcript, and manifest artifacts.
- Ten-file proof bundles written manifest-last; empty, missing, non-regular,
  symlinked, oversized, trailing, corrupted, and digest-mismatched artifacts
  fail closed.
- Go verification re-extracts the immutable representation and requires a
  byte-identical canonical result. Restricted C validates the result,
  manifest, observation, body, semantic closure, and digest graph without
  network access or writes.
- A local MCP stdio server performs the actual initialize/tool-call lifecycle.
  It is an isolated binding and not a normative protocol implementation.
- A loopback-only HTTP service exposes the required catalog, invocation,
  result, provenance, bundle, and well-known routes. The client cannot supply
  a URL or destination.
- The browser Lab shows admission, fresh-mode capability, explicit
  `not_probed` health, typed/native/transformation/evidence/verification views,
  generated schema, and bundle download. Its requests explicitly omit
  credentials.
- The non-fetching origin submission page opens a human-review email and does
  not crawl, compile, or activate the submitted URL.

## Full acceptance commands

The complete E1 and E2 sequence was run from a clean generated-artifact state
at the tested commit:

```bash
make clean && make build && make test && make demo && make demo-e2
```

Outcome: PASS. This preserved and reran the exact E1 commands while extending
the suite with three new Go fuzz targets, shared E2 Go/C vectors, a second C
libFuzzer target, and the deterministic E2 agent transcript.

Supplemental commands:

```bash
go test -count=1 -json ./...
go test -race ./...
go test -cover ./...
go vet ./...
make generate-e2
git diff --exit-code -- generated/e2
gofmt -l cmd internal
shellcheck scripts/*.sh
node --check lab/static/app.js
./scripts/check-docs.sh
git diff --check

gcc -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic \
  -o /tmp/tw-e2-final-gcc \
  verifier/c/e2_main.c verifier/c/e2.c \
  verifier/c/observation.c verifier/c/sha256.c
clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic \
  -o /tmp/tw-e2-final-clang \
  verifier/c/e2_main.c verifier/c/e2.c \
  verifier/c/observation.c verifier/c/sha256.c
for source in verifier/c/e2_main.c verifier/c/e2_artifact_main.c \
  verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c; do
  output=/tmp/final-$(basename "$source").plist
  clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion \
    -Wshadow -Wpedantic -o "$output" "$source"
done
```

All passed with no race, vet, compiler, sanitizer, static-analyzer,
formatting, generator, documentation, or script finding. The final JSON test
stream contained 154 named Go test passes and no named failure or skip.

## Fresh-origin evidence

The final binary observed and verified both publisher-authored and external
fresh representations:

```bash
bin/twirx-lab invoke --root . --results var/e2-final-fresh-twirx \
  --origin twirx-project --operation project.getStatus --mode fresh
bin/twirx-lab verify --root . --results var/e2-final-fresh-twirx \
  --bundle <generated-directory>
bin/tw-verify-result-c <generated-directory>

bin/twirx-lab invoke --root . --results var/e2-final-fresh-world-bank \
  --origin world-bank-indicators \
  --operation development.getIndicator --mode fresh \
  --input country=CHL --input indicator=SP.POP.TOTL --input year=2024
bin/twirx-lab verify --root . --results var/e2-final-fresh-world-bank \
  --bundle <generated-directory>
bin/tw-verify-result-c <generated-directory>
```

Both produced four resolved fields and passed Go re-extraction and restricted-C
verification. The final run produced TWIRX result
`sha256:5b072d53786500d8ba16fb354d66f91f3444a2e19d0d83d6ec73c3b81dc6e495`
and World Bank result
`sha256:7dd49166746e839f8a604b9757e87537dfda077ba3f31a9a42618089dec8d7e5`.
These identify time-specific observed provider representations; they are not
claims of objective truth or permanent availability.

## Conformance and adversarial evidence

| Evidence | Result |
|---|---:|
| Named Go test passes | 154 |
| Go fuzz targets | 7 passed |
| E1 shared Go/C observation vectors | 16 passed |
| E2 shared bundle/artifact cases | 13 passed |
| E1 C libFuzzer executions | 5,000, no crash |
| E2 C libFuzzer executions | 5,000, no crash |
| C sanitizer configurations | ASan + UBSan, no finding |
| C production compilers | GCC + Clang, no warning/error |
| Concurrent identical publications | 16 passed under race detector |
| Generated operations | 5 from one contract source |
| Catalog entries | 3, including one external official API |

The shared E2 cases accept a valid bundle, result, manifest, and semantic
closure and reject a corrupt body, absent final manifest, symlinked body, and
trailing bytes in each canonical artifact class.

## Performance, load, and controlled comparison

The raw commands, host, scope, samples, artifact sizes, exclusions, and
summaries are in:

- `reports/e2-performance.md`;
- `reports/e2-load.md`;
- `reports/e2-browser-comparison.md`.

On the measured host, 20 replay invocations at concurrency 8 all succeeded.
A fresh process receiving 50 requests admitted exactly its configured burst of
20 and rejected 30 with HTTP 429. The browser comparison is one 156-byte local
fixture only and is explicitly not generalized.

## Security and deployment validation

The fresh fetch policy rejects non-public addresses and constrains both the
initial URL and every redirect to the reviewed catalog hostname. Response
bytes, redirects, and a strict representation-header allowlist are bounded.
Cookies, authorization, arbitrary headers, credentialed browser requests,
browser execution, model calls, shell execution, plugins, cgo, and `unsafe`
are absent from the trusted E2 request path.

Secret scans at the tested commit:

```bash
gitleaks git . --redact
gitleaks dir . --redact
GIT_ALLOW_PROTOCOL=file trufflehog git \
  file:///home/shiva/typed-web-genesis \
  --branch agent/e2-live-provenance-lab \
  --no-verification --no-update --json \
  --results=verified,unknown,unverified
```

Gitleaks 8.30.1 found zero items in 21 reachable commits and zero in the full
working tree, including the preserved untracked website and planning packs.
TruffleHog 3.96.0 reported zero verified items and one unverified URI heuristic:
the intentional embedded-credential rejection fixture in
`internal/safefetch/safefetch_test.go`.

The proposed Lab Caddy configuration was sent over SSH on standard input to
the VPS's installed Caddy binary and validated successfully without activating
or writing it:

```bash
ssh agent@116.202.50.220 \
  'caddy validate --config - --adapter caddyfile' \
  < lab/deploy/Caddyfile
```

No Lab files, service unit, Caddy site, DNS record, or firewall rule were
activated. The existing apex website, Mintlify documentation, Proton Mail DNS,
and unrelated VPS repositories and services were not changed.

## Exact unresolved risks and limitations

1. The Lab has not been deployed or tested through public Caddy/TLS. Public
   invocation, service-unit verification against installed paths, firewall
   validation, and `lab.twirx.org` surface checks therefore do not exist.
2. Application URL and DNS controls are not network isolation. The VPS still
   needs a separate least-privilege worker/egress boundary, private-range and
   metadata blocking, redirect tests at the network layer, quotas, monitoring,
   and incident/revocation procedures before public fresh-origin execution.
3. The in-memory rate limits reset on restart and are not distributed. There
   is no cache, origin-health probe, circuit breaker, or abuse-suspension
   control; health is honestly reported as `not_probed`.
4. The external provider can change content, schema, policy, availability, or
   terms. The committed replay fixture is evidence of one representation and
   does not confer provider authority or permanence.
5. Restricted C validates canonical structure, digest relationships,
   observation/body binding, and closure independently, but it does not
   independently execute the JSON extraction and semantic transformation.
   Go performs re-extraction; a second adapter implementation remains future
   work.
6. A sufficiently privileged local filesystem attacker can replace the
   implementation and evidence together. Regular-file checks and immediate
   rehashing narrow artifact substitution but do not defeat host compromise.
7. The E2 operated corpus covers publisher JSON, a controlled JSON fixture,
   and one official JSON API. JSON-LD, Atom/RSS/XML, structured HTML, and six
   real origins remain E3 work.
8. Performance and load evidence comes from one local host and small fixtures.
   It excludes public TLS, VPS contention, distributed load, cache behavior,
   and general origin latency.
9. Hosted GitHub CI still has no executed runner evidence. The complete local
    suite is recorded without converting the historical startup failure into a
    hosted-CI claim.
10. The private GitHub repository still has immutable old PR metadata described
    in `reports/public-readiness.md`. A fresh sanitized public repository or
    confirmed provider purge remains necessary before changing visibility.

## Deviations from the work order

- The impossible cyclic digest requirement was not implemented. Accepted ADR
  003 and the acyclic detached-digest publication record are the explicit
  substitute.
- The public Lab was not activated because its format is not admitted and the
  production egress boundary is not yet in place.
- No feed/XML origin was added; the work order makes that conditional, and E3
  is the defined multi-origin/source-class gate.
- Cache-hit evidence and live health are absent because E2 deliberately has no
  cache or health-promoting subsystem. The API says `not_probed`.
- No third-party MCP SDK was added. The bounded local stdio binding implements
  the required protocol subset with the standard library and remains isolated
  from the language-neutral core.

## Recommendation

**PASS as an E2 local implementation candidate with its publication topology
accepted. FAIL for gate admission and public Lab activation until the VPS
service/egress boundary passes deployment and public-surface validation.**

The next release action is founder review of the accepted topology and hardened
deployment plan. E3.0 control-plane work may proceed offline and stacked on
this candidate, but no live E3 ingestion or E2 deployment is authorized until
the service and egress boundary are reviewed and validated.
