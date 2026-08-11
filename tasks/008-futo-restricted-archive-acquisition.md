# Task 008: FUTO restricted archive acquisition

## Status

Implemented and exercised for one steward-approved sealed work order. The
admitted RFC Editor acquisition contains two exact historical captures, four
fixed-provider network requests, and a final manifest written last. The
scheduler remains disabled and no broader retrieval is authorized.

## Objective

Execute one sealed, human-approved Common Crawl plan without adding network
authority to the offline importer or public runtime. Preserve the index and
range response before parsing, reconcile every resulting artifact, and label
all accepted evidence as historical archive observation rather than current
publisher state.

## Invariants

- only `index.commoncrawl.org` and `data.commoncrawl.org` are eligible network
  destinations;
- no caller-selected URL, host, range, collection, port, scheme or redirect;
- all request authority derives from a validated sealed work order and exact
  returned capture metadata;
- public-address DNS and destination revalidation apply to every connection;
- redirects are disabled and exact `GET`, URL, `206` and `Content-Range`
  relationships fail closed;
- bounded raw index and compressed-range bytes are stored before their
  respective parsers run;
- the offline importer remains network-incapable;
- the final acquisition manifest is published last and independently
  reverified;
- no semantic packet, delta, view or canon admission occurs in this task;
- no scheduler, arbitrary URL, browser, model, payment, authenticated action,
  production PostgreSQL or Meridian mutation is added.

## Acceptance

The implementation and real-acquisition gates pass only for
`rfc-editor-futo-history`. The decision artifacts and exact selected captures
are committed, the bounded plan was explicitly run, and the final acquisition
manifest independently verifies. The first attempt failed closed after raw
evidence retention and produced no final manifest; it remains separately
identified as a non-admitted diagnostic attempt. A partial directory without a
verified final manifest is never an admitted acquisition.
