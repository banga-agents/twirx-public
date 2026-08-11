# FUTO sanitized public-source release

**Status:** PASS — a fresh public repository exists and its exported source,
history and local validation are clean.

**Audit date:** 2026-08-11

**Private source revision:**
`95a537a0a2d54e160a4b67643be2f637f5a7bea5`

**Public repository:**
`https://github.com/banga-agents/twirx-public`

**Public root commit:**
`6646df57baac2b79539d375ec1a0d184e4015bc2`

**Export manifest SHA-256:**
`799ba682bbcb0dae751cae41ea6446dd8ede41a3db96b8ca48906b1aa0cac350`

**Export manifest entries:** `643`

**Tracked public files:** `644`

## Result

`scripts/export-public-source.sh` created a fresh source tree from the exact
private source revision without copying `.git` history. It removed every
retained third-party homepage body, WARC record, raw Common Crawl index
response and failed-attempt raw acquisition artifact named by
`PUBLIC_EVIDENCE_PROFILE.md`.

The exported tree was initialized as a new repository with one root commit and
published publicly as `banga-agents/twirx-public`. The private TWIRX repository
remains private. No legacy Git objects, branches, pull-request commits or
server-side private review metadata were transferred.

The public repository retains TWIRX-authored source, specifications,
conformance fixtures, policy decisions, acquisition manifests, digests,
packets, deltas and reports. It includes the exact private source revision and
a deterministic SHA-256 manifest. The export fails closed if any known raw
archive filename remains.

## Commands executed

```bash
export_parent=$(mktemp -d /tmp/twirx-public-export-run.XXXXXX)
export_tree=$export_parent/tree
scripts/export-public-source.sh "$export_tree" \
  95a537a0a2d54e160a4b67643be2f637f5a7bea5

cd "$export_tree"
sha256sum -c PUBLIC_EXPORT_MANIFEST.sha256
test ! -e .git
test -z "$(find . -type f \( \
  -name representation.body -o \
  -name warc-record.gz -o \
  -name 'range-*.warc.gz' -o \
  -name 'index-*.jsonl' \
  \) -print)"
GOMAXPROCS=1 make test

/tmp/twirx-futo-tools/gitleaks dir "$export_tree" \
  --redact --no-banner
/tmp/twirx-futo-tools/trufflehog filesystem "$export_tree" \
  --only-verified --no-update --json

git init --initial-branch=main
git add --all
git commit -m "Initial public TWIRX release"
gh repo create banga-agents/twirx-public \
  --public --source=. --remote=public --push

public_clone=$(mktemp -d /tmp/twirx-public-clone.XXXXXX)
git clone https://github.com/banga-agents/twirx-public.git "$public_clone"
cd "$public_clone"
test "$(git rev-list --count HEAD)" -eq 1
sha256sum -c PUBLIC_EXPORT_MANIFEST.sha256
/tmp/twirx-futo-tools/gitleaks git . \
  --log-opts=--all --redact --no-banner
GIT_ALLOW_PROTOCOL=file /tmp/twirx-futo-tools/trufflehog git \
  "file://$(pwd)" --only-verified --no-update --json
```

Private evidence enforcement was separately verified:

```bash
TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE=1 GOMAXPROCS=2 \
  go test ./cmd/twirx-snapshot ./internal/snapshotruntime
```

## Test results

- export manifest: all 643 source entries verified;
- raw-archive filename scan: zero retained file;
- `.git` presence in the export before initialization: absent;
- complete exported-tree offline test, conformance, sanitizer and fuzz suite:
  PASS with `GOMAXPROCS=1`;
- the first `GOMAXPROCS=2` complete run encountered a transient Go fuzz
  deadline while the shared host was under load; the bounded one-worker rerun
  completed every target and is the recorded release result;
- public test profile: exactly two private-evidence rebuild tests explicitly
  SKIP because the raw third-party bytes are intentionally absent;
- private test profile with required evidence: both real archive packet/delta
  integration tests PASS;
- Gitleaks exported filesystem: zero finding;
- TruffleHog exported filesystem: zero verified or unverified finding;
- fresh public clone: one reachable commit, manifest PASS, Gitleaks zero and
  TruffleHog zero.

The two conditional tests are not weakened in the private audited repository.
They execute whenever the evidence exists, and
`TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE=1` makes its absence a hard failure.
All controlled parser, malformed-input, verifier, query, conformance and fuzz
tests remain mandatory in the public export.

## Hosted CI limitation

The public workflow was dispatched twice without changing `ci.yml`. GitHub
created the `genesis` check but assigned no runner (`runner_id: 0`) and created
no steps. Its exact annotation is:

> The job was not started because your account is locked due to a billing issue.

Run URL:
`https://github.com/banga-agents/twirx-public/actions/runs/31507308068`

This is an account-level GitHub startup failure, not a workflow or
implementation failure. No hosted execution is claimed, and no workflow
change is justified by this evidence.

## Invariants implemented

- no raw third-party archive representation enters the public tree;
- source-native packets and derivation digests remain unchanged;
- public export is tied to one exact private source revision;
- public history contains no private GitHub PR metadata or preceding Git
  object history;
- no implementation behavior, network authority or Genesis read-only boundary
  changed;
- missing explicitly required private evidence fails closed;
- the original engineering repository remains private.

## Unresolved risks

- A public reviewer cannot independently rebuild the two real archive-derived
  packets without reacquiring the identified captures or receiving separately
  authorized evidence access.
- GitHub-hosted CI remains unavailable until the account billing lock is
  resolved; complete local release validation is preserved separately.
- This publication boundary is conservative engineering policy, not legal
  advice.

## Deviation

Hosted GitHub CI did not execute because GitHub rejected the job before runner
assignment. The workflow was not changed because the failure is external to
its registration or implementation.

## Next recommended gate

Resolve the GitHub account lock and rerun the unchanged workflow. This is not a
blocker to the integrity of the sanitized source release, but the result must
remain visibly distinct from completed hosted CI.
