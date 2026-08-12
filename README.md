# TWIRX

**An open, self-hostable semantic data plane for the Web.**

TWIRX continuously compiles public sources into reusable, typed, proof-bearing
semantic state and capabilities so agents can query, compare and monitor the
Web without repeatedly browsing and reinterpreting the same interfaces. It
preserves what an origin represented and the exact derivation from evidence to
every typed value.

**TWIRX is the reference implementation and public service of the Typed Web Commons.**
The Typed Web Commons is the protocol commons and stewardship
structure. The Semantic Data Plane is the core architecture. Agent Utility
Universes are the public product built on that plane.

This repository contains the verified E1 evidence kernel, E2 typed operation
and proof path, the Atlas admission controls, an immutable read-only Query Lab,
and the E4.5 Opportunity Utility release candidate. E4.5 compiles 83,087 real
Grants.gov records into 1,037,679 source-derived packets and 83,087 provisional
frames. Its mappings remain candidates, privacy-sensitive eligibility text is
withheld, and the larger runtime is not deployed on the shared demonstration
host. See [`reports/e4-5-opportunity-admission.md`](reports/e4-5-opportunity-admission.md).
The versioned public evidence release is
[`v0.4.5-rc.1`](https://github.com/banga-agents/twirx-public/releases/tag/v0.4.5-rc.1).

```text
controlled origin
      ↓
policy-constrained observation
      ↓
content-addressed evidence
      ↓
canonical observation envelope
      ↓
independent verification
      ↓
deterministic adapter
      ↓
native statement + semantic view
      ↓
field-level provenance
```

## What this release proves

The first vertical slice can:

1. Retrieve a controlled public representation through an explicit network policy.
2. Store the response body in a SHA-256 content-addressed store.
3. emit a deterministic CBOR observation envelope with a strict CDDL schema.
4. Verify the envelope and body using both Go and an independent restricted-C verifier.
5. Execute a manually admitted JSON adapter without network access.
6. Preserve the provider's native term and lexical value beside the semantic interpretation.
7. Return field-level provenance containing the observation, body, adapter, locator, mapping, and transformation chain.

E2 adds a catalog-only Lab with five typed read operations, deterministic proof
bundles, Go/C result verification, and generated CLI, JSON Schema, OpenAPI, and
local MCP bindings. Its deployable HTTP surface is explicitly replay-only;
fresh observations remain a local CLI workflow until the isolated egress
boundary is admitted. It does **not** accept arbitrary URLs or attempt browser
automation, model-assisted schema induction, registry federation, writes,
payments, or blockchain anchoring.

E3 freezes exactly 500 public-knowledge origin candidates and reports catalog,
policy, technical, publisher, health, adapter-trust, and mapping-trust state
as separate dimensions. Three human policy decisions are complete for TWIRX,
World Bank and the exact RFC Editor archive profile; 497 candidates remain
pending. The controlled origin is a `test_fixture` and is excluded from public
counters. The Atlas control-plane process has no HTTP client. Its deterministic
admission factory and sealed-egress candidate cannot turn agent-prepared
metadata into policy, semantic or execution authority. Five commercial/access
candidates remain descriptive review inputs, not canonical offers or
executable routes.

The Atlas-500 runtime gate derives one explicit admission work item and one
dry-run frontier outcome for every selected origin. Its loopback workload
traverses all 500 identities and performs 50,000 concurrent lookups without
turning catalog-scale execution into a claim that 500 origins are
policy-approved, compiled or live. Schedulers remain disabled unless exact
separate authority admits an execution.

The immutable snapshot demonstration compiles admitted E2 replay operations
and the approved RFC Editor archive profile into canonical semantic packets,
embeds their complete proof, materializes two bounded views, and answers exact
typed queries and packet traces without a network request. Its generated build
report separates 500 selected identities from the three public origins that
currently have packets and excludes the controlled fixture by default. The
read-only runtime exposes all 500 identities through bounded origin discovery
while reporting packet state separately.

## Truth contract

Typed Web does not claim that a provider's content is objectively true. It makes a narrower and verifiable claim:

> This origin returned these bytes at this time; this adapter extracted this native statement at this locator; these declared transformations and mappings produced this typed value.

Every canonical result must either carry that derivation or explicitly remain unresolved.

## Run the complete local demonstration

Requirements:

- Go 1.23 or later. Go 1.26 is recommended for current security and toolchain support.
- GCC with C2x support.
- Clang 17 or later for sanitizer tests.
- Bash, Python 3, and curl.

```bash
make demo
make demo-e2
make demo-e3
make demo-e3-worker
make demo-semantic-snapshot
make stress-e2
make stress-e3-500
make stress-semantic-snapshot
make stress-semantic-snapshot-scale
```

The demo starts a controlled local origin, records an observation, verifies it in Go and C, shuts the origin down, and then extracts the typed result offline.

Run the complete test suite:

```bash
make test
```

The suite includes committed Go/C conformance vectors, Go fuzz targets,
a Clang libFuzzer target with AddressSanitizer and UndefinedBehaviorSanitizer,
network-policy regressions, corrupted-evidence checks, and offline end-to-end
replay. It does not require the public internet.

The E2 demonstration selects an admitted controlled origin, invokes the real
typed operation through CLI and MCP, shows native and semantic values with
field provenance, verifies the manifest-last bundle in Go and C, and replays
it offline.

The bounded stress command invokes all five admitted replay operations through
the loopback HTTP service, validates every typed response, rehashes every
distinct downloaded proof bundle, verifies each publication in both primary
and restricted implementations, records process resource samples, and checks
that configured overload is rejected. It makes no public-TLS or fresh-origin
capacity claim.

The Atlas-500 stress command starts the actual read-only Atlas HTTP service on
literal loopback, derives the complete 500-origin admission queue and
frontier, exercises every origin directly and through pagination and filters,
runs 100 concurrent full-catalog rounds, rejects malformed requests, and
requires byte-identical status after restart. It has no public-origin egress
capability.

The Semantic Snapshot demonstration creates a new temporary immutable release,
verifies every manifest-bound byte, runs a population query and a two-origin
query, and retains the directory for proof inspection. The builder and runtime
perform zero origin-network requests.

The Semantic Snapshot stress command starts the literal-loopback runtime,
executes a bounded concurrent two-origin query workload, requires stable query
and result identities, records latency and process resource samples, restarts
the service, and requires byte-identical status. Its throughput applies only to
the recorded host, 18-packet replay corpus and stated workload.

The scale variant compiles 25,000 additional controlled native-statement
packets, segments the packet and proof indexes, verifies the immutable snapshot,
samples canonical packets with the restricted-C verifier, and exercises the
same HTTP path. Controlled packets are labeled `test_fixture`, remain absent
from public materialized views, and are excluded by every runtime default; this
is capacity evidence, not a claim of 25,000 public semantic statements.

Build only:

```bash
make build
```

## CLI

```bash
# Controlled local fixture. Loopback is denied unless explicitly enabled.
bin/tw observe \
  --url http://127.0.0.1:18080/product/sku-001 \
  --out var/demo \
  --cas var/cas \
  --allow-loopback

bin/tw verify \
  --observation var/demo/observation.cbor \
  --cas var/cas

bin/tw-verify-c var/demo/observation.cbor var/cas

# This command does not contact the origin.
bin/tw extract \
  --observation var/demo/observation.cbor \
  --cas var/cas \
  --adapter adapters/testorigin-product/adapter.json \
  --out var/demo/result.json
```

## Repository map

| Path | Purpose |
|---|---|
| `cmd/tw` | Genesis command-line interface |
| `cmd/tw-test-origin` | Controlled fixture origin |
| `cmd/twirx-lab` | E2 CLI, local MCP server, generator, and loopback Lab service |
| `cmd/twirx-stress` | Bounded E2 HTTP/proof stress and evidence client |
| `cmd/twirx-atlas` | E3 offline validation, metrics, full-Atlas stress, and loopback read-only API |
| `cmd/twirx-snapshot` | Offline immutable snapshot builder, verifier, query/trace CLI, and loopback runtime |
| `cmd/twirx-archive` | Offline sealed-work-order planner, archive evidence importer, and verifier; no HTTP client |
| `cmd/twirx-archive-acquire` | Operator-only fixed-host Common Crawl acquisition under a sealed work order; not linked into the public runtime |
| `internal/safefetch` | URL policy, resolution, redirects, limits, and retrieval |
| `internal/cas` | Filesystem content-addressed evidence store |
| `internal/observation` | Canonical observation envelope |
| `internal/adapter` | Deterministic manual adapter runtime |
| `verifier/c` | Independent bounded C verifier |
| `schemas/cddl` | Normative Genesis wire schemas |
| `adapters` | Admitted example adapters |
| `conformance` | Fixtures, adversarial vectors, and expected results |
| `contracts/e2` | Canonical bounded TWIR operation contracts |
| `origins` | Reviewed E2 origin catalog and replay fixtures |
| `lab` | Live Lab UI and hardened deployment templates |
| `generated/e2` | Generated JSON Schema, OpenAPI, MCP, and CLI bindings |
| `atlas` | Exact E3 candidate selection and separately admitted registry |
| `generated/e3` | Evidence-derived Atlas metrics |
| `internal/snapshotartifact` | Bounded snapshot carriage artifacts and parser |
| `internal/snapshotbuild` | Deterministic E2-evidence-to-packet snapshot compiler |
| `internal/snapshotruntime` | Verified materialized query and packet trace runtime |
| `internal/archiveimport` | Bounded Common Crawl index/WARC parsing and evidence-before-parsing spool |
| `internal/archiveacquire` | Exact official-host index/range retrieval and manifest-last acquisition reconciliation |
| `spec/data-plane` | Normative E3.3 packet, query, delta, state, security, and benchmark design |
| `deploy/postgresql` | Inactive PostgreSQL deployment and recovery design |
| `docs` | Mintlify technical documentation |
| `snapshotlab` | Dependency-free immutable Query Lab UI and hardened deployment profile |
| `web` | Dependency-free generated public project website |
| `funding` | Public treasury, expenses, wallet declarations, and receipts policy |
| `tasks` | Scoped implementation work orders for coding agents and maintainers |

## Constitutional constraints

- The protocol is defined by specifications and conformance, not by one implementation.
- No language, cloud, chain, model provider, browser, or organization is mandatory.
- Native source meaning is preserved before semantic normalization.
- Model output may propose; it cannot promote itself into the canon.
- Canonical writes pass through admission and conformance.
- Genesis is public, read-only, local-first, and browser-free.
- There is no protocol token and no sale of governance rights.
- Funding, maintainer compensation, infrastructure, and project expenses are disclosed according to [`FUNDING.md`](FUNDING.md).

## Documentation

The technical documentation is in [`docs/`](docs/). Mintlify now uses `docs.json`; preview it locally with:

```bash
cd docs
mint dev
```

Build and verify the public website without network access:

```bash
cd web
go run .
```

## Contributing

Read [`MANIFESTO.md`](MANIFESTO.md), [`CHARTER.md`](CHARTER.md), [`SECURITY.md`](SECURITY.md), [`CONTRIBUTING.md`](CONTRIBUTING.md), and [`AGENTS.md`](AGENTS.md) before changing protocol behavior.

The first Codex work order is [`tasks/001-genesis-source-statement-slice.md`](tasks/001-genesis-source-statement-slice.md).

## Status

**Genesis Preview.** E1 is implemented and locally attested. E2's acyclic
publication candidate passes locally. The E3 admission factory, disabled
sealed-egress worker, packet/query contracts, archive importer and immutable
snapshot runtime are implemented candidates; none creates arbitrary-URL,
browser, model, payment or write authority.

The exact FUTO snapshot carries 500 selected Atlas identities, three completed
human policy decisions, 15 public-source packets across three origins, five
separately labeled fixtures, two materialized views and one genuine historical
origin delta. The read-only runtime exposes all 500 identities and their actual
packet state without treating selection as review, observation, compilation or
live status. The immutable public Query Lab is live at
[`lab.twirx.org`](https://lab.twirx.org/), performs zero origin calls and
accepts no arbitrary URL. A fresh sanitized public repository is available at
[`banga-agents/twirx-public`](https://github.com/banga-agents/twirx-public).
An isolated encrypted Storage Box archive has passed byte-identical restore;
the planned versioned Object Storage replica remains open. Production PostgreSQL remains
prohibited on the shared Meridian host. See
[`reports/futo-grant-readiness.md`](reports/futo-grant-readiness.md) and
[`spec/data-plane/`](spec/data-plane/).
