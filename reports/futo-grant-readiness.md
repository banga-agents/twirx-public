# FUTO final send-gate audit

**Recommendation: PASS — the application is ready to send**

**Audited at:** `2026-08-12T07:29:39Z`

**Integrated private source revision before this report:**
`15aba7b7eb373fe954ccd05b0e77a7de29a741ae`

**Sanitized public main before publishing this report:**
`e56d40cfd1daea60ed056b84e17871473137ce9c`

**Versioned E4.5 evidence release:**
[`v0.4.5-rc.1`](https://github.com/banga-agents/twirx-public/releases/tag/v0.4.5-rc.1),
targeting `f531eb5d359041027d4e4302ea31438a65d488bf`

**Live Query Lab snapshot:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`

## Executive result

The final public-evidence gate passes. TWIRX now presents one consistent,
reviewable path from its mission to working software, exact limitations and a
milestone-bound funding request:

- the sanitized public repository is live and current;
- the E4.5 Opportunity Utility evidence is published as a versioned prerelease;
- the live immutable Query Lab remains available and read-only;
- the website and Mintlify documentation present the Semantic Data Plane and
  Agent Utility Universe program consistently;
- the revised six-page grant application is downloadable from the release;
- public links, page rendering, security headers and runtime behavior were
  checked from the public network;
- the complete local release suite passed on the final implementation lineage;
- final public history contains no verified secret.

The application does not claim that the large E4.5 candidate is the currently
deployed Lab runtime. The public Lab serves the smaller admitted snapshot. The
1.73 GB E4.5 release remains published engineering evidence pending separate
target-host admission.

## Send-gate status

| Gate | Result | Evidence |
| --- | --- | --- |
| Founder review of E4.5 | **PASS** | PR #22 was admitted and merged as `71770df...`; the public admission report records the decision and exact residual limits. |
| Versioned public evidence | **PASS** | Public prerelease `v0.4.5-rc.1` exposes the manifest, admission report and grant PDF. |
| Sanitized public source | **PASS** | `banga-agents/twirx-public` is public; the audited pre-report main is `e56d40c...`; each deterministic export identifies its exact private source revision. |
| E4.5 quantitative evidence | **PASS WITH SCOPE LABELS** | 83,087 accepted real source records and provisional frames, 1,037,679 source-derived packets, 747,783 candidate mappings, 31/31 proof-linked investigations and 3.107 ms local warm-query p95. |
| Live Query Lab | **PASS** | HTTPS runtime serves immutable snapshot `547398...`, 500 selected identities, 15 public packets across three origins, two views, one genuine historical origin delta and a four-row cross-origin query with zero network requests. |
| Website | **PASS** | `twirx.org`, `/futo/`, `/funding/`, `/roadmap/` and `/manifesto/` render correctly from the active immutable static release. |
| Documentation | **PASS** | `docs.twirx.org` uses TWIRX branding and exposes the FUTO release, roadmap, funding and Opportunity Utility documentation. |
| Application | **PASS** | Six-page revised PDF is public; maintainer compensation is described professionally as dedicated full-time engineering capacity. |
| Security scan | **PASS** | Gitleaks reports zero findings across the final seven-commit public history and current tree. TruffleHog reports zero verified secrets; its five unverified URI matches are intentional fake embedded-credential literals in adversarial `_test.go` cases. |
| VPS boundary | **PASS** | Static website and Query Lab are active; only ports 80 and 443 are externally reachable. Port 8088 and the Lab's internal 8091 listener are externally unreachable. The Lab service is unprivileged and network-restricted to loopback. |
| Off-host durability | **PASS FOR SEND GATE** | The independent encrypted archive passed data verification and a byte-identical restore. A second dedicated versioned Object Storage copy remains follow-up work, not a send blocker. |
| Hosted GitHub Actions | **DISCLOSED EXTERNAL LIMITATION** | The workflow registers its intended `genesis` job, but GitHub creates no steps or runner because of the account-level startup lock. Hosted execution is not claimed and is not a release blocker. |

## Exact public claim boundary

### Deployed immutable Lab

```text
500 selected Atlas identities, not 500 working adapters
3 public origins with packets
15 real public packets
5 controlled fixture packets, excluded by default
1 genuine historical origin delta
2 materialized views
1 four-row cross-origin query
0 query-time origin requests
0 browser or model execution authority
```

### Published E4.5 engineering candidate

```text
1 primary source family
83,133 source records seen
83,087 source records accepted
46 source records rejected
1,037,679 real source-derived packets
747,783 candidate mapping claims
83,087 provisional Opportunity frames
35 World State frames
83,122 combined frames
31/31 proof-linked investigations
3.107 ms local warm-query p95 across 10,000 iterations
0 runtime origin calls
0 runtime browser executions
0 model authority
```

The E4.5 mappings are candidates, not canon, and frames remain in the
`provisional_semantic` trust lane. Contact-like fields and descriptions are
excluded from the public projection; eligibility lexical content is withheld.
TWIRX does not infer current availability, applicant eligibility, currency,
relevance or rank.

## Release artifacts

| Artifact | SHA-256 |
| --- | --- |
| `release-manifest.json` | `cbaaa1cd2b41f698f7b423a516727f5a7907bba56ac6c17136528f40f45d7690` |
| `TWIRX_FUTO_GRANT_APPLICATION_V2.pdf` | `87368e4348c2bfd95694ee34232f3ffc747d0475671cab9bf030058e8bfddebd` |

The downloaded public assets matched these digests byte for byte.

## Commands executed

Implementation and public-source validation:

```bash
make test FUZZ_WORKERS=1
(cd web && go run . && go test ./...)
make docs-check
git diff --check
jq empty web/data/*.json web/site.json docs/docs.json

scripts/export-public-source.sh "$export_tree" \
  15aba7b7eb373fe954ccd05b0e77a7de29a741ae
(cd "$export_tree" && sha256sum -c PUBLIC_EXPORT_MANIFEST.sha256)
make test FUZZ_WORKERS=1

gitleaks git "$public_clone" --redact --no-banner
gitleaks dir "$public_clone" --redact --no-banner
trufflehog git https://github.com/banga-agents/twirx-public.git \
  --json --no-update
```

Public surface and runtime verification:

```bash
curl -fsS https://twirx.org/
curl -fsS https://twirx.org/futo/
curl -fsS https://twirx.org/funding/
curl -fsS https://twirx.org/roadmap/
curl -fsS https://twirx.org/manifesto/

curl -fsS https://docs.twirx.org/
curl -fsS https://docs.twirx.org/start/futo-release
curl -fsS https://docs.twirx.org/governance/roadmap
curl -fsS https://docs.twirx.org/governance/funding-transparency
curl -fsS https://docs.twirx.org/protocol/opportunity-utility-universe

curl -fsS https://lab.twirx.org/api/v1/status
curl -fsS -H 'content-type: application/json' \
  --data-binary @examples/semantic-query-two-origins.json \
  https://lab.twirx.org/api/v1/query

chromium --headless --disable-gpu --no-sandbox --hide-scrollbars \
  --window-size=1440,1400 --screenshot=... https://twirx.org/
chromium --headless --disable-gpu --no-sandbox --hide-scrollbars \
  --window-size=1440,1800 --screenshot=... https://twirx.org/futo/
chromium --headless --disable-gpu --no-sandbox --hide-scrollbars \
  --window-size=1440,1400 --virtual-time-budget=8000 \
  --screenshot=... https://lab.twirx.org/
chromium --headless --disable-gpu --no-sandbox --hide-scrollbars \
  --window-size=1440,1400 --screenshot=... https://docs.twirx.org/

gh repo view banga-agents/twirx-public \
  --json url,visibility,defaultBranchRef
gh release view v0.4.5-rc.1 --repo banga-agents/twirx-public \
  --json url,isPrerelease,targetCommitish,assets
```

Target-host checks were read-only and limited to TWIRX units, ports and release
paths. The active static release is
`/srv/twirx/site/releases/20260812T071543Z-f531eb5`. Caddy and
`twirx-snapshot-lab.service` are active. The Lab unit runs as the dedicated
`twirx-snapshot` user with `NoNewPrivileges=yes`, `ProtectSystem=strict`,
`ProtectHome=yes`, a 256 MiB memory limit, a 25% CPU quota, 32-task limit and
loopback-only systemd IP policy.

## Validation results

- normal Go tests: PASS;
- all Go fuzz targets at the admitted three-second budget with one worker:
  PASS;
- GCC and Clang restricted-C builds: PASS;
- ASan and UBSan: PASS;
- shared Go/C valid and adversarial conformance: PASS;
- restricted-C E4 deterministic sample verification: PASS;
- both restricted-C libFuzzer harnesses, 5,000 runs each: PASS;
- E1/E2 end-to-end and Semantic Snapshot integration: PASS;
- documentation checks: PASS;
- website generation and tests: PASS, 16 static pages and no third-party
  request dependency;
- public export manifest: PASS;
- public-history Gitleaks: zero findings;
- public-history TruffleHog: zero verified secrets; five known fake URI
  literals in adversarial tests;
- browser rendering: PASS for homepage, FUTO path, Lab and documentation;
- public Query Lab: resolved four-row query, 20 packets scanned, four matched,
  four returned, five fixtures excluded and zero network requests.

An initial full-suite invocation did not actually honor the intended one-worker
limit because the Makefile's fuzz worker setting remained at four. One target
ended during Go fuzz shutdown with `context deadline exceeded`, after 322,981
executions, without a crash or saved failing input. The exact target passed on
rerun, and the complete suite then passed with `FUZZ_WORKERS=1`. No failure was
suppressed and no test was weakened.

## Public links verified

- `https://twirx.org/`
- `https://twirx.org/futo/`
- `https://twirx.org/manifesto/`
- `https://lab.twirx.org/`
- `https://docs.twirx.org/`
- `https://github.com/banga-agents/twirx-public`
- `https://github.com/banga-agents/twirx-public/releases/tag/v0.4.5-rc.1`

All returned HTTP 200. The `www` hostname returns the intended permanent
redirect to the apex. The website sends CSP, HSTS, `nosniff`, no-referrer,
COOP, CORP and Permissions Policy headers, exposes no cookies, and denies
repository and deployment-internal paths.

## Files changed in the final public-evidence gate

- `CHARTER.md`
- `MANIFESTO.md`
- `README.md`
- `PUBLIC_EVIDENCE_PROFILE.md`
- `scripts/export-public-source.sh`
- `reports/e4-5-opportunity-admission.md`
- `reports/futo-grant-readiness.md`
- `docs/docs.json`
- `docs/index.mdx`
- `docs/start/futo-release.mdx`
- `docs/start/genesis-status.mdx`
- `docs/protocol/opportunity-utility-universe.mdx`
- `docs/governance/roadmap.mdx`
- `docs/governance/funding-transparency.mdx`
- `web/data/e4-utility-release.json`
- `web/data/futo-release.json`
- `web/data/project-status.json`
- `web/pages/index.html`
- `web/pages/futo.html`
- `web/pages/funding.html`
- `web/pages/roadmap.html`
- `web/pages/manifesto.html`
- `web/site.json`

The final evidence gate changed public narrative, evidence metadata, export
hygiene and documentation. It did not change trusted implementation behavior.

## Invariants preserved

- source-native terms and lexical values remain preserved before mapping;
- required evidence still fails closed;
- optional missing content remains explicitly unresolved;
- mappings and model-like outputs cannot promote themselves into canon;
- Genesis remains read-only;
- no arbitrary URL, browser, payment, authentication, write or action
  capability was introduced;
- no runtime dependency was added;
- no E1, E2, E3 or E4 validation behavior was weakened;
- controlled fixtures are not counted as real public packets;
- 500 selected identities are not described as 500 compiled origins;
- the public Lab and the large E4.5 candidate remain visibly separate;
- no unrelated Meridian repository, service, storage, RAID or data was
  accessed or changed.

## Exclusions and unresolved risks

1. The 1.73 GB E4.5 runtime has not passed target-host admission and is not
   deployed. The live Lab intentionally remains on the smaller admitted
   immutable snapshot.
2. The candidate covers one substantial Opportunity source family, not 500
   compiled origins or all five proposed universes.
3. The restricted-C verifier checks a deterministic E4 sample rather than the
   entire million-packet corpus.
4. Continuous refresh is disabled. The one-shot acquisition authority is
   consumed, and a new explicit policy decision is required before retrieval.
5. The benchmark is local and scoped; it is not a universal performance claim
   or target-host capacity admission.
6. A dedicated second versioned Object Storage copy remains to be created.
   The independent encrypted archive and byte-identical restore already pass.
7. GitHub-hosted CI still stops before assigning a runner or creating steps.
   Complete local execution is the claimed test evidence.
8. Five fake embedded-credential URLs intentionally remain in adversarial
   tests. They are not live credentials and all occur in `_test.go` files.
9. Earlier public history contains non-secret identifiers for unrelated
   infrastructure in a superseded operations report. The current public tree
   excludes that report; Gitleaks and TruffleHog found no verified secret.

These risks remain disclosed but do not block sending the application. None
changes the truth of the public working proof or the E4.5 release boundary.

## Deviations

- The full E4.5 corpus was not deployed to Meridian. This is intentional: the
  final-send instruction explicitly permits the smaller admitted Lab snapshot
  to remain live until the large release passes separate host admission.
- A second Object Storage replica was not created because no dedicated TWIRX
  bucket and least-privilege identity has yet been admitted. Unrelated storage
  was left untouched.
- Hosted CI did not execute because the external account-level startup block
  remains. No workflow was changed or weakened.
- No email was sent by the engineering agent. Sending the external grant
  request remains a founder action.

## Next recommended gate

Send the FUTO application now. Then begin E4 Agent Utility Universe Alpha on a
separate branch while preserving `v0.4.5-rc.1` and the live Lab as protected
baselines. The first engineering subgate should admit the segmented Universe
Snapshot on measured target-host limits before any larger deployment, followed
by World State and Opportunity ontology modules and the Visual Atlas Agent.
