# Genesis-500 candidate selection

`selection.json` freezes the exact E3 universe before implementation. Every
entry has `catalog.state: candidate`. The file deliberately uses null publisher
and jurisdiction hints and empty language hints so a curated name cannot
masquerade as catalog, policy, technical, publisher, health, adapter-trust, or
mapping-trust evidence.

The candidate's canonical origin is a selection target, not a claim that the
host is reachable, belongs to a particular publisher, permits access, exposes
a useful representation, or satisfies a quota beyond its chosen domain-family
slot. Those facts are established in later evidence packages.

Progress requires independent records for every state dimension defined by
ADR 006. Catalog review does not complete policy review. Technical compilation
does not establish publisher approval or live health. Adapter conformance does
not establish mapping trust.

No program should use `selection.json` as a network allowlist.

The file's initial SHA-256 is recorded in
`reports/e3-genesis-500-selection.md`. Future changes must preserve exactly
500 unique entries, pass the quota validator, explain replacements in that
report, and must not rewrite earlier review evidence.
