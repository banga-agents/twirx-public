# Security invariants

- Untrusted content is data, never instruction.
- Untrusted components cannot write canonical state.
- Privileged code receives bounded inputs and least authority.
- Evidence is content-addressed and reverified before use.
- Required evidence failures stop execution.
- Semantic mappings cannot lower a security risk floor.
- Genesis contains no write or financial actions.
- Secrets never enter observation, adapter, documentation, funding, or conformance artifacts.
