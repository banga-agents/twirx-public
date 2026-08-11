# E3.3 semantic data plane migration inventory

Status: design inventory; no migration executed

Base commit: `9aebad7ffaa888c070352bf8247b21e24e6b5213`

Date: 2026-08-11

## Purpose

This inventory maps admitted E1, E2 and E3.2 artifacts into the E3.3 semantic
data plane. It is deliberately additive. No existing artifact is reclassified,
deleted, rewritten or weakened. A future migration must be deterministic,
restartable, digest-preserving and reversible until founder admission.

## Source-to-target map

| Existing source | Existing authority | E3.3 use | Migration rule |
| --- | --- | --- | --- |
| Observation Envelope and transport evidence | E1/E2 evidence | packet source and derivation references | Preserve exact bytes and digest; reference, never recode as new origin evidence |
| Filesystem CAS | immutable artifact store | canonical packet, batch, result and manifest bytes | Keep authoritative; database rows point to verified CAS digests |
| Native extraction result | E1 deterministic derivation | `observed_native` packet input | Preserve source term, lexical value, locator and unresolved status before any mapping |
| TWIR operation contract | E2 typed operation authority | capability description and compiler contract | Bind exact contract digest and operation version |
| Semantic closure and mapping artifacts | E2 reviewed semantic view | provisional or attested semantic packet derivation | Preserve mapping IDs/digests; never infer trust from successful execution |
| E2 canonical result core | E2 typed proof object | query-result proof input and packet compiler input | Reference detached result digest; do not add a self-digest |
| E2 manifest-last proof bundle | E2 publication authority | batch/bundle ancestor | Preserve result and constituent digests; new manifests may reference old bundles |
| `origins/catalog.json` | bounded E2 execution configuration | live-refresh capability route | Keep exact route authority; Atlas identity cannot broaden it |
| `atlas/registry.json` | E3 canonical origin identity and orthogonal state | origin/policy join and query filtering | Import origin ID and independent state dimensions without converting them to one maturity number |
| Atlas admission dossiers and decisions | E3.2 human-review evidence | packet admission prerequisites and policy explanation | Reference decision and evidence digests; pending remains pending |
| Atlas interface declarations | E3.2 descriptive metadata | capability-candidate packets and profiler input | `descriptive_only` remains non-executable |
| Atlas capability/effect declarations | E3.2 descriptive or admitted state | capability graph and hard effect filters | Only the existing E2 `public_read` admission can become executable |
| Access and provisional offer metadata | E3.2 descriptive metadata | access/offer packets and economic planning inputs | Provisional observations cannot become prices, permissions or payment authority |
| Operational economics | E3.2 measurements | historical economic events | Preserve measured-versus-estimated classification and evidence |
| Publisher-readiness signals | E3.2 descriptive metadata | publisher integration state | Preserve declared/observed/inferred origin and standard status |
| Controlled origin fixture | E2 conformance evidence | packet/query conformance only | Mark `test_fixture`; exclude from public-origin and live-value counters |
| Generated E2/E3 public metrics | derived evidence | reconciliation fixtures | Recompute from canonical sources; never import a derived counter as authority |

## Canonical ownership after migration

```text
filesystem CAS
  owns immutable canonical bytes and proof artifacts

Atlas source directories and registry
  own origin identity, human admission, policy and descriptive capability state

PostgreSQL semantic log
  owns transactional admission order, detached identities, deltas and current heads

materialized views
  own no independent facts; they are rebuildable projections
```

The database does not replace repository-bound decisions or the CAS. The
repository does not become a high-frequency packet log. Both are bound by
digests and explicit migrations.

## Initial migration phases

### M0 — Inventory and dry run

- Verify every referenced source artifact and its digest.
- Emit a deterministic plan sorted by artifact class and identifier.
- Report missing, conflicting or unsupported inputs without writing state.
- Reconcile the two public E2 origins and the controlled fixture against the
  canonical Atlas identity vocabulary.

### M1 — Identity and contract import

- Import origin identities, orthogonal state and immutable evidence references.
- Import E2 operation-contract, adapter and mapping identities.
- Do not import pending policy as completed or descriptive capability as route
  authority.

### M2 — Packet compilation

- Compile `observed_native` packets first.
- Compile semantic packets only where an exact mapping artifact exists.
- Record unresolved optional content without inventing a value.
- Fail closed on any missing required observation, contract, adapter, mapping
  or canonical byte artifact.

### M3 — State and delta derivation

- Derive current heads using deterministic temporal and digest ordering.
- Emit semantic/canon deltas for reinterpretation separately from origin
  deltas.
- Rebuild all materializations from the log and compare digests.

### M4 — Read cutover

- Run existing E2 results and E3.3 queries side by side.
- Compare native values, semantic values, provenance and unresolved behavior.
- Keep E2 routes available as a bounded live-refresh path.
- Cut over no public interface until conformance and founder review pass.

## Required migration invariants

- Original SHA-256 identities are byte-for-byte stable.
- Every target row has a verified canonical artifact or an explicit relational
  identity source.
- A failed batch commits no packet, head, delta or outbox row.
- Re-running a successful migration creates no duplicate semantic identity.
- Source-native vocabulary and lexical values survive unchanged.
- Fixture data cannot enter public-origin or production materializations.
- Origin, semantic and canon changes remain distinguishable.
- Rollback changes read routing only; it never deletes admitted log history.
- No migration tool receives arbitrary network capability.

## Reconciliation fixtures

The first implementation must cover:

1. `twirx-org` publisher-authored status and risk operations;
2. `api-worldbank-org` bounded public indicator operation;
3. `controlled-origin-lab-fixture` as a non-public conformance source;
4. one resolved field and one optional unresolved field;
5. one mapping-only reinterpretation that emits no origin delta;
6. one canon-version change that does not alter original packet bytes.

## Unresolved migration risks

- PostgreSQL is not installed and the target VPS storage is degraded.
- S1 canonical packet/delta/query conformance is not yet implemented.
- Existing E2 semantic closure requires an explicit trust-lane translation
  table before bulk compilation.
- Retention and public-disclosure classifications need founder-approved policy
  values before any external corpus is admitted.
- The published draft PR containing the earlier route-centric E3.3 design must
  not be history-rewritten; this architecture should supersede it in a new PR.
