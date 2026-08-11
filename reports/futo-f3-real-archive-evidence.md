# FUTO F3-F5 local real-archive evidence

**Recommendation:** PASS for local archive, delta, snapshot, query and stress
evidence; NOT ADMITTED for public publication or deployment

**Policy review time:** `2026-08-11T12:15:50Z`

**Snapshot source revision:**
`d13c0bfaf4aed19b9db541b3fd7ebc97e251ebc3`

**Complete validation revision:**
`bdee40c52bbd2be137cfadc577bac6098c799968`

## Result

The Genesis steward's three decisions were materialized without widening any
approved route, effect or retrieval scope:

- `twirx-org`: `completed + permit_live`, limited to the exact
  publisher-authored public project surfaces in the proposal;
- `api-worldbank-org`: `completed + permit_with_constraints`, limited to the
  existing bounded E2 Indicators operation;
- `rfc-editor-org`: `completed + profile_only`, migrated atomically to
  `https://www.rfc-editor.org` and limited to the exact homepage captures in
  `CC-MAIN-2025-30` and `CC-MAIN-2026-25`.

The scheduler remains disabled and its generated frontier contains zero jobs.
No arbitrary URL, live RFC Editor crawl, authentication, browser, model,
payment, action or canon-promotion authority was added.

## Real archive acquisition

The sealed work order digest is:

```text
sha256:c96281801f9e95e603fb950a77b98ac3cf9ad0bf29920259a1e062b59dfd3724
```

The restricted helper made exactly four requests: two bounded requests to the
official Common Crawl index host and two exact byte-range requests to the
official Common Crawl data host. The final manifest was published last and
rehashes to:

```text
sha256:a28259dfbb3f3e0fa70002fc4c8769ad0e78503a53100481da41d63ecff6b198
```

| Collection | Capture UTC | Compressed bytes | Representation bytes | Representation digest |
| --- | --- | ---: | ---: | --- |
| `CC-MAIN-2025-30` | `2025-07-08T10:21:38Z` | 15,237 | 69,089 | `sha256:fe8b7109b112b755ca1f869eb36ec585159afe86a7f6fe4504a32538cab57152` |
| `CC-MAIN-2026-25` | `2026-06-07T20:53:41Z` | 36,598 | 174,086 | `sha256:6b591e959e45e918dac2fc9ab829ef0ac5a75b1908c864b0432324a7b02c5f89` |

Every capture is labeled `archive_observation`, `historical`,
`observed_by: common_crawl`, and `current_publisher_statement: false`.

The first acquisition attempt retained the index response and compressed WARC
bytes before failing closed on a legitimate repeated `WARC-Protocol` field.
It has no final acquisition or capture manifest and therefore has no admission
authority. The parser was extended only for WARC fields defined as repeatable;
duplicate singleton fields remain rejected. The successful acquisition was
then started in a new immutable directory.

## Source-native packet and delta proof

The offline archive profile extracts exactly the first
`html/head/title[1]/text()` lexical bytes. It does not decode HTML entities,
normalize whitespace or infer a shared semantic concept.

| Capture | Exact native lexical value | Packet digest |
| --- | --- | --- |
| 2025 | ` &raquo; RFC Editor` | `sha256:984057b744839253deff9c944d9fee394a072805af77120671f06b84d9bec2ca` |
| 2026 | `RFC Editor` | `sha256:a5c7a0bf8d1b39574d70b65460c7891e0e99fe3f22b884f91d90637224875cf4` |

One `origin + modified` delta binds the two packet and observation identities:

```text
sha256:efd6c4341375dc17cd366973c903c7cfaf5129d3805761c8a52a16410f406033
```

No semantic reinterpretation delta or canon delta is emitted because neither a
mapping nor canon module changed. This preserves the three delta classes
instead of fabricating changes that did not occur.

## Immutable snapshot and query evidence

The snapshot manifest was rebuilt byte-identically from the committed inputs:

```text
snapshot_id:     sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5
created_at:      2026-08-11T12:59:03Z
source_revision: d13c0bfaf4aed19b9db541b3fd7ebc97e251ebc3
```

Verified contents:

```text
500 Atlas identities
3 public origins with packets
1 controlled fixture origin
6 operations
20 packets total: 15 public, 5 controlled fixtures
1 real origin delta
2 materialized views
75 proof artifacts
0 compiler or runtime origin-network requests
0 current-publisher claims from archive evidence
```

The historical RFC Editor query returned both exact lexical values, excluded
all five fixtures and made zero network requests. Its query and result IDs are:

```text
query:  sha256:fb2fcb83f5ccaaaec1d5724d0a966ea4388221615cc608b68f6026e91dee8f72
result: sha256:bba1a9d1fc8656223eaba0882aa1639775fe4ec58f6db469787cfbaf31992386
```

The existing cross-origin query returned four proof-linked rows across
`twirx-org` and `api-worldbank-org`, excluded fixtures and made zero network
requests. Its query ID is:

```text
sha256:9ba2b73f4c43134a104cb89c65031b555d3a99dc4247c7f2a0cb22561e6458e7
```

