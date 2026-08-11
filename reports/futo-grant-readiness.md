# FUTO grant-readiness audit

**Recommendation: FAIL — two operational release gates remain**

**Audit date:** 2026-08-11

**Current private release revision:**
`95a537a0a2d54e160a4b67643be2f637f5a7bea5`

**Fresh public repository:**
`https://github.com/banga-agents/twirx-public`

**Public root commit:**
`6646df57baac2b79539d375ec1a0d184e4015bc2`

**Immutable snapshot:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`

**Staged Query Lab releases:**
`20260811T153300Z-95a537a`

## Executive result

TWIRX now has a clean public repository, a verified evidence kernel,
independent Go/C verification, three explicit human policy decisions, one
sealed two-period Common Crawl acquisition, two source-native archive packets,
one genuine origin delta, a 500-identity immutable snapshot, a real
cross-origin query, a read-only snapshot runtime and a searchable Atlas-500
Lab interface. The complete local release suite passes.

The release is not yet ready to send because two operational gates remain:

1. authoritative DNS still does not publish `lab.twirx.org`, so the already
   staged and isolated Query Lab cannot obtain TLS or pass public wire tests;
2. no dedicated TWIRX storage identity or independent encrypted repository has
   been authorized, so a byte-identical off-host restore has not been proved.

The existing `meridianv2raw` bucket and `quantlab-archive-bx41` Storage Box
contents were deliberately left untouched. Reusing unrelated infrastructure
would violate the approved isolation boundary.

GitHub-hosted CI is a separately disclosed limitation: GitHub rejected the
public workflow before assigning a runner because the account is locked for a
billing issue. It does not indicate a source or workflow failure, but hosted
execution is not claimed.

## Blocking send-gate status

| Gate | Status | Evidence or blocker |
| --- | --- | --- |
| Fresh sanitized public repository | **PASS** | `banga-agents/twirx-public` is public with one fresh root commit, 644 tracked files, a 643-entry verified manifest, no legacy Git history and zero Gitleaks/TruffleHog findings. |
| Tested engineering branch integrated | **PASS FOR RELEASE CANDIDATE** | The current draft branch is published in PR 17 and complete local release evidence is bound to `95a537a...bea5`; founder merge remains intentionally separate. |
| One authoritative query contract family | **PASS** | ADR 011 retains `tw.semantic-query/0.1` as the sole authoritative family. |
| Three human policy decisions | **PASS** | TWIRX, World Bank and RFC Editor decisions use steward review time `2026-08-11T12:15:50Z` and no broader retrieval authority. |
| Real Common Crawl import | **PASS** | Two exact RFC Editor homepage captures; acquisition manifest `sha256:a28259...b198`. |
| Genuine semantic delta | **PASS WITH SCOPE LABEL** | One source-native origin delta; no semantic/canon delta because no interpretation/canon change occurred. |
| Cross-origin query | **PASS LOCAL** | Four rows across TWIRX and World Bank, zero network requests. |
| Atlas-500 explorer | **PASS ON LOOPBACK** | All 500 selected origin identities are searchable and inspectable; exact packet-bearing state is displayed separately. |
| `lab.twirx.org` HTTPS and read-only Lab | **BLOCKED ON DNS** | Refreshed service is active on `127.0.0.1:8092`, Caddy site validates, but authoritative Namecheap DNS returns no `lab` record. |
| Object Storage and independent restore | **BLOCKED ON ISOLATED CREDENTIALS** | Existing Meridian and Quantlab resources were untouched. A dedicated TWIRX bucket/identity and new encrypted Borg path are required. |
| Website source and reviewer path | **PASS IN SOURCE / DEPLOYMENT PENDING LAB** | Sixteen generated pages pass all checks and link the public repository and Lab; the new release is intentionally not activated while the Lab hostname is dead. |
| Mintlify documentation | **PASS IN SOURCE / EXTERNAL DEPLOYMENT UNVERIFIED** | Release documentation exists; the external live deployment has not been bound to this draft revision. |
| Real versus fixture counters separated | **PASS** | Fifteen public packets and five controlled fixture packets are separate; fixtures are excluded by default. |
| Professional funding request | **PASS IN SOURCE** | The $6,000 first milestone funds full-time maintainer engineering capacity, tools, infrastructure, documentation, security and administration against published outputs. |
| Ambitious 90-day Atlas program | **PASS IN SOURCE** | The program attempts deterministic dossiers and bounded profiles across all 500 selected origins and reports every outcome; evidence-bearing claims remain exact. |
| Secret/privacy/public-readiness scan | **PASS** | Public export and fresh public clone have zero Gitleaks and TruffleHog findings and exclude raw archive bodies. |
| Hosted GitHub CI | **EXTERNAL STARTUP FAILURE** | No runner or step started. Exact GitHub annotation: account locked due to billing issue. No workflow correction is warranted. |
| Final PASS/PASS_WITH_CONDITIONS report | **FAIL** | This report remains FAIL until public Lab and independent restore gates pass. |

## Exact public claim boundary

```text
500 selected Atlas identities, not 500 working adapters
3 completed human policy decisions
1 real archive profile
2 exact historical archive captures
3 public origins with packets
15 public packets
5 controlled fixture packets, excluded by default
1 real origin delta
2 materialized views
1 cross-origin query
75 proof artifacts
0 snapshot-build or runtime origin-network requests
```

All 500 origins are now operationally present in the catalog and reviewer UI.
Only three currently have admitted packets because the founder approved exactly
three bounded policy scopes and explicitly authorized no broader retrieval.
That distinction preserves human control and prevents catalog selection from
being misrepresented as completed compilation.

The next 90-day program has no artificial ceiling: it will generate dossiers
for all 500, complete human decisions as evidence permits, attempt bounded
profiling for every eligible origin, and continue compiling useful sources.
The published delivery floors are 250 usable profiles, 100 observations, 50
native schemas, 25 deterministic adapters, 12 live read-only origins and
100,000 real or visibly provisional packets. Denied, uncertain, constrained
and failed outcomes remain first-class results rather than being hidden.

## Validation and security evidence

Commands executed against the exact release candidate or documented snapshot
source:

```bash
GOMAXPROCS=2 make test
cd web && go run .
node --check snapshotlab/static/app.js
git diff --check

