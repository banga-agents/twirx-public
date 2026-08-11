# TWIRX public website

A static site built by a single Go program using only the standard library.

There is no package manager, no lockfile, no `node_modules`, and no build-time
network access. The entire toolchain is [`main.go`](main.go) plus
[`evidence.go`](evidence.go) — a reviewer can read all of it in one sitting.
That is a deliberate choice for a project whose argument is that a trusted
computing base should be small enough to audit.

Exactly one first-party JavaScript module ships, on `/proof/explorer/` only,
under an 8 KB gzipped budget enforced by the build. Every other page contains
none, and the explorer's full content is present without it.

## Build

```bash
cd web
go run .                 # build into web/dist, then verify it
go run . -serve :8080    # build, verify, and serve at http://localhost:8080
go run . -out /tmp/site  # build somewhere else
go run . -evidence       # regenerate data/evidence-e1.json from the repository
```

`-evidence` reads committed artefacts at the repository root and rewrites the
generated proof file. It needs `make demo` to have been run first, because the
raw origin bytes come from the content-addressed store that produces. The
ordinary build never regenerates that file — it re-verifies it.

The build fails, loudly and with a non-zero exit code, if any check does not
pass. There is no separate lint step to forget to run.

## What the build verifies

| Check | Rule |
|---|---|
| Internal links | Every site-absolute `href` resolves to a file that was actually built |
| Relative links | Rejected; pages must use absolute site paths |
| Inline JavaScript | No inline `on*` handler and no inline `<script>` body, anywhere |
| Third-party subresources | No `src`, stylesheet `href`, or CSS `url()` pointing at another host |
| Metadata | Every page has a unique `<title>` and a unique meta description |
| Headings | Exactly one `<h1>` per page |
| Language | `<html lang>` present |
| Link targets | No `target="_blank"` |
| JavaScript | Only on pages in `scripts_allowed_on`, only same-origin `src`, never inline |
| JavaScript budget | Total first-party JS under `script_budget_gzip_bytes` when gzipped |
| JavaScript contents | No remote URL and no dynamic `import()` |
| Canonical host | `base_url` must resolve to `canonical_host` |
| Proof data | `data/evidence-e1.json` must exist and its digest chain must hold |
| Proof metrics | Every metric must name a source report and a measured scope |
| Risk register | Every recorded unresolved risk must appear on `/proof/` |

### The proof chain is re-verified, not trusted

On every build the site recomputes the SHA-256 of the raw origin bytes it
publishes and compares it with the digest recorded in the observation, recomputes
the SHA-256 of the adapter file and compares it with the digest every field's
provenance cites, and checks that all fields agree on the same observation, body,
and adapter. **If the chain does not hold, the site does not build.** Changing a
single character of the published origin bytes fails the build with a digest
mismatch.

Accessibility, contrast, and factual accuracy are not machine-checkable here and
are covered in
[`../reports/public-foundation-website.md`](../reports/public-foundation-website.md).

## Layout

```text
web/
  main.go            the whole generator
  site.json          site configuration and the page list
  evidence.go        generates data/evidence-e1.json from repository artefacts
  data/              machine-readable facts, rendered at build time
                     evidence-e1.json is generated; the rest are authored
  templates/         shared layout and partials
  pages/             one template per route
  static/            stylesheet, icon, and the explorer module, copied verbatim
  deploy/            hosting, DNS runbook, CSP, and host config files
  dist/              build output (not committed)
```

## Facts live in `data/`, not in prose

Every mutable figure — test counts, commit identifiers, risk registers, funding
totals, gate states — lives in a JSON file under `data/` and is rendered into
the pages at build time. The same files are published at `/data/*.json` so a
reader or an agent can diff what the pages claim against a stable surface.

Prose is never the source of a number. To correct a figure, edit the JSON.

Each file records its own provenance in a `_source` key. **The Gate 1 evidence
report is authoritative**: if `data/project-status.json` and
`reports/gate-1-genesis.md` ever disagree, the report wins and the JSON is
wrong.

## Adding or editing a page

1. Add an entry to `pages` in `site.json` (slug, file, title, description, and
   the nav label, kicker, heading, and standfirst).
2. Create `pages/<file>` defining a single `{{define "content"}}` block.
3. Run `go run .`.

Status labels come from the `chip` template function and must be one of
`implemented`, `specified`, `planned`, `research`, `normative`, `explanatory`,
`complete`, `in-progress`, `next`, or `not-proven`. An unknown status fails the build,
which is the point: a technical claim cannot be published without a status.

## Before publishing

`site.json` sets `base_url` to `https://twirx.org` and `canonical_host` to
`twirx.org`; the build fails if those disagree. Documentation links point at
`https://docs.twirx.org`, which must exist before launch or the links will 404.

See [`deploy/README.md`](deploy/README.md) for hosting, headers, and the
Content-Security-Policy recommendation.

## Relationship to the protocol module

This is a separate Go module with no requirements, so the protocol module's
package list, test events, and coverage record are unaffected by it. Nothing in
`web/` is part of the protocol, and the site is not a runtime dependency of
anything.
