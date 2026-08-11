# The Typed Web Manifesto

## Preamble

The Web was created so knowledge and capability could cross boundaries.

Its primary interface was built for human perception: pages, links, forms, visual hierarchy, gestures, and browsers. That interface changed civilization. It made information reachable, publishing universal, and cooperation possible across distance.

Now another class of participant is arriving. Software agents can search, compare, reason, coordinate, and act. Yet they enter the Web through an interface that was not designed for them. They imitate human browsing, infer meaning from presentation, repeat expensive navigation, and often lose the evidence that would allow another person or machine to verify what they found.

The answer cannot be a permanent dependence on proprietary browsers, private model providers, opaque extraction systems, or a handful of companies deciding what the machine-readable Web means.

We therefore establish the Typed Web Commons: an open protocol and public technical canon through which web representations may become typed, inspectable, provenance-bearing, and eventually safely actionable.

We are not building a replacement for the Web.

We are building the missing semantic and evidentiary layer through which the Web can remain open in an agent-driven age.

---

## I. The purpose

Typed Web exists to make public web resources directly usable by agents while preserving accountability to people, publishers, and evidence.

A successful interface must answer more than “what value did the system return?” It must also answer:

- Which origin supplied the representation?
- What bytes were observed?
- When were they observed?
- Where in the representation did the source statement appear?
- Which extractor read it?
- Which transformations were applied?
- Which semantic mapping interpreted it?
- Which schema and ontology versions were in force?
- What remains uncertain or unresolved?
- What authority, if any, approved the interface?

The agent should not have to trust an invisible chain of inference. The chain must be part of the result.

---

## II. The truth we can honestly guarantee

Typed Web is not a ministry of truth and must never become one.

A provider may be accurate, mistaken, deceptive, outdated, satirical, compromised, or in disagreement with another provider. Typing a statement does not transform it into objective fact. A high-confidence model prediction does not create authority. A blockchain commitment does not make the committed proposition true.

Our truth contract is deliberately narrower and stronger:

> We can prove what an origin represented, what our system observed, and how a declared derivation produced the value shown to the agent.

The protocol therefore distinguishes:

```text
origin
representation
observation
source statement
extraction assertion
transformation
semantic interpretation
agent-facing result
```

These must never be silently collapsed.

When two origins disagree, the system preserves both sourced statements. When a mapping is uncertain, the system says so. When evidence is incomplete, the result is incomplete. When an adapter breaks, degradation is visible. When no reliable interpretation exists, the correct output is unresolved—not a plausible invention.

Truthfulness in this project means fidelity to source and derivation.

---

## III. Native meaning before normalization

The world does not speak through one vocabulary.

Websites, communities, scientific fields, cultures, jurisdictions, markets, and institutions use different terms for reasons that matter. A universal interface must not flatten those differences merely to make machine access convenient.

Typed Web therefore preserves the source-native term and lexical value before adding a shared semantic view.

```text
native: current_price = "19,99 €"
semantic: OfferPrice.amount = "19.99"
semantic: OfferPrice.currency = "EUR"
```

The normalized view is useful. The native statement is permanent.

Mappings are not invisible aliases. They are versioned claims with context, evidence, authorship, status, and review history. Identity is scoped and revisable. Global equivalence is rare. Contradiction is information, not database corruption.

This discipline allows the semantic layer to grow without becoming a machine for erasing nuance.

---

## IV. A small kernel and an unbounded extension space

We do not begin by attempting to classify all reality.

The stable kernel defines how to express:

- origins and representations;
- observations and evidence;
- terms, types, relations, roles, and contexts;
- time, version, identity, and uncertainty;
- resources, operations, effects, and capabilities;
- authority, policy, mandates, and receipts.

Domain communities may then build modules for publishing, science, commerce, government, finance, health, education, travel, and domains not yet imagined.

Every origin may retain a source-native schema. Separate mapping modules relate native concepts to shared concepts. Multiple registries may coexist. Agents may choose their trust policies. The official commons curates a canon, but it does not monopolize expression.

