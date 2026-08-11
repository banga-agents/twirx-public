# E3.3 read-only Semantic Snapshot conformance

The executable corpus is generated entirely from committed inputs so it does
not duplicate large proof bundles in Git.

`scripts/test-semantic-snapshot.sh` performs the shared flow:

1. build the snapshot twice from the same source revision and creation time;
2. require identical detached snapshot IDs;
3. verify the manifest and every artifact in Go;
4. verify the canonical manifest and all generated packet bytes with the
   independent restricted-C implementation;
5. execute the fixed population and two-origin query fixtures in `examples/`;
6. compare achieved counts with `expected-baseline.json`;
7. require zero runtime network requests and default fixture exclusion.

Unit tests add malformed and adversarial coverage for unknown JSON members,
digest substitution, artifact tampering, self-consistent but false proof-index
relationships, fixture leakage, live-refresh authority, oversized canonical
objects and non-loopback service binding.

`scripts/stress-semantic-snapshot.sh` is the bounded operational workload. It
uses a proxy-disabled, literal-loopback, destination-pinned Go client, checks
every canonical query/result identity, records integer latency and process
resource measurements, and requires identical status after restart.

The baseline records achieved facts only. Funding targets remain separate in
the snapshot build report and are not conformance expectations.
