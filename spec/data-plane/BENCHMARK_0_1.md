# E3.3 benchmark and value-evidence methodology 0.1

**Authority:** Normative measurement methodology for E3.3 S7-S10 claims

**Status:** Design; no performance result claimed

## Objective

E3.3 must measure whether compiling public representations into reusable,
proof-bearing semantic state creates meaningful agent value. Parser-only
nanoseconds are insufficient. Measurements must include acquisition,
compilation, state admission, query, proof, reuse, freshness and cost.

## Compared routes

For the same frozen task and origin snapshot, measure where applicable:

1. search/discovery plus agent interpretation;
2. headless browser plus agent interpretation;
3. publisher-native API;
4. TWIRX bounded live compiled route;
5. TWIRX materialized semantic query.

Browser/model routes run outside the trusted TWIRX runtime in a controlled
benchmark harness. They cannot write canon or production state. If a route is
not applicable or cannot be reproduced, report it as excluded rather than
inventing a comparison.

TWIRX is not expected to beat a publisher-native API at merely calling that
API. The tested value is shared semantics, cross-origin joins, interpretation
reuse, provenance, subscriptions and a stable interface across origins.

## Required metrics

### Work and latency

- origin requests, redirects and transferred compressed/decompressed bytes;
- acquisition, evidence write, parse, extraction, packet assembly and
  verification latency;
- database admission and materialization update latency;
- query planning, materialized read, proof assembly and total latency;
- delta publication/freshness lag;
- CPU milliseconds, peak resident memory and storage growth;
- PostgreSQL WAL bytes, temporary bytes, rows/packets/deltas touched and lock
  wait;
- subscription delivery and resume latency.

### Value ratios

```text
semantic compression ratio = raw bytes retrieved / task-relevant typed bytes

context compression ratio =
  tokens required for page/browser interpretation /
  tokens required for typed result plus schema

interpretation reuse factor =
  verified agent queries served / compiler or semantic interventions

browser avoidance rate =
  known-origin requests served without browser / all known-origin requests

semantic bandwidth =
  proof-bearing semantic units / (elapsed time * infrastructure cost)

freshness lag =
  semantic-delta publication time - origin-change observation time
```

Every numerator and denominator is published. Division by zero is `undefined`,
not infinity. Ratios do not mix warm/cold runs, different tasks, different
source snapshots or different proof requirements.

### Economics

- human review and adapter-maintenance seconds;
- requests, transfer, CPU, memory, storage and proof bytes;
- measured or explicitly modeled infrastructure cost and method version;
- sponsor/funding class separate from origin authority and planner rank;
- effective semantic cost per invocation:

```text
(onboarding cost + accumulated maintenance cost) / verified invocations served
+ measured query cost
```

## Corpus and workload

S7 freezes a manifest containing origin IDs, exact representation/observation
digests, packet/batch/canon versions, query cores, expected resolution state and
route applicability. The minimum funding demonstration target is an acceptance
goal, not current evidence:

```text
500 cataloged origins
100 completed policy decisions
100 safely profiled origins
50 immutable observed origins
25 native schemas
12 deterministic adapters
8 live read-only origins
100,000 admitted semantic packets
3 materialized cross-origin views
1 semantic delta stream
```

Performance evidence must separately state the actual achieved counts.
Controlled fixtures, archive observations, simulated deltas and live publisher
evidence have distinct labels and counters.

## Run classes

- **cold:** process/service cache cleared by a documented non-destructive
  method; no claim about kernel or device cache unless actually controlled;
- **warm:** exact workload repeated after one complete run;
- **steady:** fixed concurrency for at least the declared measurement interval;
- **change burst:** deterministic packet/delta batch applied while queries and
  subscriptions continue;
- **rebuild:** materialization dropped and recreated from immutable history;
- **recovery:** isolated restored state serves the corpus with public delivery
  disabled.

At minimum report median, p95, p99, maximum, sample count, failures, host,
kernel, CPU, memory, storage, PostgreSQL version/config digest, commit, corpus
digest, concurrency and exact command. Averages alone are insufficient.

## Stress and acceptance dimensions

- 1, 8, 32 and measured-safe concurrent query clients;
- bounded ingestion overlapping reads;
- maximum allowed query result/proof sizes;
- ontology cycles and maximum allowed expansion depth;
- one slow/stale origin without global queue starvation;
- subscriber reconnects, duplicate delivery and retained-cursor boundary;
- database restart and killed admission transactions;
- storage soft/hard alarms, WAL/archive lag and materialization backlog;
- no-browser/no-model deterministic query baseline;
- optional vector/model feature disabled baseline.

Exact throughput thresholds are set after the remediated VPS baseline and S2
prototype. They cannot be retrofitted after observing results without
recording the change.

## Browser comparison constraints

The browser comparison uses a pinned browser/container, fixed page snapshots
or declared live capture window, no unrelated extensions, recorded request log,
and a measured task-output equivalence rule. It reports startup, requests,
bytes, CPU, memory, latency, agent-input tokens, output correctness and evidence
completeness. It cannot generalize one fixture into a universal multiplier.

## Reproducibility and publication

- Normal conformance and unit tests remain fully offline.
- Live benchmarks require founder-approved origins and work orders.
- Raw machine-readable measurements and a human report are digest-bound.
- Failed and excluded runs remain in the report.
- No claim is copied to the website until generated evidence at the exact
  public commit supports it.
- Independent reruns state their host and differences; they do not overwrite
  the original record.
