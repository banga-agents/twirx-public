# FUTO grant-readiness audit

**Recommendation: FAIL — do not send the grant request yet**

**Audit date:** 2026-08-11

**Validated engineering revision:**
`bdee40c52bbd2be137cfadc577bac6098c799968`

**Website evidence revision:**
`ded2b190f722e1e0ab661817926a692994ebd9a1`

**Website deployment revision:**
`b1f098f77e2cc8df273f5a9027c6845256007102`

**Sanitized public-export revision:**
`2c6fd8f4adfab9e11dcd3268ddfe71c910805919`

**Active website release:**
`20260811T135847Z-b40b0540f5f8` with normalized content-manifest SHA-256
`022cfeb5533131a7f826606e893af2cef4beaa704fb2456c4e2bd165f5105c3d`

**Immutable snapshot:**
`sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5`
from source revision `d13c0bfaf4aed19b9db541b3fd7ebc97e251ebc3`

## Executive result

The largest missing proof is no longer technical interpretation. TWIRX now has
three human policy decisions, one real two-period Common Crawl acquisition,
two source-native archive packets, one genuine origin delta, a 500-identity
immutable snapshot, a real cross-origin query and a verified read-only runtime.
The complete local suite passes.

The release is still not ready for FUTO because the reviewer cannot access the
source and runtime through a clean public repository and HTTPS Lab, off-host
durability has not been demonstrated, Mintlify is not synchronized to the
release, and the fresh public export has not yet applied the approved-safe
raw-evidence exclusion profile. The main website is now release-bound,
deployed and wire-verified; that blocker is closed. The Query Lab is installed,
isolated and validated on literal loopback, but it is not publicly activated
until `lab.twirx.org` resolves. The remaining items are blocking release tasks,
not reasons to weaken or inflate the engineering claims.

## Blocking send-gate status

| Gate | Status | Evidence or blocker |
| --- | --- | --- |
| Fresh sanitized public repository | **EXPORT PASS / PUBLICATION BLOCKED** | A 644-entry raw-evidence-free export from `2c6fd8f...5919` passes the complete public test suite and both secret scanners. Creating the new public GitHub repository is not yet authorized. |
| Tested engineering branch integrated | **PASS FOR F0** | PR 16 was merged to `main` as `fbf1d6a`; the follow-up policy/archive/report work remains in draft PR 17. |
| PR 15 reconciliation and one contract family | **PASS** | ADR 011 retains `tw.semantic-query/0.1` as the sole authoritative query family. |
| Website source tracked and release-bound | **PASS** | The dependency-free generator and all 16 pages are tracked in PR 17. Public counters bind to website evidence revision `ded2b190...d9a1` and snapshot `sha256:547398...f1389c5`. |
| Mintlify docs synchronized | **SOURCE PASS / DEPLOYMENT BLOCKED ON DRAFT** | Revision `633f34a` adds the exact FUTO release and snapshot-runtime pages and removes stale counters/funding language. Mintlify skipped the draft deployment; merge/preview and live wire verification are pending. |
| `lab.twirx.org` HTTPS and read-only Lab | **STAGED / BLOCKED ON DNS** | The dedicated read-only service is active on `127.0.0.1:8092`; its Caddy site validates but is not imported. `lab.twirx.org` does not yet resolve. See `reports/futo-query-lab-staging.md`. |
| Three human policy decisions | **PASS** | TWIRX, World Bank and RFC Editor decisions use steward review time `2026-08-11T12:15:50Z`. |
| Real Common Crawl import | **PASS** | Two exact RFC Editor homepage captures; acquisition manifest `sha256:a28259...b198`. |
| Genuine semantic delta | **PASS WITH SCOPE LABEL** | One source-native origin delta; no semantic/canon delta because no interpretation/canon change occurred. |
| Cross-origin query | **PASS LOCAL** | Four rows across TWIRX and World Bank; zero network requests. |
| Static proof and reviewer path | **PASS WITH SCOPE LABEL** | `https://twirx.org/demo/`, `/proof/` and `/futo/` are live, but the downloadable snapshot/proof bundle and live query API are not yet public. |
| Object Storage and independent restore | **BLOCKED** | No upload, encrypted backup or clean byte-identical restore evidence. |
| Real versus fixture counters separated | **PASS LOCAL** | 15 public packets and five controlled fixture packets are separate; fixtures are excluded by default. |
| One coherent funding ask | **PASS** | The public surface consistently requests $30,000 for 90 days, beginning with a $6,000/30-day milestone and its exact six-line allocation. |
| Public contact and mail DNS | **PASS WITH LIMITATION** | Two Proton MX, SPF, DMARC and three DKIM records remain present; `rick@twirx.org` is published. End-to-end inbound delivery was not performed in this gate. |
| Secret/privacy/public-readiness scan | **PASS FOR SANITIZED EXPORT** | History has no verified secret. The audited public export excludes raw WARC/HTML, passes its manifest and full public suite, and has zero Gitleaks or TruffleHog finding. See `reports/futo-public-source-export.md`. |
| Final PASS/PASS_WITH_CONDITIONS report | **FAIL** | This report intentionally remains FAIL until every blocking item above is resolved. |

