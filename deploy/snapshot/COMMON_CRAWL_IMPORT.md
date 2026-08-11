# Bounded Common Crawl archive importer

Status: offline importer and restricted acquisition helper implemented; no real
archive acquisition admitted

Common Crawl is a historical observation provider. Its captures are neither a
current publisher statement nor automatic policy permission.

## Input authority

The importer accepts only a reviewed work-order file containing:

- canonical Atlas `origin_id`;
- canonical origin host and permitted representative routes;
- one or two explicitly selected Common Crawl collection identifiers;
- capture count and byte budgets;
- policy-decision evidence digest;
- expected evidence classification.

There is no public URL parameter. Redirects do not broaden the work order.
The separate operator-only acquisition helper permits HTTPS connections only
to the exact official index and data hosts. It accepts no URL, host, range,
collection or redirect parameter. Every request is derived from a validated
sealed work order and the returned exact index record. The human review that
creates the order must select a collection from Common Crawl's official
collection list. Both tools record the exact selected collection ID in the
resulting evidence.

## Hard bounds

```text
periods per origin                         2
captures per period                       3
index requests per origin                 4
index response per request                256 KiB
compressed WARC range per capture          2 MiB
decompressed WARC record per capture       8 MiB
retained representation body per capture  5 MiB
global concurrency                        2
full 500-origin network stop              8 GiB
initial retained archive-evidence stop     2 GiB
```

Range responses are accepted only when status, `Content-Range`, returned byte
count and requested archive object agree. Bytes are written to an immutable
temporary evidence spool, size checked and hashed before WARC parsing. Partial,
oversized, compressed-bomb, wrong-range, wrong-host, malformed WARC and digest
failure cases are rejected without packet compilation.

## Required classification

Every admitted capture records at least:

```text
evidence_class                 archive_observation
freshness                      historical
current_publisher_statement    false
observed_by                    common_crawl
collection_id                  exact selected ID
capture_timestamp              exact archive timestamp
warc_filename                  exact archive object
warc_offset                    exact non-negative offset
warc_length                    exact bounded length
```

Two capture periods may produce an historical origin delta. They cannot prove
when the live publisher changed a representation. A later mapping or canon
change emits a semantic or canon delta, never a fabricated origin delta.

## Required adversarial suite

- non-official index/data host and scheme;
- origin or route absent from the work order;
- redirect and DNS result outside the admitted official host;
- alternate numeric or embedded-credential URL;
- invalid negative, overflowing, overlapping or oversized range;
- `200 OK` full-object response to a bounded range request;
- wrong `Content-Range`, truncated body and trailing body;
- oversized index line, duplicate capture and ambiguous digest;
- malformed WARC headers, chunking, length and compression;
- decompression ratio and retained-body limit violations;
- HTML/WARC content that attempts to supply execution instructions;
- a capture whose final target does not match the reviewed origin identity.

Normal repository tests remain offline. Network acquisition is an explicit,
budgeted operator action and cannot be invoked by the public runtime.

## Restricted acquisition workflow

`twirx-archive-acquire` is the only component in this workflow with an HTTP
client. It is not linked into the snapshot runtime, public Lab or offline WARC
parser. It contacts only `index.commoncrawl.org` and `data.commoncrawl.org`
through the same public-address DNS, TLS and destination checks used by the
secure egress boundary. Redirects are disabled.

```bash
make bin/twirx-archive-acquire

bin/twirx-archive-acquire run \
  --work-orders atlas/archive-work-orders \
  --id ORIGIN-WORK-ORDER \
  --out /new/immutable/acquisition-directory

bin/twirx-archive-acquire verify \
  --root /immutable/acquisition-directory
```

The helper stores the full bounded index response before parsing it. For each
accepted index record it then requests only the exact declared byte range,
stores the compressed range before WARC parsing, invokes the offline importer,
and immediately re-verifies the resulting capture spool. It writes
`acquisition-manifest.json` last. A failed or interrupted run may retain raw
diagnostic evidence, but it has no publication authority without that final
manifest and successful verification.

The command line deliberately has no `--url`, `--host`, `--collection`,
`--range` or redirect option. A completed policy review and syntactically valid
work order are necessary but not sufficient to run a real acquisition: the
founder must explicitly approve the exact origin, route and archive periods.

## Offline operator workflow

`twirx-archive` contains no HTTP client. It validates only locally supplied
files under a sealed work order whose policy review is completed and whose
decision explicitly permits bounded archive profiling.

```bash
make bin/twirx-archive

bin/twirx-archive plan \
  --work-orders atlas/archive-work-orders \
  --id ORIGIN-WORK-ORDER

bin/twirx-archive inspect-index \
  --work-orders atlas/archive-work-orders \
  --id ORIGIN-WORK-ORDER \
  --collection CC-MAIN-YYYY-NN \
  --route https://reviewed.example/exact-route \
  --response /path/to/bounded-index-response.jsonl

bin/twirx-archive import \
  --work-orders atlas/archive-work-orders \
  --id ORIGIN-WORK-ORDER \
  --collection CC-MAIN-YYYY-NN \
  --route https://reviewed.example/exact-route \
  --response /path/to/bounded-index-response.jsonl \
  --capture 0 \
  --warc /path/to/exact-range-response.gz \
  --http-status 206 \
  --content-range 'bytes START-END/TOTAL' \
  --out /new/immutable/capture-directory

bin/twirx-archive verify --spool /immutable/capture-directory
```

`plan` reports `network_requests_made: 0` and prints only exact
`index.commoncrawl.org` queries derived from the sealed routes and collection
IDs. `inspect-index` rejects excess records, duplicate or ambiguous captures,
wrong origins, wrong collections and unsafe WARC paths. `import` requires the
exact `206` status and `Content-Range`, publishes the work order, selected
index record, range-response metadata and compressed WARC member before WARC
parsing, and writes the final manifest last. A parser failure therefore leaves
raw evidence but never a complete admitted capture.

The offline tool remains network-incapable. No live request is authorized
merely because it can print an official-host URL, and no archive result is a
current publisher statement.
