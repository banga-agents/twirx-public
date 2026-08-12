# E4 source acquisitions

Each directory below is a bounded, source-specific acquisition. Work orders
are derived from reviewed contracts and human policy artifacts. Evidence is
stored and manifest-complete before any E4 importer runs.

The directory is not a crawler frontier. It contains no arbitrary-URL input,
continuous scheduler, browser route, authenticated route, or write action.

`world-bank-e2-matrix` is a one-time manual acquisition derived entirely from
the existing E2 contract allowlist: three countries, two indicators and six
single years. Its execution summary records a resumed invocation as 28 network
requests plus eight already-complete verified spools. Across the initial and
resumed invocations, the directory contains exactly 36 independently verified
response spools. One non-200/non-JSON response remains immutable rejected
evidence and is excluded from compilation.