## Publicly claimable local evidence

Subject to resolving redistribution and publication:

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

The controlled 25,018-packet result remains capacity evidence only and is not
part of this 20-packet real/public snapshot.

## Validation and security evidence

Commands executed against the exact validation revision or its documented
snapshot source revision:

```bash
make test
GOMAXPROCS=2 go test -race ./...
GOMAXPROCS=2 go vet ./...
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

/tmp/twirx-futo-tools/gitleaks git . \
  --log-opts=--all --redact --no-banner

GIT_ALLOW_PROTOCOL=file /tmp/twirx-futo-tools/trufflehog git \
  "file://$(pwd)" --only-verified --no-update --json

scan_root=$(mktemp -d /tmp/twirx-futo-tree.XXXXXX)
git archive HEAD | tar -x -C "$scan_root"
/tmp/twirx-futo-tools/gitleaks dir "$scan_root" --redact --no-banner
GIT_ALLOW_PROTOCOL=file /tmp/twirx-futo-tools/trufflehog filesystem \
  "$scan_root" --only-verified --no-update --json

(cd web && GOMAXPROCS=2 go test ./...)
(cd web && GOMAXPROCS=2 go vet ./...)
(cd web && make check)

rsync -a --delete --chmod=D750,F640 web/dist/ \
  agent:/srv/twirx/site/releases/20260811T135847Z-b40b0540f5f8/
sudo chgrp -R caddy \
  /srv/twirx/site/releases/20260811T135847Z-b40b0540f5f8
sudo find /srv/twirx/site/releases/20260811T135847Z-b40b0540f5f8 \
  -type d -exec chmod 2750 {} +
sudo find /srv/twirx/site/releases/20260811T135847Z-b40b0540f5f8 \
  -type f -exec chmod 0640 {} +
```

Results:

- complete offline `make test`: PASS after bounding short Go fuzz concurrency
  to four workers;
- race detector, vet and whitespace validation: PASS;
- restricted-C snapshot, all 20 packet cores and one delta core: PASS;
- 5,000-request loopback stress: 5,000 success, zero failure, zero origin call;
- Gitleaks history: 61 commits, zero finding;
- TruffleHog history and tracked tree: zero verified or unverified finding;
- Gitleaks tracked tree: two detections on one retained source-representation
  line, requiring the redistribution treatment described below.
- website generator: 16 pages and 31 static files, all claim, link, CSP,
  metadata, JavaScript-budget and evidence-chain checks passed;
- deployed bytes: local and VPS normalized release manifests match exactly;
- production wire: all 16 pages, six selected assets/data files and redirects
  passed; `_headers`, `_redirects`, `.git` and repository reports are
  unreachable; the live homepage is byte-identical to the local build;