## Local stress result

The literal-loopback immutable runtime served the historical query 5,000 times
at concurrency eight with 5,000 successes, zero failures and zero origin
calls:

```text
duration: 609,432 us
rate:     approximately 8,204 requests/second
p50:      691 us
p95:      1,123 us
p99:      1,351 us
```

This measurement applies only to the stated host, 20-packet immutable snapshot
and bounded query. It is not a public-network, large-corpus or production
capacity claim.

## Exact commands executed

```bash
bin/twirx-archive-acquire run \
  --work-orders atlas/archive-work-orders \
  --id rfc-editor-futo-history \
  --out atlas/archive-acquisitions/rfc-editor-futo-history

bin/twirx-archive-acquire verify \
  --root atlas/archive-acquisitions/rfc-editor-futo-history

bin/twirx-snapshot build \
  --root . \
  --out var/futo-public-snapshot-d13c0bf-rebuilt \
  --source-revision d13c0bfaf4aed19b9db541b3fd7ebc97e251ebc3 \
  --created-at 2026-08-11T12:59:03Z \
  --archive-acquisition rfc-editor-futo-history

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

bin/twirx-snapshot serve \
  --snapshot var/futo-public-snapshot-d13c0bf-rebuilt \
  --id sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5 \
  --listen 127.0.0.1:18091

bin/twirx-snapshot stress \
  --base http://127.0.0.1:18091 \
  --query examples/semantic-query-rfc-editor-history.json \
  --requests 5000 \
  --concurrency 8 \
  --out var/futo-archive-stress-rebuilt.json

make test
GOMAXPROCS=2 go test -race ./...
GOMAXPROCS=2 go vet ./...
git diff --check
```

The first complete `make test` attempt encountered a Go fuzz-harness
`context deadline exceeded` after `FuzzCompressedWARC` had otherwise completed
its generated cases. The same one-second oversubscription flake had previously
occurred in another large fuzz target. `FUZZ_WORKERS=4` now bounds short Go
fuzz concurrency without removing a target or changing parser limits. The
complete `make test` command then passed at validation revision `bdee40c`:

- all Go package and integration tests;
- all 22 one-second Go fuzz targets;
- observation C verification under ASan/UBSan: two valid accepted, 14 invalid
  rejected, corrupted evidence rejected;
- shared E2 Go/C conformance;
- E3.3 S1 restricted-C conformance: 56 total, 16 accepted, 40 rejected;
- three 5,000-run C libFuzzer campaigns under ASan/UBSan;
- E2 end-to-end source-statement/provenance replay;
- immutable Semantic Snapshot integration;
- documentation navigation checks;
- race detection, vet and whitespace validation.

The restricted-C verifier also accepted the snapshot manifest, all 20 packet
cores and the real delta core. Base64-wrapped packet and delta cores were
decoded into bounded temporary regular files before verification because the C
CLI intentionally seeks its input file.

## Security and publication limitation

Gitleaks 8.30.1 and TruffleHog 3.96.0 found no verified secret in the complete
61-commit reachable history. TruffleHog also found no verified or unverified
secret in an exact tracked-tree export.

Gitleaks's tracked-tree scan reported two detections on the same line of the
exact retained 2026 RFC Editor representation. The source page embeds a
browser-side Typesense client value. It is not a TWIRX credential and changing
the retained body would invalidate the source-evidence digest. Nevertheless,
the decision approved retrieval, not public redistribution of that embedded
value. Public snapshot/repository publication therefore remains blocked until
the steward records a redistribution treatment or the architecture keeps raw
representation evidence outside the public source repository while preserving
verifiability. The value is deliberately not repeated in this report.

## Files changed

- three admission decisions, policy evidence spools and canonical Atlas state;
- RFC Editor canonical-host migration ADR and derived selection/admission data;
- one sealed exact archive work order and one manifest-last acquisition;
- one source-native archive profile, parser, packet compiler and delta path;
- immutable snapshot build, verification, query, packet, trace, proof and delta
  runtime paths and adversarial tests;
- RFC historical query example, conformance evidence and tasks 007-009;
- `Makefile` dependency and bounded fuzz-worker corrections;
- this evidence report.

## Unresolved risks and deviations

1. The FUTO snapshot profile uses the already-published acquisition-manifest
   digest as the delta `batch_id` to avoid a self-digest cycle. This is
   disclosed in task 009 and does not silently amend the normative packet-batch
   contract.
2. Archive packets remain `observed_native`; no semantic mapping is claimed.
3. The raw retained representation requires an explicit public-redistribution
   treatment before publication.
4. Object Storage upload/versioning, independent encrypted Storage Box backup
   and byte-identical restore have not occurred.
5. The snapshot runtime has not been deployed to `lab.twirx.org`.
6. No PostgreSQL, continuous crawling, browser, model, payment or write-action
   work was performed.

## Next recommended gate

Resolve the raw-evidence publication treatment, publish and independently
restore the exact immutable snapshot off-host, then review the stateless
Meridian activation plan. Deployment remains a separate founder-admitted gate.
