# TWIR Core 0.1 — E2 profile

**Authority:** Normative for E2 conformance

**Status:** Specified and implemented by the E2 contract loader

This deliberately small profile defines primitive values, records, optional
fields, lists, resources, operations, typed errors, effects, evidence
requirements, native references, semantic references, semantic-module closure,
and explicit `resolved`/`unresolved` field state.

Only the `read` effect is admitted in Genesis. An operation has finite typed
input, a finite result record, bounded values, and declared error codes. Every
output field declares its source-native term and locator before its semantic
term, transformations, and mapping relation. A semantic view is additive and
never replaces the native statement.

The canonical project contract is `contracts/e2/contracts.json`. It is input
to generators, not a claim that JSON or the Go loader is normative. The
deterministic operation artifact encoded by the CDDL profile and shared
conformance vectors controls interoperability.

E2 intentionally excludes unions, arbitrary recursion, write effects,
capabilities, payments, remote public MCP, arbitrary origins, browser
discovery, and model-generated promotion.
