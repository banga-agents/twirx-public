# Trust transitions and ranking ordinals 0.1

Status: normative for E3.3 S1 and later planner conformance

These tables prevent an implementation from inventing a hidden trust or
ranking order. They are comparison dimensions after hard query constraints,
not claims of objective truth. They MUST NOT be summed into one undisclosed
score.

## Trust-lane transition table

Packets are immutable. A transition means a new packet plus the listed
semantic delta while source-evidence identity remains equal.

| From | To | Required mapping state | Allowed semantic delta |
| --- | --- | --- | --- |
| `observed_native` | `provisional_semantic` | `candidate` | `mapped` |
| `provisional_semantic` | `provisional_semantic` | `candidate` | `remapped`, `narrowed`, `broadened`, `disputed` |
| `provisional_semantic` | `attested_semantic` | `reviewed` | `attested` |
| `attested_semantic` | `attested_semantic` | `reviewed` | `remapped`, `narrowed`, `broadened` |
| `attested_semantic` | `provisional_semantic` | `candidate` or `disputed` | `de_attested`, `disputed` |
| `provisional_semantic` | `observed_native` | `none` | `disputed` |
| `attested_semantic` | `observed_native` | `none` | `de_attested` |

Every other lane transition is rejected in 0.1. A revoked mapping cannot
appear in an admitted semantic packet. Revocation creates an explicit new
packet in a permitted lower lane and retains the prior packet and delta.

Publisher attestation affects extraction/mapping evidence only. It does not
permit a transition that lacks the required mapping, semantic closure and
human/canon decision artifacts.

## Delta class table

| Class | Source evidence | Kinds |
| --- | --- | --- |
| `origin` | added, removed or changed | `added`, `modified`, `withdrawn`, `restored`, `source_retracted` |
| `semantic` | unchanged | `mapped`, `remapped`, `narrowed`, `broadened`, `disputed`, `attested`, `de_attested` |
| `canon` | unchanged | `module_added`, `module_superseded`, `mapping_superseded`, `closure_changed` |

An origin delta cannot be emitted from a mapping/canon change. A semantic or
canon delta cannot be emitted when its before/after source-evidence digests
differ.

## Deterministic ordinal dimensions

Higher values are better after the query's hard filters unless the table says
otherwise.

### Trust lane

```text
observed_native         1
provisional_semantic    2
attested_semantic       3
```

This is proof/mapping maturity, not source authority.

### Mapping status

```text
disputed       0
none           1
candidate      2
reviewed       3
revoked        excluded
```

For a native-only query, `none` is not penalized; the mapping dimension is
inactive. A query requiring reviewed semantics excludes `none`, `candidate`,
`disputed` and `revoked` before Pareto construction.

### Freshness

```text
unknown    0
stale      1
current    2
```

Archive observations are historical. They cannot receive `current` merely
because their archive retrieval was recent.

### Proof completeness

```text
packet     1
field      2
bundle     3
```

### Source authority tier

The packet's `authority-class` remains an auditable vocabulary identifier. A
versioned policy maps it to one of these query-local tiers:

```text
unclassified             0
tertiary_aggregator       1
independent_secondary     2
publisher_primary         3
publisher_attested        4
```

Government status, popularity, sponsorship, price and TWIRX operator revenue
do not automatically change this tier. The query/ontology context selects the
applicable authority policy, and the plan publishes its digest.

### Uncertainty (lower is better)

```text
attested reviewed mapping                   0
deterministic native statement              1
reviewed but non-attested semantic mapping  2
candidate mapping                           1,000,000 - confidence-millionths
unknown or disputed                         1,000,000
revoked                                     excluded
```

Candidate confidence never changes trust lane or canon status.

## Pareto invariants

- semantic coverage, authority, proof, freshness and historical reliability
  are maximized;
- latency, monetary cost and uncertainty are minimized;
- sponsorship and funding class are absent from dominance and tie-breaking;
- no missing measurement is silently assigned a favorable value;
- ties use origin ID bytes, operation/materialization ID bytes and packet
  digest bytes after the named preference dimensions;
- the plan exposes every dimension, exclusion and tie-break used.
