# TWIRX Ontology Fabric 0.1

**Authority:** Normative E4.0 design

**Status:** Implementation candidate with admitted local World State and
Opportunity corpus evidence; no complete E4 merge or Opportunity deployment admission

The Ontology Fabric extends the Semantic Data Plane without replacing semantic
packets. The protocol is language-neutral. Go, restricted C, JSON authoring
files and generated SDK types are non-normative implementations and carriage
formats.

Normative files:

- `SEMANTIC_FRAME_0_1.md` defines proof-linked composite frames.
- `ONTOLOGY_MODULE_0_1.md` defines bounded versioned modules and semantic diff.
- `MAPPING_CLAIM_0_1.md` defines scoped, reviewable native-to-semantic claims.
- `SEMANTIC_UNIVERSE_0_1.md` defines domain packages and utility accounting.
- `OPPORTUNITY_SOURCE_PILOT_0_1.md` documents the bounded, source-specific
  E4.5 acquisition and offline compilation boundary.
- `schemas/cddl/semantic-data-plane.cddl` defines canonical deterministic-CBOR
  layouts and bounds.

Canonical objects never contain their own digest. Missing required proof fails
closed. Native source statements remain in contributing packets and are never
rewritten by a frame, mapping, module, model or generated artifact.
