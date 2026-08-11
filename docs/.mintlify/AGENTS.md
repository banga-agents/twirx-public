# Typed Web documentation agent instructions

## Voice

- Technical, exact, direct, and calm.
- Describe implemented behavior separately from planned behavior.
- Never use marketing superlatives without measurements.
- Use “origin representation” or “source statement” instead of asserting that provider content is true.
- Preserve the distinction between evidence integrity, publisher identity, authority, confidence, and freshness.

## Required page structure

Protocol pages should include:

1. Purpose.
2. Normative or non-normative status.
3. Data model or contract.
4. Invariants.
5. Failure behavior.
6. Security considerations.
7. Conformance requirements.
8. Implementation status.

## Code examples

- Use executable repository commands and real project paths.
- Show error behavior where relevant.
- Do not invent APIs that are not implemented; label future examples explicitly as design direction.
- Never include secrets, private keys, seed phrases, personal addresses, or live credentials.

## Architecture

- The protocol is language-neutral.
- Go and C are Genesis implementations, not normative authority.
- Models propose but do not canonize.
- Browser automation is a later isolated fallback.
- Genesis is read-only, local-first, deterministic, and evidence-native.
