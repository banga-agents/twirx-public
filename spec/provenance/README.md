# Field-level provenance

Every result field contains:

```text
request_url
final_url
retrieved_at
body_digest
observation_hash
adapter_id
adapter_version
adapter_digest
extraction_method
locator
transform_chain
mapping_relation
```

Provenance accompanies unresolved optional fields as well as resolved fields. A field without evidence cannot masquerade as a canonical resolved value.