bin/twirx-archive-acquire verify \
  --root atlas/archive-acquisitions/rfc-editor-futo-history

bin/twirx-snapshot verify \
  --snapshot var/futo-public-snapshot-d13c0bf-rebuilt \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5

bin/twirx-snapshot query \
  --snapshot var/futo-public-snapshot-d13c0bf-rebuilt \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5 \
  --file examples/semantic-query-rfc-editor-history.json

bin/twirx-snapshot query \
  --snapshot var/futo-public-snapshot-d13c0bf-rebuilt \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5 \
  --file examples/semantic-query-two-origins.json

scripts/export-public-source.sh "$export_tree" \
  95a537a0a2d54e160a4b67643be2f637f5a7bea5
cd "$export_tree"
sha256sum -c PUBLIC_EXPORT_MANIFEST.sha256
GOMAXPROCS=1 make test
/tmp/twirx-futo-tools/gitleaks dir . --redact --no-banner
/tmp/twirx-futo-tools/trufflehog filesystem . \
  --only-verified --no-update --json

git clone https://github.com/banga-agents/twirx-public.git "$public_clone"
cd "$public_clone"
test "$(git rev-list --count HEAD)" -eq 1
sha256sum -c PUBLIC_EXPORT_MANIFEST.sha256
/tmp/twirx-futo-tools/gitleaks git . \
  --log-opts=--all --redact --no-banner
GIT_ALLOW_PROTOCOL=file /tmp/twirx-futo-tools/trufflehog git \
  "file://$(pwd)" --only-verified --no-update --json
