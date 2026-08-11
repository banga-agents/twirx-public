# Public-readiness audit after history privacy remediation

**Recommendation:** **FAIL — keep the GitHub repository private**

**Audit date:** 2026-08-10

**Audited commit:** `9111aa31a53e2fb2f733a8ffdb43db47ae1950e2`

**Audited tree:** `9c599bb590ca09215dbcf6029707174c23bf2f28`

The reachable Git branch history, candidate tree, funding files, and complete
Gate E1 validation pass the public-readiness checks. One GitHub-hosted metadata
surface remains release-blocking: merged PR #1 retains its pre-remediation
merge object identifier in the GitHub API even though that object is no longer
reachable from a branch or advertised pull-request ref. Because the repository
is still private, this metadata has not been intentionally published. Resolve
that server-side retention through repository recreation or confirmed GitHub
purging before changing visibility.

No code, schema, fixture, protocol, test, workflow, wallet, or implementation
behavior changed during remediation or this audit.

## Privacy remediation

The founder confirmed that the former commit address was not intended to be
public and supplied the verified GitHub noreply replacement. The removed value
is not repeated in this report.

The rewrite used official `git-filter-repo` release 2.47.0, revision
`a40bce548d2c`. Its private mapping rule replaced the single matching author or
committer email value with
`210024963+banga-agents@users.noreply.github.com`; it did not transform file
content, names, messages, dates, or any other metadata.

Before rewriting, two permission-restricted recovery artifacts were created
outside the repository:

- a complete Git bundle containing every pre-rewrite ref;
- a working-tree archive containing all uncommitted website, documentation,
  report, and planning-pack files.

Their paths, digests, and the complete old-to-new commit map are retained
privately and are intentionally not committed to this public report.

Dry-run and post-rewrite comparison results:

- seven commits compared;
- one private-email metadata occurrence replaced;
- four commit identifiers changed because the affected merge and its three
  descendants were rewritten;
- three unaffected commit identifiers remained identical;
- every file tree, commit message, parent relationship, author/committer name,
  timestamp, and timezone matched after applying the commit map.

Repository-local Git identity now uses the verified noreply address. A local
pre-push hook rejects author or committer addresses outside the approved
noreply identities. This hook is local protection and is not a substitute for
GitHub's account-level email privacy and push-blocking settings.

## Scope

The tracked audit covered:

- 130 files at the audited commit;
- all eight commits reachable from local and remote-tracking branch refs;
- 253 `git rev-list --objects --all` entries, including 153 unique blobs;
- 131 unique historical paths;
- author, committer, subject, parent, ref, and tree metadata;
- every current and historical funding file;
- current GitHub branch refs and advertised pull-request refs.

The only historical-only path is `.github/workflows/ci.yaml`, the earlier
byte-preserving workflow-registration rename. There are no submodules or Git
LFS entries.

Uncommitted website, documentation, reports, and phase packs were preserved
and scanned separately. At the final snapshot they comprised 83 files and
843,599 bytes, including generated `web/dist/` output. They are not part of the
audited commit or reachable history.

## Secret and privacy findings

No private key, seed or recovery phrase, API key, access token, password,
cookie, bearer credential, wallet private key, credential file, identity
document, residence address, personal phone number, bank/card record, or
operational private endpoint was found in the rewritten reachable history.

- Gitleaks 8.30.1: zero findings in rewritten reachable history.
- TruffleHog 3.96.0: zero verified secrets and one unverified URI heuristic.
  The heuristic is the synthetic embedded-credential negative test in
  `internal/safefetch/safefetch_test.go`, which verifies URL credentials are
  rejected.
- The all-blob scanner found zero BIP-39 sequences, private-key formats,
  common provider tokens, credential assignments, risky historical paths, or
  binary blobs.
- The rewritten reachable metadata contains zero occurrences of the removed
  email fingerprint and zero non-noreply author/committer addresses.
- Gitleaks and TruffleHog returned zero findings for each separately scanned
  uncommitted area: `TWIRX_GATE_2_PUBLIC_LAUNCH_PACK/`,
  `TWIRX_NEXT_PHASE_PACK/`, `web/`, and the uncommitted reports.

Loopback, RFC 1918, link-local, carrier-grade NAT, and ULA addresses occur only
in explicit local fixtures, rejection tests, deny lists, and generated proof
examples. No internal service endpoint or cloud metadata endpoint was found.

## Funding wallets and tracked hygiene

- `funding/wallets.json` declares an empty wallet list and pending status.
- `funding/wallets.example.json` contains only
  `0xPUBLIC_ADDRESS_ONLY`.
- The uncommitted Gate P1 wallet template contains replacement placeholders
  only.
- The funding ledger contains only its header; no real receipt, expense,
  wallet, payment, identity, or account record is tracked.
- Complete reachable history contains no earlier wallet material.

No environment file, editor state, build output, cache, temporary credential,
backup file, private report, identity document, invoice, or receipt is tracked
now or appears under a reachable historical path. Local `/bin/`, `/var/`, and
`web/dist/` outputs are ignored generated artifacts.

The tracked bootstrap checksum contains the low-risk historical source path
`/mnt/data/typed-web-genesis.zip`. It is not a credential or identity value,
but it remains unnecessary environment detail for future ordinary cleanup.