The kernel remains conservative. The extension space remains open.

---

## V. Open protocol, plural implementations

No implementation language is constitutional.

No compiler, package registry, runtime, cloud provider, browser, model provider, blockchain, corporation, foundation, or nation owns the Typed Web.

The protocol is defined by:

- normative specifications;
- canonical encodings;
- public test vectors;
- conformance suites;
- security invariants;
- reproducible evidence.

The first implementation may be written in Go. The first independent verifier may be written in C. Other conforming implementations may use Rust, SPARK, Java, Zig, a formally verified system, or a language that does not yet exist.

Implementations may compete. Conformance allows them to cooperate.

Self-hosting must remain possible. Federation must remain possible. Forking must remain possible. No mandatory hosted service may become the price of participation.

---

## VI. Security is not an ornamental layer

Typed Web will process adversarial inputs by design.

We assume that an attacker may control the URL, DNS response, redirect, headers, compression, document structure, JavaScript, metadata, ontology proposal, adapter package, prompt-injection text, contributor account, or one ordinary service credential.

We therefore design for compromise containment rather than fantasies of invulnerability.

- Hostile content is data, never instruction.
- Untrusted workers hold no canonical authority.
- Fetchers, browsers, adapters, and models submit candidates; they do not promote them.
- Unsafe code is bounded behind narrow interfaces.
- Resource use is limited.
- Evidence is content-addressed.
- Releases are immutable and signed.
- Security-sensitive semantic changes receive special review.
- Canonical state can be reconstructed.
- Revocation and degradation are visible.

A page cannot declare itself verified. A model cannot promote its own mapping. An operation cannot reduce its own risk classification. A signed package can still be wrong; signatures identify responsibility and integrity, not truth.

---

## VII. Browsers are a fallback, not the permanent interface

Many websites are already backed by structured sources: official APIs, JSON endpoints, GraphQL, feeds, server-rendered state, JSON-LD, semantic HTML, and emerging agent interfaces.

Typed Web discovers and prefers the least brittle, most authoritative available representation.

```text
publisher-declared interface
official public API
agent protocol endpoint
public structured endpoint
structured metadata
semantic HTML
DOM extraction
isolated browser interaction
```

Browser automation remains important for discovery and compatibility. It should not remain the default tax paid on every agent request.

The long-term success of the project is not that we become perfect at reverse-engineering websites. It is that publishers begin to expose authoritative typed interfaces voluntarily, and reverse engineering becomes progressively less necessary.

---

## VIII. Human and publisher sovereignty

Agent access must not become an excuse for unbounded extraction or unauthorized action.

Typed Web respects authentication, access control, reasonable rate limits, publisher verification, correction, and revocation. It does not normalize bypassing paywalls, evading anti-abuse controls, concealing automated access, or imposing unreasonable infrastructure costs on others.

Read-only access is the default.

Actions require explicit capabilities. Financial, destructive, irreversible, or legally consequential operations require stronger mandates, exact scope, limits, state binding, receipts, and human control. No agent receives general authority when a narrow authorization will suffice.

The protocol should help publishers declare what agents may read and do—not merely help agents work around them.

---

## IX. Cryptography without speculation

Cryptography can make the system more accountable.

It may secure artifact hashes, signatures, transparency logs, publisher attestations, revocations, scoped mandates, execution receipts, and settlement records. Public chains may optionally notarize commitments or coordinate payments.

But the chain is not the Web, not the ontology, not the source of truth, and not the government of the commons.

There will be no protocol token, no sale of governance rights, no tokenized ownership of adapters, and no artificial scarcity imposed on a public semantic layer.

Cryptography is a tool for integrity and authorization. Speculation is not the mission.

---

## X. Models may learn from the architecture; they may not rule it

The architecture will produce an extraordinary semantic training record:

