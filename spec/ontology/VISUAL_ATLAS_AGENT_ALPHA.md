# Visual Atlas Agent alpha execution contract

**Authority:** Explanatory E4 implementation profile

**Status:** Deterministic controlled demonstration implemented; public UI and
real-universe corpus pending

## Authority flow

```text
curated question
  -> visible typed query
  -> bounded immutable snapshot query
  -> canonical Semantic Frames
  -> frame slots
  -> packet digests
```

Natural language is not execution authority. The E4 controlled demonstration
uses a reviewed scenario registry and no model. A future model MAY propose the
same typed query, but schema validation and an admitted deterministic runtime
remain the only execution authority.

## Initial scenarios

- controlled World State inspection;
- controlled Opportunity inspection;
- Research discovery, explicitly unresolved until data admission;
- Security discovery, explicitly unresolved until data admission;
- Agent Economy capability discovery, explicitly unresolved until data
  admission.

The controlled results MUST state `evidence_class: test_fixture`,
`current_claims_made: false` and `fixture_counted_public: false`.

The controlled investigation path coordinates the World State and Opportunity
scenario queries. It is not a semantic join and MUST NOT infer equivalence or
a relationship across the two frames. Its purpose is to prove deterministic
multi-universe planning while retaining each frame's independent proof path.

## Required plan evidence

Every execution exposes:

- scenario and question;
- concepts and universe;
- exact typed query and detached query digest;
- selected immutable layout and available frame count;
- result limit;
- network, browser and live-source call counts;
- model authority and deterministic execution authority.

Every returned frame exposes its detached digest, native key, frame type,
trust lane, completeness, slots, values, mapping IDs, conflict state and exact
packet digests. Full packet/source bundle retrieval remains a separate proof
operation and must fail closed if unavailable.
