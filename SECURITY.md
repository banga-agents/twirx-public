# Security Policy

## Status

TWIRX Genesis includes a bounded catalog-only Live Provenance Lab candidate.
It is **not** approved for public multi-tenant arbitrary-URL service deployment.

## Reporting a vulnerability

Do not disclose suspected vulnerabilities in a public issue, discussion, social post, or pull request.

Once the GitHub repository is created, enable **Private vulnerability reporting** and use the repository's **Security** tab to submit reports. Until that channel is configured, keep the report private and contact the maintainer through an existing trusted private channel.

A useful report includes:

- affected commit or release;
- component and trust boundary;
- reproduction steps or fixture;
- expected and observed behavior;
- potential impact and blast radius;
- whether credentials, canonical state, or cross-origin access are involved;
- any proposed mitigation.

Do not test against third-party websites without authorization. Use the controlled test origin and local adversarial fixtures.

## Security invariants

1. Public fetch policy rejects private and reserved destinations.
2. Redirects are revalidated.
3. Response bytes, time, redirects, and ports are bounded.
4. Evidence is stored by digest and reverified before extraction.
5. Adapters cannot contact the network during offline extraction.
6. Native statements are preserved beside semantic views.
7. Missing required fields fail closed.
8. Optional missing fields become `unresolved` rather than fabricated.
9. The independent C verifier has no network or registry authority.
10. Genesis permits only idempotent read operations.
11. No model output or browser worker can write canonical state.
12. No seed phrase, private key, API token, or unredacted personal invoice enters the repository.
13. Privileged JSON parsing rejects duplicate keys and trailing values under explicit depth, scalar, container, token, and byte limits.
14. Canonical metadata and extraction results are published atomically; interrupted writes do not expose partial final artifacts.
15. Public Lab clients identify reviewed origins and operations; they cannot supply a destination URL.
16. E2 proof bundles are admitted only after the final manifest exists and every listed regular file validates.
17. The Lab application binds to loopback behind the public TLS edge and runs as a dedicated unprivileged service account.
18. Atlas candidate selection is never a fetch allowlist and cannot contribute to cataloged or later technical counts.
19. Catalog, policy, technical, publisher, health, adapter-trust, and mapping-trust state remain orthogonal and require their own evidence.
20. The E3.0 Atlas API is GET-only, bounded, loopback-only, and has no HTTP client.
21. A policy-bound Atlas record binds the exact policy-set digest and cannot disagree with its policy fields.
22. Pending review and unknown, unreachable, redirect-limit, and not-observed robots states cannot authorize live retrieval.
23. The E3.1 frontier is deterministic and dry-run only; it contains no destination URLs and grants no egress authority.
24. The Observatory fixture worker accepts only literal `127.0.0.1` robots jobs, rejects redirects, and stores evidence before parsing.
25. Fixture-worker output cannot promote Atlas policy, technical, publisher, health, adapter-trust, mapping-trust, or canonical state.
26. Controlled fixtures are explicitly scoped as `test_fixture` and excluded from all Genesis-500 public-origin counters.
27. E2 execution entries fail closed when their canonical registry ID, host, publisher, or scope binding disagrees.
28. The Atlas admission queue and dry-run frontier each cover every selected origin exactly once; missing dossiers and catalog records become explicit blocked states, never implicit permission.
29. The Atlas stress client accepts only an exact literal-loopback HTTP origin with an explicit port, disables proxies and redirects, and pins every connection to that address.
30. A Semantic Snapshot is unavailable until its canonical manifest is written last and every declared regular artifact verifies by size and digest.
31. Snapshot import rejects path escape, symlinks, hard links, devices, FIFOs, sockets, count drift, sequence gaps, duplicate packets, proof-index mismatch, and public views containing fixtures.
32. The snapshot builder invokes only committed E2 replay evidence and performs zero origin-network requests.
33. The snapshot runtime has no refresh or state-write path, binds only to a literal loopback address, and rejects live-refresh query authority.
34. Snapshot HTTP requests, headers, processing time and concurrency are bounded; unknown JSON fields, duplicate keys and trailing values fail closed.
35. Targets, candidate identities, public compiled origins, fixture origins, packets and current claims remain distinct generated counters.
36. Snapshot proof downloads resolve only an admitted packet digest and exact proof-index artifact name; visitor input never becomes a filesystem path.

## Supported security baseline

The repository should be built with current supported toolchains. The CI target should include:

- Go tests, race detector where applicable, vet, and fuzzing;
- GCC and Clang builds of the C verifier;
- AddressSanitizer and UndefinedBehaviorSanitizer;
- static analysis;
- dependency and secret scanning;
- reproducible artifact metadata;
- adversarial conformance vectors.

## Public-deployment gate

Before exposing arbitrary URLs to untrusted public users, the project requires:

- separate worker and control networks;
- dedicated egress proxy and DNS control;
- process, container, or microVM isolation;
- no worker access to PostgreSQL, registry signing keys, cloud metadata, or control credentials;
- per-tenant quotas and origin budgets;
- audit logging and anomaly detection;
- tested revocation and recovery;
- external security review.
