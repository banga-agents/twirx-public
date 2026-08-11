# Semantic kernel — design baseline

**Status:** pre-RFC design.

## Kernel terms

```text
Origin
Representation
Observation
SourceStatement
Extraction
Derivation
Evidence
Term
Type
Relation
Role
Context
Time
IdentityReference
Mapping
Resource
Operation
Effect
Capability
Policy
Authority
Mandate
Receipt
Version
Module
```

## Module algebra

```text
import
extend
specialize
compose
profile
constrain
map
translate
deprecate
supersede
fork
```

## Compatibility classes

```text
ADDITIVE
RESTRICTIVE
BROADENING
MEANING_CHANGING
IDENTITY_CHANGING
EFFECT_CHANGING
SECURITY_CHANGING
DEPRECATION
```

## Non-negotiable semantic laws

- Native statements survive every mapping.
- Claims remain attributed to origins.
- Context and time are first-class.
- Identity is scoped and revisable.
- Absence has explicit states.
- Contradictions remain representable.
- Mappings carry provenance and status.
- Published versions are immutable.
- Operational inference is bounded.
