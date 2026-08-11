# Genesis Threat Model

## Protected assets

- canonical protocol artifacts;
- observation and body integrity;
- typed-result and manifest-last proof-bundle integrity;
- adapter and ontology identities;
- funding wallet declarations and treasury records;
- maintainer credentials and signing keys;
- local filesystem outside the configured workspace;
- private networks and cloud metadata;
- the distinction between source statements and semantic interpretations.

## Adversary control assumptions

Assume an attacker may control:

- submitted URL text;
- DNS records and address changes;
- redirects;
- HTTP headers and status;
- compressed and malformed bodies;
- JSON keys, values, nesting, and duplicate semantic claims;
- metadata that attempts to impersonate publisher authority;
- adapter proposals;
- ontology and mapping proposals;
- prompt-injection content used by future models;
- one ordinary contributor account;
- resource-exhaustion attempts.

Assume an attacker may also submit or alter candidate identity, origin state,
policy, publisher, authority, semantic, and training-readiness metadata in an
attempt to obtain an inflated public count or network admission.

## Genesis trust boundaries

```text
untrusted URL
    ↓
URL policy and controlled resolver
    ↓
HTTP fetcher
    ↓
immutable body in CAS
    ↓
canonical observation envelope
    ↓
Go verifier + independent C verifier
    ↓
manually admitted deterministic adapter
    ↓
result with native and semantic views
```

The controlled fixture flag weakens public-network policy only for loopback development. It must not become a public service option.

The E2 public boundary is narrower than the local observer:

```text
origin_id + operation_id + bounded typed input
    ↓
reviewed catalog and contract
    ↓
per-client, origin, concurrency, and deadline limits
    ↓
admitted immutable replay observation + transport evidence
    ↓
offline extraction and manifest-last publication
    ↓
Go re-extraction + restricted-C bundle verification
```

The deployable E2 HTTP surface is replay-only and reports that state through
status, discovery, origin and OpenAPI views. Fresh execution fails closed
before quota or execution admission. No public E2 field accepts a URL.
Submitted origin URLs enter a human review queue and are not fetched
automatically. Explicit local CLI fresh workflows remain outside this public
boundary.

The E3.0 Atlas control plane adds a non-network boundary:

```text
exact candidate selection (untrusted as network authority)
    ↓ strict bounded validation
canonical registry with orthogonal evidence-attested state
    ↓ deterministic metric derivation
loopback read-only discovery API
```

The process contains no HTTP client. Selected candidates cannot be fetched or
invoked. A technical-stage claim cannot imply policy completion, publisher
approval, current health, adapter trust, or mapping trust; each dimension
requires its own evidence.

The E3.2 admission and retrieval candidate adds two boundaries without
activating public retrieval:

```text
per-origin evidence + explicit human decision
    ↓ deterministic admission render
canonical registry and policy bindings
    ↓ sealed work-order ID only
dedicated unprivileged egress worker
    ↓ DNS/address/redirect/TLS validation + host firewall
manifest-last immutable evidence spool
```

Agent-prepared proposals cannot claim human approval. The worker cannot accept
a caller-supplied URL, issue a work order, parse content, access secrets or
databases, or mutate the registry.

Interface, capability, payment, offer, and publisher-readiness declarations
are untrusted descriptive metadata. Discovery does not imply safety,
publisher approval, policy permission, or executability. E3.2 can mark only an
already compiled E2 public-read operation as admitted; WebMCP, browser,
authenticated, write, financial, legal/material, and destructive effects have
no execution path.

## Primary threats and controls

### Server-side request forgery

Controls:

- HTTP/HTTPS allowlist;
- no URL credentials;
- private and reserved address rejection;
- direct dialing of validated resolved addresses;
- fresh DNS resolution and address validation before every connection;
- redirect revalidation;
- E2 initial and redirected URLs constrained to the reviewed catalog host;
- E3.2 routes constrained to sealed work orders, exact admitted hosts,
  standard HTTPS, human-decision and policy-evidence digests;
- cgroup-scoped network denial for private, loopback, link-local, metadata,
  multicast and reserved ranges in the unactivated service candidate;
- standard ports by default;
- request timeout and body limits.

Residual risk:

- the systemd network boundary is a reviewed candidate, not target-host
  activation evidence;
- a compromised host kernel, resolver stack, or privileged operator remains
  outside application enforcement;
- per-origin public egress still requires completed human review and an exact
  founder-approved work order.

### Evidence substitution

Controls:

- SHA-256 body addressing;
- envelope body digest and size;
- rehash before extraction;
- independent C verification;
- immutable path convention;
- manifest-last bundle publication;
- regular-file and symlink rejection;
- result bindings to input, observation, transport, adapter, contract, and semantic closure.

