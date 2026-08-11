# Task 009: FUTO real archive semantic proof

## Status

Implementation candidate pending the complete release validation report.

## Objective

Compile the two exact, admitted RFC Editor historical captures into immutable
source-native Semantic Packets, classify their demonstrated source change as
one origin delta, and expose packets, proof, query results, and the delta from
the read-only Semantic Snapshot runtime.

## Authorized input

Only:

```text
atlas/archive-acquisitions/rfc-editor-futo-history
```

The acquisition must reconcile with the current sealed work order and the
current completed policy-decision evidence. This task performs no network
request and does not authorize another acquisition.

## Invariants

- preserve the exact first `html/head/title[1]/text()` UTF-8 lexical bytes;
- do not decode entities, normalize whitespace, infer a semantic concept, or
  promote a mapping;
- classify every packet as `archive_observation`, historical,
  `observed_native`, stale, and not a current publisher statement;
- copy and bind the complete independently verifiable acquisition and capture
  evidence into the snapshot proof surface;
- emit an origin delta only when the before and after source-evidence digests
  and source-native lexical values both differ;
- keep the snapshot compiler and runtime network-free, immutable and
  read-only;
- exclude controlled fixtures from public counts and default queries;
- expose bounded packet, trace, proof and delta reads only;
- no canon promotion, model, browser, live refresh, payment, authentication,
  write action, PostgreSQL deployment, scheduler, or arbitrary URL.

## Acceptance

- two archive packets independently re-extract from retained representations;
- one origin `modified` delta binds both packet and source-evidence identities;
- the historical title query returns both exact native lexical values;
- the delta page and canonical delta download are bounded and immutable;
- any proof, packet, delta, manifest, profile, plan, or representation
  tampering fails admission;
- Go tests, fuzzing, vet, restricted-C GCC/Clang/ASan/UBSan verification,
  snapshot end-to-end tests, and documentation checks pass.

The delta `batch_id` topology remains a separately disclosed contract issue:
the FUTO snapshot profile binds it to the already-published acquisition
manifest so no self-digest cycle is introduced. This task does not silently
amend the normative packet-batch contract.
