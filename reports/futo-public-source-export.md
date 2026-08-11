# FUTO sanitized public-source export

**Status:** PASS — deterministic export and local audit complete; repository
publication remains unperformed and requires explicit founder authorization.

**Audit date:** 2026-08-11

**Source revision:**
`2c6fd8f4adfab9e11dcd3268ddfe71c910805919`

**Export manifest SHA-256:**
`e08ccceb0e7f7075cd293874fa31b07fa6208afc7caf2f4ce8102f744d4b6cac`

**Manifest entries before validation builds:** `644`

## Result

`scripts/export-public-source.sh` created a fresh source tree from the exact Git
commit without copying `.git` history. It removed every retained third-party
homepage body, WARC record, raw Common Crawl index response and failed-attempt
raw acquisition artifact named by `PUBLIC_EVIDENCE_PROFILE.md`.

The export retains TWIRX-authored source, specifications, conformance fixtures,
policy decisions, acquisition manifests, digests, packets, deltas and reports.
It adds a source-revision file and a deterministic SHA-256 manifest. The script
fails closed if any known raw-archive filename remains.

No repository was created and nothing was published by this gate.

## Commands executed

```bash
export_parent=$(mktemp -d /tmp/twirx-public-export-run.XXXXXX)
export_tree=$export_parent/tree
scripts/export-public-source.sh "$export_tree" HEAD

cd "$export_tree"
sha256sum -c PUBLIC_EXPORT_MANIFEST.sha256
test ! -e .git
test -z "$(find . -type f \( \
  -name representation.body -o \
  -name warc-record.gz -o \
  -name 'range-*.warc.gz' -o \
  -name 'index-*.jsonl' \
  \) -print)"
GOMAXPROCS=2 make test

/tmp/twirx-futo-tools/gitleaks dir "$export_tree" \
  --redact --no-banner
/tmp/twirx-futo-tools/trufflehog filesystem "$export_tree" \
  --only-verified --no-update --json
```

Private evidence enforcement was separately verified:

```bash
TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE=1 GOMAXPROCS=2 \
  go test ./cmd/twirx-snapshot ./internal/snapshotruntime
```

## Test results

- export manifest: all 644 source entries verified;
- raw-archive filename scan: zero retained file;
- `.git` presence: absent;
- complete public offline test and fuzz suite: PASS;
- public test profile: exactly two private-evidence rebuild tests explicitly
  SKIP because the raw third-party bytes are absent;
- private test profile with required evidence: both real archive packet/delta
  integration tests PASS;
- Gitleaks exported filesystem: zero finding;
- TruffleHog exported filesystem: zero verified or unverified finding.

The two conditional tests are not weakened in the private audited repository.
They execute whenever the evidence exists, and
`TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE=1` makes its absence a hard failure.
All controlled parser, malformed-input, verifier, query, conformance and fuzz
tests remain mandatory in the public export.

## Invariants implemented

- no raw third-party archive representation enters the public tree;
- source-native packets and derivation digests remain unchanged;
- public export is tied to one exact source revision;
- export has no private GitHub PR metadata or preceding Git object history;
- no implementation behavior, network authority or Genesis read-only boundary
  changed;
- missing explicitly required private evidence fails closed.

## Unresolved risks

- The new public repository does not yet exist.
- A public reviewer cannot independently rebuild the two real archive-derived
  packets without reacquiring the identified captures or receiving separately
  authorized evidence access.
- This publication boundary is conservative engineering policy, not legal
  advice.

## Deviation

The FUTO work order requested a fresh public repository. This gate prepared and
validated its exact source tree but did not create or publish the repository
because public-repository authorization and its final name have not been
provided.

## Next recommended gate

After explicit founder authorization, initialize the audited export as a new
one-commit repository, publish it under the approved name, run the public-host
secret scan and GitHub checks, then bind the website and FUTO application to
its URL and commit.