```

Results:

- complete normal Go tests and all Go fuzz smoke targets: PASS;
- GCC and Clang restricted-C builds: PASS;
- ASan and UBSan: PASS;
- Go/C conformance and adversarial vectors: PASS;
- four 5,000-run restricted-C libFuzzer targets: PASS;
- E1/E2 end-to-end and snapshot integration suites: PASS;
- website generator: 16 pages, 420 KiB class, zero third-party requests and
  all claim/link/CSP/accessibility/evidence checks PASS;
- `snapshotlab/static/app.js`: syntax PASS;
- public exported-tree complete suite: PASS with one bounded Go worker;
- public filesystem and fresh-history security scans: zero finding;
- immutable Lab service and 500-origin endpoint: PASS on literal loopback;
- public GitHub Actions: no hosted execution; runner ID zero and no steps.

## Hosted CI evidence

Run URL:
`https://github.com/banga-agents/twirx-public/actions/runs/31507308068`

Job result:

```text
job: genesis
status: completed
conclusion: failure
runner_id: 0
steps: []
annotation: The job was not started because your account is locked due to a billing issue.
```

A CI-only rerun produced the same startup failure. No `ci.yml` change was made
because GitHub did register the intended job and the account lock prevented
runner assignment before workflow execution.

## Exclusions

- Untracked architecture packs and handoff reports were not included in the
  public source, security conclusions or release claims.
- Raw third-party WARC, HTML, index responses and failed acquisition bytes are
  not present in the public repository.
- Local `var/` snapshots, binaries, stress output and scanner artifacts remain
  untracked build evidence.
- Production PostgreSQL, continuous crawling, browser automation, model
  training, payments, authenticated actions and arbitrary URL retrieval are
  not claimed or authorized.
- The existing Meridian Object Storage bucket, Quantlab Storage Box contents,
  RAID and unrelated repositories/services were not accessed or changed.
- Archive packets state what Common Crawl captured historically; they do not
  assert current publisher state or objective truth.

## Unresolved risks

1. `lab.twirx.org` is absent from authoritative DNS, so TLS and public wire
   verification cannot run.
2. No isolated off-host object release and encrypted independent restore has
   passed.
3. The current live main website predates the new Lab/public-repository release;
   deploying the synchronized build before the Lab hostname works would create
   broken primary calls to action.
4. Mintlify's external deployment is not bound to the current draft revision.
5. GitHub-hosted CI cannot start while the account is locked for billing.
6. Public reviewers cannot reconstruct the two archive packets from retained
   raw bytes in the public repository; the capture identifiers and derived
   proof remain available for reacquisition.
7. The runtime benchmark is local and small-corpus; no production capacity is
   inferred.

## Files changed by this release train

- immutable Atlas-500 origin-list and origin-detail runtime endpoints;
- dependency-free Atlas-500 reviewer interface and failure-mode tests;
- exact public packet-state derivation with controlled fixtures excluded;
- professional funding language and ambitious but testable 500-origin program;
- removal of the stale duplicate `site/` source tree;
- FUTO Atlas explorer, public export, Lab staging, off-host durability and
  readiness reports;
- fresh sanitized public repository and one-root-commit publication;
- refreshed isolated Query Lab binary and UI release on Meridian.

## Invariants preserved

- Genesis remains read-only;
- no arbitrary URL, scheduler, browser, model, payment or write authority;
- no runtime dependency added;
- no E1/E2 behavior or conformance weakened;
- source-native lexical values and derivation remain preserved;
- missing required evidence fails closed;
- 500 catalog identities are never described as 500 working adapters;
- no unrelated Meridian data, credential, repository or service touched.

## Deviations

- Public Lab activation and website deployment are deferred because the
  authoritative DNS prerequisite is absent.
- Off-host publication/restore is deferred because only existing unrelated
  storage resources were identified; no isolated TWIRX credentials were
  provided.
- Hosted CI did not execute because of an account-level GitHub billing lock.
  The workflow was neither broadened nor changed.

## Next recommended gate

Publish and verify the `lab` CNAME, activate the validated Caddy site, complete
public browser/wire checks and atomically deploy the synchronized website.
In parallel, create a dedicated TWIRX Object Storage identity and new encrypted
Borg repository path, then execute a byte-identical restore drill. Advance this
report to PASS or PASS WITH CONDITIONS only after both operational proofs are
recorded.
