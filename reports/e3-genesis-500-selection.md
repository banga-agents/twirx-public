# E3 Genesis-500 selection and quota report

**Selection version:** 2026-08-10

**Selection artifact:** `atlas/genesis-500/selection.json`

**SHA-256:** `209c3629ad58c60ea4477436332ec4847b3f0ecaf7b091046923c95c87f53fc5`

## Result

The selection contains exactly 500 unique HTTPS origin candidates and exactly
matches the nine domain-family quotas.

| Domain family | A0 candidates |
|---|---:|
| Government, law, and public data | 100 |
| Science, research, and scholarly infrastructure | 80 |
| Standards, technical documentation, and open source | 60 |
| Economics, markets, and public disclosure | 60 |
| Journalism and public-interest publishing | 50 |
| Health and public health | 50 |
| Climate, environment, and Earth systems | 50 |
| Education, reference, and culture | 30 |
| Humanitarian, civic, and global development | 20 |
| **Total** | **500** |

All 500 records are `A0`, `unreviewed`, and `network_observed: false`.
Publisher, jurisdiction, and language hints are deliberately unknown. The
file therefore establishes selection breadth only. It establishes zero A1,
A2, A3, A4, A5, A6, A7, A8, or A9 origins and proves none of the overlapping
surface or diversity targets.

## Commands executed

```bash
jq -e '.format == "tw.atlas-selection/0.1" and
  (.candidates|length)==500' atlas/genesis-500/selection.json

jq -r '.candidates | group_by(.domain_family)[] |
  [.[0].domain_family,length] | @tsv' \
  atlas/genesis-500/selection.json

jq -r '.candidates[].canonical_origin' \
  atlas/genesis-500/selection.json | sort | uniq -d

jq -r '.candidates[].id' \
  atlas/genesis-500/selection.json | sort | uniq -d

jq '[.candidates[] | select(.maturity == "A0")] | length' \
  atlas/genesis-500/selection.json

jq '[.candidates[] | select(.network_observed == false)] | length' \
  atlas/genesis-500/selection.json

sha256sum atlas/genesis-500/selection.json
```

The duplicate commands produced no output. The two state counts returned 500.

## Exclusions and unresolved review work

- No candidate was fetched, profiled, or treated as a network destination.
- Host reachability, redirects, current ownership, registrable domain,
  publisher identity, jurisdiction, languages, authority, robots, terms,
  attribution, rate limits, retention, authentication, and risk remain
  unreviewed.
- The list includes subdomain origins. Publisher concentration cannot be
  evaluated until publisher identities are reviewed rather than guessed from
  hostnames.
- Candidate placement is a portfolio selection decision, not a semantic or
  factual claim about provider content.
- Diversity and representation-surface targets remain at zero evidence-backed
  coverage until A1/A3 records exist.
- TWIRX itself is one selected candidate so its publisher-authored interface
  can traverse the same maturity and evidence rules as every other origin.

## Recommendation

**PASS for the exact E3.0 candidate-selection prerequisite.**

**FAIL for A1+ public Atlas claims, policy-based access, or network use.**
