# Public evidence profile

TWIRX preserves exact source representations privately for verification, but
does not assume that access through a public archive grants permission to
republish every captured byte.

The FUTO public source export therefore includes:

- TWIRX source code, specifications and conformance fixtures;
- steward-approved origin policy decisions;
- acquisition work orders and bounded capture metadata;
- representation and artifact digests;
- source-native semantic packets and deltas;
- reproducible verification and acquisition commands;
- reports that distinguish archive observations from current publisher
  statements.

It excludes:

- retained third-party homepage bodies;
- compressed WARC records and byte-range responses containing those bodies;
- raw Common Crawl index responses;
- failed-attempt raw acquisition artifacts.

The excluded artifacts remain available only in the private evidence store.
Their digests and provenance references remain public so an independently
authorized reviewer can reacquire and compare them. Public Query Lab routes
deny raw proof-artifact downloads.

Two integration tests that rebuild the real archive-derived packet and delta
run normally whenever the private evidence is present. In the sanitized public
tree they report an explicit skip. Setting
`TWIRX_REQUIRE_PRIVATE_ARCHIVE_EVIDENCE=1` converts absence into a hard failure.
All controlled parser, verifier, query, conformance and adversarial tests remain
mandatory in the public tree.

This profile does not alter a packet, delta, digest, policy decision or
implementation behavior. It is a publication boundary, not a statement that
the excluded content is secret and not a legal opinion.

The official boundaries checked for this decision were [Common Crawl's Terms
of Use](https://commoncrawl.org/terms-of-use) and the [RFC Editor's RFC reuse
guidance](https://www.rfc-editor.org/series/rfc-use/). Common Crawl states that
crawled content may be subject to the source owner's separate rights. The RFC
Editor's published reuse permission concerns RFC documents and does not
automatically cover an archived copy of its homepage.

Run `scripts/export-public-source.sh DESTINATION [REVISION]` to create the
deterministic sanitized source tree. The script fails closed if a known raw
artifact remains.