## Gate E1 validation

The complete suite was run in an isolated checkout at the exact audited
commit:

```bash
make clean && make build && make test && make demo
```

Outcome: PASS.

- 94 named Go tests passed; zero failed and zero skipped.
- Four Go fuzz targets completed without a panic or invariant failure.
- The C ASan/UBSan corpus accepted two valid vectors, rejected 14 invalid
  vectors, and rejected corrupted evidence.
- The C libFuzzer target completed 5,000 runs without a crash or sanitizer
  finding.
- Offline end-to-end extraction, documentation navigation, and the complete
  demo passed.

All supplemental Gate E1 commands passed:

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
  -o /tmp/twirx-final-c-gcc \
  verifier/c/main.c verifier/c/observation.c verifier/c/sha256.c
clang -std=c2x -O2 -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic \
  -o /tmp/twirx-final-c-clang \
  verifier/c/main.c verifier/c/observation.c verifier/c/sha256.c
clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/twirx-final-main.plist verifier/c/main.c
clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/twirx-final-observation.plist verifier/c/observation.c
clang --analyze -std=c2x -Wall -Wextra -Werror -Wconversion -Wshadow \
  -Wpedantic -o /tmp/twirx-final-sha256.plist verifier/c/sha256.c
```

Coverage remained: adapter 71.3%, atomicfile 56.8%, CAS 63.0%, CBOR 61.3%,
bounded JSON 76.5%, observation 56.1%, and safe fetch 76.7%. The host-specific
benchmark was:

```text
BenchmarkResolveJSONPointer-16  3842841  329.6 ns/op  128 B/op  8 allocs/op
```

`cd web && make check` also passed after replacing the rewritten merge
reference in generated site data. It built ten pages and verified all four
cited evidence commits.

## GitHub ref and PR evidence

The private GitHub repository remained private throughout remediation.

After exact-lease, atomic force updates:

- `main` points to sanitized merge commit
  `ac4f9948ad21b319b11f9caeee9bd4e472c39780`;
- `codex/ci-workflow-registration` points to sanitized commit
  `d6ed40eacade30e5942c3f7160a396b8cb4299bf`;
- `codex/gate-1-genesis` remains unchanged at
  `40210f3cf73004d454733d8b09048c2c5391b4d4`;
- PR #2 remains open and draft, its head uses the sanitized CI commit, and its
  regenerated merge ref contains zero removed-email fingerprint matches;
- no advertised PR #1 merge ref remains, and its advertised head ref is the
  unchanged safe Gate E1 commit.

Residual GitHub limitation: GitHub's API still records PR #1's original
pre-remediation merge OID. The API value is intentionally omitted here because
the old-to-new mapping is private. Changing Git branch refs cannot change that
immutable merged-PR field. This is outside local Git history but may become a
publicly visible metadata path if repository visibility changes.

## Commands and rewrite controls

Principal history operations and checks were:

```bash
git bundle create repository-pre-rewrite.bundle --all
git bundle verify repository-pre-rewrite.bundle
tar --exclude='./.git' --exclude='./bin' --exclude='./var' \
  -czf working-tree-pre-rewrite.tar.gz .

git-filter-repo --force --dry-run --email-callback '<private mapping rule>'
git-filter-repo --force --email-callback '<private mapping rule>'
git fsck --full --no-reflogs

git push --atomic \
  --force-with-lease=refs/heads/main:<privately recorded old OID> \
  --force-with-lease=refs/heads/codex/ci-workflow-registration:<privately recorded old OID> \
  origin <sanitized main refspec> <sanitized CI refspec>

git ls-remote origin 'refs/heads/*' 'refs/pull/*/head' 'refs/pull/*/merge'
```

Temporary rewrite and scanning tools stayed outside the repository and did not
become runtime, build, workflow, or module dependencies. `.github/workflows/ci.yml`
was not modified.

## Exclusions and unresolved risks

1. The private backup bundle intentionally retains pre-remediation metadata.
   It is permission-restricted outside the repository and must not be copied to
   the public VPS tree or committed.
2. Unreachable objects and local reflogs can retain old objects until garbage
   collection. VPS migration must use a fresh clone of sanitized refs, not a
   copy of the local `.git` directory.
3. GitHub's merged PR #1 metadata retention is the sole public-release blocker.
   Repository recreation from the sanitized mirror or confirmed GitHub-side
   purging requires a separate destructive-operation decision.
4. No real funding wallet is declared, so wallet ownership, network, asset,
   and control validation remains future work.
5. The uncommitted website/docs/phase-pack snapshot is not part of this audited
   commit. It was separately scanned and preserved, but any future commit
   containing it requires a final tracked-tree and generated-artifact audit.
6. Scanner non-findings cannot prove absence of an arbitrarily encoded or
   steganographic secret.

## Recommendation

**FAIL for GitHub publication; PASS for continued private development and a
VPS deployment made from sanitized refs and separately scanned static site
artifacts.** Keep the GitHub repository private. Resolve PR #1's retained
server-side merge metadata before making the repository public. No
implementation-behavior blocker remains.