Residual risk:

- local filesystem compromise can replace code and evidence together unless release signing and external transparency are added.

### Parser and memory corruption

Controls:

- memory-safe Go in the privileged Genesis path;
- tiny bounded CBOR subset;
- fixed C envelope schema;
- C verifier input size cap;
- dual compiler builds and sanitizers;
- shared positive and negative Go/C conformance vectors;
- Go mutation fuzzing and a sanitizer-backed C libFuzzer target;
- no C networking, plugin loading, or canonical writes.

Residual risk:

- parser defects remain possible and require continuous fuzzing and independent implementations.

### Semantic injection

Controls:

- provider text remains untrusted data;
- native term and value remain visible;
- mappings are explicit and versioned;
- adapters are manually admitted;
- no model exists in the canonical path;
- required fields fail closed.

Residual risk:

- a reviewed adapter can still encode a mistaken interpretation; conformance and disagreement mechanisms must grow.

### Resource exhaustion

Controls:

- URL, response, envelope, JSON byte, nesting, scalar, container, token, redirect, timeout, and read limits;
- duplicate JSON keys and trailing top-level values rejected before extraction;
- no browser or model in Genesis;
- no unbounded ontology reasoning;
- fixed extraction operations;
- bounded per-client and per-origin token buckets;
- a global invocation-concurrency cap and process-level systemd limits;
- a bounded loopback stress harness that treats configured HTTP 429 responses
  as admission outcomes, validates every successful typed response, and
  rehashes downloaded proof artifacts;
- a one-worker E3.2 process lease, bounded failure circuit, cooldown,
  revocation, and emergency stop;

Residual risk:

- the E3.2 circuit breaker is local pilot state, not distributed abuse
  suspension or fleet-wide health management;
- application limits cannot replace host resource controls, network egress policy, monitoring, or incident response;
- the in-memory token buckets reset on process restart and are not a distributed abuse-control system.

### State and metric inflation

Controls:

- every non-unknown interface, capability, access, offer, and readiness claim
  binds an immutable SHA-256 evidence artifact;
- inferred commercial candidates remain visibly provisional and outside
  canonical public counters until explicit human admission;
- sponsorship, payment metadata, and commercial status cannot affect policy,
  trust, semantic fit, authority, or ranking;
- exactly 500 unique, origin-only HTTPS candidates under fixed family quotas;
- candidate hints are explicitly null/empty;
- policy records bind the SHA-256 digest of the exact policy-set bytes;
- policy fields in the registry must match their reviewed record;
- dry-run frontier output identifies itself as network-disabled and contains
  no destination URLs;
- the derived admission work queue covers all 500 selected origins while
  preserving missing dossiers as `not_prepared`;
- the dry-run frontier covers all 500 selected origins exactly once, including
  candidates absent from the canonical registry;
- the full-catalog stress client accepts only a literal-loopback HTTP origin,
  disables proxies and redirects, and pins its transport destination;
- admitted records must preserve a selected identity;
- per-origin sources and evidence are independently hashed before rendering;
- only explicit completed human admission artifacts enter canonical state;
- pending agent-prepared proposals remain outside canonical state;
- each state dimension has its own bounded vocabulary and evidence rules;
- pending reviews never contribute to completed-policy counts;
- controlled fixtures are separately counted and excluded from all public-origin counters;
- exact technical stage and technical-at-or-beyond counts are published without collapsing other dimensions;
- model readiness is derived from all corpus thresholds and defaults false;
- unknown fields, duplicate keys, trailing data, quota drift, unsafe evidence
  paths, and digest substitutions fail closed.

Residual risk:

- human reviewers can make identity, policy, authority, or semantic mistakes;
- RFC 9309 expresses crawler preferences, not authorization, and terms or
  policy interpretation can still be mistaken;
- repository control can replace validation and evidence together until
  signed releases and external transparency exist;
- the candidate list is unreviewed and may contain obsolete, redirected, or
  misclassified origins.

### Treasury compromise

Controls:

- only public addresses enter the repository;
- seed phrases and private keys are forbidden;
- project wallet separated from personal trading wallets;
- migration to multisignature control after additional trusted stewards exist;
- ledger reconciliation against public chain records.

Residual risk:

- a single-founder wallet is a temporary Genesis concentration of authority and must be disclosed as such.

### Persistent semantic-state poisoning and overwrite

E3.3 introduces a design for reusable packet history, materialized state and
subscriptions. It is not deployed in the current gate.

Controls:

- canonical packets always retain native subject, predicate, locator and
  lexical value before optional semantic mapping;
- observed-native, provisional-semantic and attested-semantic lanes remain
  visibly distinct;
