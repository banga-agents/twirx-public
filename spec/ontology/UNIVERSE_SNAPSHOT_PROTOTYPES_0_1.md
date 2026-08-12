# Universe Snapshot prototypes 0.1

**Authority:** Explanatory implementation profile under ADR 015

**Status:** Controlled benchmark candidate; not a public snapshot format

## Purpose

The E4 prototype compares three immutable derived layouts over the same exact
canonical Semantic Frame bytes:

1. a native dictionary/posting layout for bounded exact retrieval;
2. a column-oriented scan layout representing the minimum analytical
   baseline that can be measured without adding a runtime dependency;
3. a compact binary dictionary/posting segment with lazy canonical-frame
   traces and read-only file mapping.

Neither JSON prototype artifact is normative protocol authority. Frame and
packet identity remain detached SHA-256 over their exact deterministic-CBOR
cores.

## Native layout

The native artifact contains:

- sorted universe and frame-type dictionaries;
- digest-sorted canonical frame entries;
- exact native keys, lanes and lifecycle columns;
- sorted postings for universe, frame type, native key and canonical typed
  slot value;
- exact canonical frame bytes for bounded trace.

Open verifies the artifact digest, every frame digest, canonical frame parse,
all duplicated columns, ordering, bounds and a complete recomputation of the
posting index.

## Column-oriented baseline

The analytical baseline stores equal-length columns for digest, canonical
frame bytes, universe, frame type, native key, trust lane, lifecycle and slot
values. Open verifies every row against its canonical frame.

This is not DuckDB, Parquet or an external sidecar. A real sidecar comparison
requires a separate dependency and process-boundary ADR with founder approval.

## Compact native segment

The implementation identifier is `tw.universe-compact-segment/0.1`; the first
eight file bytes are `TWUXS001`. This is a language-neutral binary profile,
not a Go object encoding. All integers are unsigned big-endian except posting
deltas, which use unsigned varints.

The fixed 64-byte header binds:

- frame, dictionary and posting counts;
- dictionary, entry, body, posting and end offsets;
- a zeroed reserved word for fail-closed future evolution.

It is followed by:

- a strictly sorted length-prefixed UTF-8 dictionary;
- digest-sorted fixed-size frame entries containing the detached SHA-256,
  body offset and length, and dictionary-local universe/type identifiers;
- contiguous canonical Semantic Frame CBOR bodies;
- strictly sorted posting keys and delta-coded frame identifiers.

Open requires an externally supplied expected SHA-256 over the entire exact
segment. It then checks section contiguity, all bounds, every canonical frame
digest and duplicated identity column, and a complete recomputation of all
postings. A posting count may never exceed the frame count. File open rejects
symlinks and non-regular files; the admitted runtime maps the verified file
read-only. Trace returns a copy of exact canonical frame bytes. The integrity
visitor walks every admitted frame in canonical digest order and supplies a
copy to its caller; it exists for complete release verification and is not an
unbounded public query.

The compact segment is a derived query index. Its local integer identifiers
have no protocol authority and are never exposed as concept, packet, frame or
origin identities. The canonical frame CBOR and its detached digest remain
authoritative.

## Query profile

The controlled prototype accepts at least one of:

- exact universe;
- exact frame type;
- exact native key;
- exact role plus canonical typed value.

Result limits are mandatory and bounded to 1–1,000. Queries do not accept SQL,
paths, URLs, scripts, models or network destinations.

## Publication boundary

Before a Universe Snapshot can be released publicly, the selected layout must
be carried by a manifest-last Semantic Snapshot extension, independently
verified, restored from off-host storage and opened by a read-only runtime.
The bounded E4.2 World State candidate exercises manifest-last publication and
Go/restricted-C constituent verification. E4.5 additionally builds a combined
World State plus Opportunity segment containing 83,122 real source-derived
frames. It remains a local founder-review release candidate rather than a
deployed replacement for the protected public FUTO baseline.

## Canonical artifact segment

Large packet and mapping collections use the derived publication container
`tw.artifact-segment/0.1` with magic `TWAS0001`. Its fixed header declares one
artifact kind, entry count, index start, body start and exact end offset.
Fixed-size index entries bind the detached SHA-256, body offset, body length
and a zero reserved word. Entries are strictly digest-sorted, bodies are
contiguous and trailing bytes are rejected. Each contained body remains a
canonical protocol object with its own identity; the container introduces no
semantic or normative authority.
