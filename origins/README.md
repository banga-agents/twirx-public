# E2 admitted origin catalog

The catalog is reviewed project configuration, not a public discovery index.
Each entry binds `registry_id` to the canonical identity in
`atlas/registry.json`; startup fails closed if the host, publisher, scope, or
execution alias disagrees. The execution catalog still owns its narrower E2
route and input admission. Canonical identity binding does not complete Atlas
policy review or activate its scheduler.

Clients submit `origin_id`, `operation_id`, and bounded typed input. They never
submit a destination URL. The service builds an endpoint from the reviewed
template, contract allowlists, and fixed HTTPS hostname, then applies the
network policy again at resolution and redirect time.

Recorded fixtures support deterministic offline replay. A fixture is evidence
of one recorded representation, not a statement that the provider or its
content is objectively true. External fresh mode is separately rate-limited
and may be disabled without removing replay.

The World Bank Indicators API was admitted for one small E2 operation after
reviewing its official v2 API overview and basic call documentation. The API
is documented as not requiring authentication. E2 permits only three country
codes, two indicator codes, six years, one result per call, and the official
`api.worldbank.org` HTTPS host.