- immutable observations;
- structural candidates;
- proposed types and mappings;
- accepted and rejected interpretations;
- drift events;
- runtime successes and failures;
- adversarial examples;
- human adjudications and rationales.

This evidence may eventually train specialized models capable of understanding web structures, operations, effects, security risks, and semantic correspondences.

But the learning system remains downstream from the evidence system.

Models propose. Deterministic validators test. Conformance measures. Humans and governed policy admit. Model releases are versioned, evaluated, red-teamed, and replaceable.

No live website should be able to retrain the system directly. No prompt should become a security boundary. Abstention is a valid and necessary output.

---

## XI. Public infrastructure requires honest sustenance

Open source does not mean invisible labor.

The commons must not be financed by founder exhaustion, unpaid maintainers, or concealed dependence on private benefactors. Human survival, focused engineering time, security review, documentation, infrastructure, and administration are legitimate project costs.

Typed Web will therefore disclose:

- received contributions;
- declared project wallets;
- maintainer compensation and runway;
- infrastructure and service expenses;
- grants and sponsorship relationships;
- material in-kind support;
- redacted receipts and invoices where publication is safe;
- treasury policies and decision records.

Contributions do not purchase protocol ownership, technical truth, private vetoes, or governance votes.

A contributor may remain private to the public where the payment channel permits it, but the project will not misuse the word anonymous. On-chain payments are pseudonymous and visible. Payment processors know their users. Financial transparency must coexist with contributor and maintainer safety.

The standard remains free. Self-hosting remains free. Competition remains permitted. Managed reliability, verified operation, institutional support, and public-interest deployments may fund the work.

---

## XII. The path to adoption

The project will grow through proof rather than declaration.

### Genesis

A local, read-only, evidence-native compiler spine. One origin. One immutable observation. One deterministic adapter. One independent verifier. Complete field-level provenance.

### Commons

Multiple source classes, conformance, trust states, adapter health, drift detection, public registry, and self-hosted runtime.

### Publisher participation

Domain verification, publisher-approved schemas, corrections, signed interfaces, declared policies, and a standard discovery location.

### Federation

Independent registries, implementations, research groups, publishers, and public institutions exchanging immutable modules and conformance evidence.

### Safe actions

Explicit effect models, least-authority mandates, human controls, receipts, disputes, and optional payment settlement.

### Standardization

Multiple mature implementations, real interoperability, publisher adoption, and neutral standards stewardship.

### Agent-native Web

Websites publish trustworthy typed capabilities by default. Agents use browsers when human presentation or exceptional compatibility requires them—not because no machine interface exists.

---

## XIII. The covenant

We establish these commitments:

1. Every canonical value carries evidence or remains explicitly unresolved.
2. Native meaning is preserved before normalization.
3. Source statements are never silently promoted into objective facts.
4. Authority, confidence, freshness, and integrity remain separate.
5. Failure, drift, uncertainty, and contradiction remain visible.
6. The protocol remains independently implementable and self-hostable.
7. No language, vendor, model, chain, or institution becomes sovereign over the commons.
8. Anyone may extend the semantic world; canonization remains governed and auditable.
9. Untrusted systems cannot grant themselves canonical, semantic, or financial authority.
10. Security is designed into every boundary from the first implementation.
11. Human and publisher sovereignty take precedence over agent convenience.
12. Funding is transparent, compensation is honest, and contributions purchase no ownership of the standard.
13. The project will not create a speculative token.
14. The work will be judged by reliability, reproducibility, restraint, and usefulness to the world.

---

## Closing

A machine-readable Web will exist.

The decisive question is whether it will be open or enclosed, evidentiary or opaque, plural or centralized, accountable or merely persuasive.

We choose an open interface commons.

We choose evidence over unsupported certainty.

We choose explicit authority over ambient power.

We choose semantic plurality without incoherence.

We choose security as care under hostile conditions.

We choose to build infrastructure that any person may inspect, implement, challenge, improve, and use.

The Typed Web belongs to everyone—or it is not the Typed Web.