- semantic keys bind origin, time, jurisdiction, language and all
  meaning-changing dimensions;
- origin, semantic and canon deltas are separate canonical classes;
- packet and delta cores are immutable and self-digest-free, with manifests
  written last;
- current heads and materialized views are rebuildable projections with exact
  packet and canon references;
- conflicting source statements are preserved rather than silently averaged
  or overwritten;
- model/vector output can generate only provisional candidates and cannot
  promote itself.

Residual risk:

- human mappings and semantic-key designs can still be mistaken;
- an admitted deterministic adapter can systematically misinterpret a source;
- a compromised canon reviewer could affect many derived views, although the
  change remains a canon/semantic delta rather than a false origin delta;
- retained public evidence can create privacy or legal risk if admission
  classification is wrong.

### State-store, subscription and recovery failure

Controls:

- PostgreSQL is an implementation choice, not protocol authority;
- canonical bytes remain in content-addressed evidence and are rehashed before
  admission;
- globally unique identity tables prevent partition-local digest constraints
  from being presented as global;
- packet and delta logs are append-only to runtime roles;
- head, delta, economic event and outbox changes commit transactionally;
- persisted cursor state, not transient notification, is subscription
  authority;
- public query objects bound rows, packets, proof bytes, live origins and
  deadlines;
- database access remains loopback/Unix-socket only and the egress worker has
  no credentials or socket access;
- encrypted off-host base backups, WAL archiving and isolated restore drills
  are activation requirements.

Residual risk:

- the target VPS currently has a degraded root RAID1 path and nearly exhausted
  swap, so database deployment is blocked;
- database bugs, operator mistakes or WAL/archive gaps may exceed recovery
  targets;
- a durable outbox provides replay but public clients can still process an
  event more than once and must use its identity/cursor;
- materialization or index corruption can temporarily make query state
  unavailable even while canonical artifacts remain intact.

### Economic and ranking capture

Controls:

- publisher offer price, operator cost, funding class and sponsor identity are
  separate typed facts;
- source authority, proof, freshness, semantic coverage, cost and uncertainty
  remain separate planner dimensions;
- hard query constraints apply before a visible Pareto frontier and named
  caller preference;
- sponsorship, revenue or commercial value cannot change policy, mapping trust,
  source authority, semantic rank or canon admission;
- every published cost/value metric identifies measured versus estimated data
  and its method version.

Residual risk:

- preference policies can encode value judgments and require transparent
  review;
- incomplete economic telemetry may understate maintenance and review cost;
- users may still treat a selected Pareto candidate as endorsement unless the
  explanation and competing sources remain visible.

### Immutable snapshot substitution and stale-state misrepresentation

The local grant demonstration compiles admitted replay evidence into a
read-only snapshot. It does not make replay data current and is not deployed by
the current gate.

Controls:

- the canonical manifest is self-digest-free, published last and identified by
  the SHA-256 digest of its exact bytes;
- every artifact path, type, byte size and digest is verified before semantic
  parsing;
- symlinks, hard links, devices, FIFOs, sockets, traversal and reserved paths
  fail closed;
- packet sequence, identity, count, proof index, materialized views and build
  report are reconciled with the manifest;
- each packet retains native term, locator and lexical value plus the complete
  E2 observation, transport, adapter, contract, mapping and closure evidence;
- offline replay is explicitly stale and never represented as a current
  publisher statement;
- 500 Atlas candidate identities are reported separately from the two public
  origins that currently produce packets;
- controlled fixtures are excluded from public counters, views, queries and
  traces by default;
- the runtime refuses live refresh, non-loopback binding and all state writes;
- query bodies, result/proof bytes, headers, timeouts and concurrency are
  bounded, and the declared query deadline is enforced during packet scans.

Residual risk:

- project-recorded replay fixtures do not independently prove that a publisher
  served those exact bytes, even though the complete project evidence is
  retained;
- human-approved mappings and E2 contracts may be mistaken;
- the Go snapshot carriage/reconciliation layer does not yet have a second
  independent implementation, although canonical constituent objects do;
- a compromised process owner can replace an active release between process
  starts unless a separately admitted read-only service and atomic updater are
  configured;
- embedded public representation bodies may later create privacy, copyright or
  retention risk if upstream admission was wrong;
- the baseline has no prior snapshot and therefore cannot honestly demonstrate
  a semantic change stream.

## Out of scope in Genesis

- authentication and private sites;
- write actions;
- payments;
- browsers and JavaScript execution;
- model-assisted schema induction;
- public arbitrary-origin or submitted-URL execution;
- publisher identity verification;
- signed releases and transparency anchoring.

Out of scope means unimplemented, not unimportant.
