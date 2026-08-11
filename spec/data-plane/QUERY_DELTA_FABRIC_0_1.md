# Query, result and subscription fabric 0.1

**Authority:** Normative for E3.3 query and subscription conformance

**Status:** S1 contract codecs and verification implemented as a review
candidate; S6 planner, state query and subscription delivery remain pending

## Typed query authority

Execution authority is the canonical `semantic-query-core`, never natural
language. A model or UI may propose a query, but a client or explicit policy
must validate and submit the typed object. No field accepts a URL, SQL, code,
MCP server address, browser instruction, credential or payment instruction.

A query binds:

- selected semantic concepts;
- exact subjects and bounded context filters;
- temporal mode;
- maximum ontology depth/path cost and allowed edge status;
- allowed sources and minimum source diversity;
- trust lanes and mapping states;
- freshness behavior;
- monetary/funding constraints;
- conflict behavior;
- materialized-state and bounded live-refresh permission;
- proof requirements;
- an explicit preference policy;
- result, packet and proof-byte ceilings.

Live refresh does not grant network authority. It merely allows the planner to
request an already admitted E3.2 work order for an already admitted origin and
route. If no such route exists, the result is stale or unresolved according to
the canonical query. Browsers remain prohibited in E3.3.

## Planning order

A conforming planner performs these stages in order:

1. Validate canonical bytes and all bounds.
2. Resolve exact concept and identity references.
3. Search native lexical terms where the query permits mapping candidates.
4. Expand reviewed ontology edges to the declared maximum depth/cost.
5. Optionally add vector/model candidates in a separate provisional class.
6. Find materialized state and admitted capabilities.
7. Apply hard source, trust, freshness, effect, proof, cost and execution
   constraints.
8. Remove dominated candidates to construct a Pareto frontier.
9. Apply the named caller preference policy with deterministic tie-breakers.
10. Read materialized state or create bounded live-refresh subplans.
11. Merge rows while preserving conflicts and source identity.
12. Publish the plan, result, proof references and economic event.

An optional candidate generator cannot skip exact retrieval, hard filtering or
explicit policy selection.

## Pareto dimensions

Each candidate is described by deterministic integer/ordinal dimensions:

```text
semantic coverage
source authority
proof completeness
freshness
historical reliability
latency estimate
monetary cost
uncertainty
```

A candidate is dominated when another is no worse in every declared dimension
and strictly better in at least one. Dominated candidates are excluded before
the preference policy. Implementations publish dimension values, exclusions
and tie-breaks in the plan artifact.

The first deterministic tie-break order is:

```text
preference-specific dimensions
origin_id bytes
operation or materialization ID bytes
packet digest bytes
```

S1 conformance must freeze the ordinal tables for authority, proof, freshness,
mapping state and uncertainty before planner implementation begins. Sponsoring
an origin or buying a commercial offer is never a ranking advantage.

## Query result

The canonical result is `semantic-query-result-core`. Each resolved or
explicitly unresolved row preserves:

- native term, locator and lexical value;
- optional semantic term and typed value;
- origin and packet identity;
- observation identity and trust lane;
- observation time.

The result binds the exact query and plan digests, snapshot sequence,
preference policy, conflicts, proof artifacts and separately measured economic
event. It contains no self-digest.

```text
query_result_digest = SHA-256(exact canonical query-result-core bytes)
```

Rows are sorted by canonical subject, predicate, origin, observation time and
packet digest. Proof artifacts are sorted by digest. A partial result must say
which bounds, stale states, unavailable origins or unresolved mappings caused
partiality.

## Conflict preservation

TWIRX does not silently average or overwrite incompatible source statements.
The query chooses one of:

- `preserve_sources`: return distinct rows and a conflict group;
- `group_equivalent`: group only under an admitted exact equivalence rule;
- `reject_conflict`: return an explicit unresolved conflict.

Even when a caller preference selects one row for presentation, the result
records `caller_policy_selected` and retains every competing packet digest in
the conflict group. Selection is not a truth judgment.

## Subscription

A canonical subscription binds a previously admitted query digest, requested
delta classes/kinds, delivery mode, durable cursor, rate cap, proof level and
optional expiry. The public Genesis transports are SSE and polling.

The cursor is a monotonically increasing admitted-log sequence. It is not a
timestamp. Clients reconnect with `resume-after-sequence`; the server either
continues after that durable event or returns an explicit cursor-expired state
with the earliest retained cursor. It never silently resumes from current.

SSE is a delivery view over the transactional outbox:

```text
packet/head/delta/outbox transaction commits
    -> dispatcher claims durable outbox row
    -> public event includes sequence + delta digest + proof reference
    -> client resumes from sequence
```

Database `LISTEN/NOTIFY` may wake a dispatcher but is not the durable event
record. A service restart cannot lose a committed event.

## Snapshot consistency

One query runs against one declared snapshot sequence. Materialized rows newer
than that sequence are excluded. A live refresh that commits during the query
is either included through a new explicit snapshot/plan or excluded from the
current result; implementation timing cannot make the result ambiguous.

## Materializations

A materialized semantic view is defined by a canonical query/definition,
canon version and admission policy. Its manifest records the through-sequence,
complete sorted packet set, output artifact, row count and build time.

Initial views are limited to founder-admitted definitions such as:

```text
latest country indicators
recent research publications
current public commercial capabilities/offers
```

The offer view represents observed publisher terms with source and validity; it
does not execute a purchase or claim that an offer remains available after its
freshness boundary.

## Failure behavior

Fail closed on:

- non-canonical queries, unknown fields or unsupported versions;
- unsorted/duplicate selectors, filters, lanes, sources or proof entries;
- contradictory time bounds or a required freshness bound that cannot be met;
- a live-refresh request without E3.2 route authority;
- an effect other than admitted public read;
- cost greater than the query maximum or a currency mismatch;
- missing packet/native/proof fields in a resolved row;
- a plan that hides dominated-candidate reasoning or applies sponsorship to
  rank;
- cursor gaps, outbox/state inconsistency or unknown retained cursor range;
- a materialization whose manifest does not reproduce from immutable history.