- browser render: home, demo and FUTO pages rendered in headless Chromium with
  no CSP, resource-load or uncaught-script error in the browser logs;
- Proton mail DNS and the Mintlify CNAME remain unchanged; docs return HTTP 200.
- the immutable Query Lab service is active and enabled on literal loopback,
  its Caddy site validates, preset queries and fail-closed URL rejection pass,
  and the public port is closed; public HTTPS is not yet claimed.
- the deterministic sanitized source export has 644 manifest entries, excludes
  all named raw archive files, passes the complete public test/fuzz suite and
  has zero Gitleaks or TruffleHog finding.

## Exclusions

- Untracked architecture packs and handoff reports were not staged, altered or
  included in the security conclusions. The complete `web/` source is now
  tracked and release-bound.
- Local `var/` snapshots, stress results, binaries and temporary scanner output
  are build evidence, not tracked release artifacts.
- Hosted GitHub CI, a public Query Lab, Object Storage durability, Storage Box
  restore and external reviewer reproduction are not claimed. The static
  website deployment is claimed and separately wire-verified.
- The archive packets describe what Common Crawl captured historically, not
  what the RFC Editor currently represents and not objective truth.

## Unresolved risks

1. The exact retained 2026 source representation contains a browser-side
   Typesense client value detected by Gitleaks. It is not a TWIRX credential.
   The conservative publication treatment excludes raw WARC/HTML and publishes
   derived proof and reproduction metadata; the sanitized export now enforces
   and verifies this boundary, but the new public repository remains uncreated.
2. Draft PR 17 remains unmerged, and the fresh sanitized public repository
   does not exist.
3. Proof durability has not been demonstrated through versioned Object Storage,
   encrypted independent backup and clean restore.
4. The generated website counters and FUTO reviewer path are release-bound and
   live. The Query Lab is privately staged but public DNS/HTTPS verification
   remains absent; Mintlify source is synchronized but external deployment is
   not yet verified.
5. The delta batch-ID topology uses the already-published acquisition manifest
   for this profile and still requires a separate normative resolution.
6. The runtime benchmark is local and small-corpus; no public or production
   capacity is inferred.

## Files changed by this release train

- policy decisions/evidence and canonical Atlas migration for three origins;
- sealed RFC Editor archive work order and immutable acquisition evidence;
- restricted WARC compatibility fix, native archive profile and adversarial
  tests;
- source-native packet, origin-delta, proof and immutable snapshot integration;
- query/delta runtime exposure and exact query fixture;
- deterministic build/test prerequisites and bounded fuzz concurrency;
- FUTO archive-evidence and readiness reports.
- dependency-free 16-page website generator, generated evidence data, working
  demo, Atlas, architecture, funding, roadmap and dedicated FUTO reviewer path;
- immutable static release deployment with strict CSP and byte-identical wire
  verification.
- immutable Query Lab UI, hardened loopback-only systemd service, separated
  Caddy/runtime roots and private staging evidence.

## Deviations

- No fresh public repository was created because repository publication and
  visibility were not authorized.
- No Object Storage, Storage Box, DNS or Mintlify content was changed. A
  stateless read-only service was installed on Meridian within the approved
  constraints; no PostgreSQL, mutable semantic state or origin retrieval was
  added.
- Caddy configuration was not changed or reloaded. Only the existing static
  site's atomic release symlink was updated. An initial switch exposed a new
  release with the wrong group permissions; it was rolled back immediately,
  permissions were corrected and verified as the Caddy user while inactive,
  and the second atomic switch passed all wire checks.
- No architecture beyond the bounded FUTO release train was added.

## Next recommended gate

Add the `lab` DNS record, activate the already-validated Caddy site and complete
public wire verification. Then publish the exact snapshot to versioned off-host
storage, prove an encrypted independent restore, synchronize Mintlify and
create a fresh sanitized public repository whose export excludes the private
raw archive bodies. The FUTO request may be sent only when this report advances
to PASS or PASS WITH CONDITIONS.
