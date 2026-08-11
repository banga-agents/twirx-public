# FUTO grant-readiness audit

**Recommendation: PASS_WITH_CONDITIONS — the public proof is live; three
operational limitations must remain disclosed**

**Audit date:** 2026-08-11

**Complete validation revision:**
`2c85b56322d381409ca11001eacec8a8111d251d`

**Fresh public repository:**
`https://github.com/banga-agents/twirx-public`

**Validated public source release:**
`4b9e138068d723bcea3db6fc9fa7e6d9cdb5850e`

**Immutable snapshot:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`

**Active Query Lab releases:**
`20260811T153300Z-95a537a`

## Executive result

TWIRX now has a clean public repository, a verified evidence kernel,
independent Go/C verification, three explicit human policy decisions, one
sealed two-period Common Crawl acquisition, two source-native archive packets,
one genuine origin delta, a 500-identity immutable snapshot, a real
cross-origin query, a read-only snapshot runtime and a searchable Atlas-500
Lab interface. The complete local release suite passes.

The bounded proof needed for a serious grant review is now public. The fresh
repository is reachable, `https://lab.twirx.org/` serves the exact admitted
immutable snapshot through TLS, all 500 selected Atlas identities can be
inspected, a real cross-origin query and genuine historical delta are
available, and complete local validation passes.

This recommendation is conditional rather than unconditional because the
versioned Object Storage replica remains pending, the live Mintlify deployment
does not yet expose the release-specific FUTO page, and GitHub-hosted CI cannot
start while the account is locked for billing. An isolated encrypted Storage
Box archive has passed full data checking and a byte-identical restore. The
remaining conditions do not erase the public mechanism or local evidence; they
must be disclosed and resolved as operational follow-up work.

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
| Fresh sanitized public repository | **PASS** | `banga-agents/twirx-public` is public with fresh sanitized history, a deterministic export manifest, no copied private review history and zero Gitleaks/TruffleHog findings. |
| Tested engineering branch integrated | **PASS FOR RELEASE CANDIDATE** | The current draft branch is published in PR 17 and complete local release evidence is bound to `2c85b56...d251d`; founder merge remains intentionally separate. |
| One authoritative query contract family | **PASS** | ADR 011 retains `tw.semantic-query/0.1` as the sole authoritative family. |
| Three human policy decisions | **PASS** | TWIRX, World Bank and RFC Editor decisions use steward review time `2026-08-11T12:15:50Z` and no broader retrieval authority. |
| Real Common Crawl import | **PASS** | Two exact RFC Editor homepage captures; acquisition manifest `sha256:a28259...b198`. |
| Genuine semantic delta | **PASS WITH SCOPE LABEL** | One source-native origin delta; no semantic/canon delta because no interpretation/canon change occurred. |
| Cross-origin query | **PASS PUBLIC** | Four rows across TWIRX and World Bank, five fixtures excluded and zero network requests at the public endpoint. |
| Atlas-500 explorer | **PASS PUBLIC** | All 500 selected origin identities are searchable and inspectable; exact packet-bearing state is displayed separately. |
| `lab.twirx.org` HTTPS and read-only Lab | **PASS** | Both authoritative nameservers resolve the host; TLS, API, browser, headers, input/method rejection, raw-proof denial and zero-origin-call behavior pass. Runtime remains literal loopback. |
| Independent encrypted restore | **PASS** | A new isolated Borg repository on the Storage Box passed `--verify-data`, byte-identical 85-file restore and exact snapshot verification. Existing archive content was neither listed nor touched. |
| Versioned Object Storage release | **CONDITION OPEN** | The existing `meridianv2raw` bucket and credentials remain untouched; a dedicated TWIRX bucket and least-privilege identity are still required. |
| Website source and reviewer path | **PASS DEPLOYED** | Sixteen generated pages pass all checks and are active in immutable release `20260811T163300Z-c0ab594`; all routes and public safety denials pass. |
| Mintlify documentation | **CONDITION OPEN** | Release documentation exists in source; `https://docs.twirx.org/start/futo-release` currently returns `404`, so external synchronization is not claimed. |
| Real versus fixture counters separated | **PASS** | Fifteen public packets and five controlled fixture packets are separate; fixtures are excluded by default. |
| Professional funding request | **PASS IN SOURCE** | The $6,000 first milestone funds full-time maintainer engineering capacity, tools, infrastructure, documentation, security and administration against published outputs. |
| Ambitious 90-day Atlas program | **PASS IN SOURCE** | The program attempts deterministic dossiers and bounded profiles across all 500 selected origins and reports every outcome; evidence-bearing claims remain exact. |
| Secret/privacy/public-readiness scan | **PASS** | Public export and fresh public clone have zero Gitleaks and TruffleHog findings and exclude raw archive bodies. |
| Hosted GitHub CI | **EXTERNAL STARTUP FAILURE** | No runner or step started. Exact GitHub annotation: account locked due to billing issue. No workflow correction is warranted. |
| Final PASS/PASS_WITH_CONDITIONS report | **PASS_WITH_CONDITIONS** | The public proof and independent encrypted restore are reviewable now; Object Storage, live Mintlify sync and hosted-CI startup remain explicitly disclosed. |

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
FUZZ_WORKERS=1 GOMAXPROCS=1 make test
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
- immutable website release: all 31 files verified byte-for-byte before the
  atomic release switch; all 16 public routes, headers, redirects, browser
  rendering and private-path denials PASS;
- public GitHub Actions: no hosted execution; runner ID zero and no steps.

## Hosted CI evidence

Public-release run URL:
`https://github.com/banga-agents/twirx-public/actions/runs/31513337632`

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

1. The encrypted Storage Box restore passes, but the second versioned Object
   Storage release is not yet configured. The local Borg passphrase and key
   export also require an additional operator-controlled offline copy.
2. Mintlify's external deployment is not bound to the current draft revision;
   the release documentation remains available in the public source tree.
3. GitHub-hosted CI cannot start while the account is locked for billing.
4. Public reviewers cannot reconstruct the two archive packets from retained
   raw bytes in the public repository; the capture identifiers and derived
   proof remain available for reacquisition.
5. The runtime benchmark is local and small-corpus; no production capacity is
   inferred.
6. The public runtime serves an immutable snapshot only and performs no fresh
   origin retrieval, continuous compilation, browser execution or writes.

## Files changed by this release train

- public HTTPS activation of the immutable Query Lab;
- immutable Atlas-500 origin-list and origin-detail runtime endpoints;
- dependency-free Atlas-500 reviewer interface and failure-mode tests;
- exact public packet-state derivation with controlled fixtures excluded;
- professional funding language and ambitious but testable 500-origin program;
- removal of the stale duplicate `site/` source tree;
- FUTO Atlas explorer, public export, Lab staging, off-host durability and
  readiness reports;
- fresh sanitized public repository and deterministic export publication;
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

- The planned Object Storage replica is deferred because the only identified
  bucket is unrelated `meridianv2raw`; it remains untouched. The separate
  encrypted Storage Box restore is complete.
- Mintlify source is synchronized, but the external production deployment has
  not published the release-specific route.
- Hosted CI did not execute because of an account-level GitHub billing lock.
  The workflow was neither broadened nor changed.

## Next recommended gate

Create a dedicated TWIRX Object Storage identity and versioned bucket, preserve
the Borg recovery material through a second offline channel, and synchronize
the live Mintlify release. Advance this report from PASS_WITH_CONDITIONS to
PASS only after those operational proofs are recorded.
