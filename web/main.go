// Command web builds the TWIRX public website into web/dist.
//
// It uses the Go standard library only. There is no package manager, no
// lockfile, no node_modules, and no build-time network access. A reviewer can
// audit the entire website toolchain by reading this one file.
//
//	go run .            build, then verify the output
//	go run . -serve     build, verify, and serve dist on http://localhost:8080
//	go run . -out DIR   build into a different directory
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Site is the whole build-time configuration, loaded from site.json.
type Site struct {
	Name       string  `json:"name"`
	Mission    string  `json:"mission"`
	BaseURL    string  `json:"base_url"`
	RepoURL    string  `json:"repo_url"`
	DocsURL    string  `json:"docs_url"`
	StatusNote string  `json:"status_note"`
	FooterLead string  `json:"footer_lead"`
	Pages      []*Page `json:"pages"`

	// CanonicalHost is the only host permitted in canonical and Open Graph
	// URLs. A public site that canonicalises to someone else's origin is a
	// build failure, not a copy-editing problem.
	CanonicalHost string `json:"canonical_host"`

	// ScriptsAllowedOn lists the page slugs permitted to load a first-party
	// script. Every other page must work, and ship, with no JavaScript at all.
	ScriptsAllowedOn []string `json:"scripts_allowed_on"`

	// ScriptBudgetGzipBytes caps the total compressed first-party JavaScript.
	ScriptBudgetGzipBytes int `json:"script_budget_gzip_bytes"`

	// Data holds every file in data/ keyed by its base name without the
	// extension, so pages can render facts instead of restating them.
	Data map[string]any `json:"-"`
}

// Page is one route. Content lives in pages/<File>.
type Page struct {
	Slug        string `json:"slug"`
	File        string `json:"file"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Nav         string `json:"nav"`
	Kicker      string `json:"kicker"`
	Heading     string `json:"heading"`
	Standfirst  string `json:"standfirst"`
}

// URL is the absolute site path for the page, always directory-style.
func (p *Page) URL() string {
	if p.Slug == "" {
		return "/"
	}
	return "/" + p.Slug + "/"
}

// OutPath is the file the page is written to, relative to the output root.
func (p *Page) OutPath() string {
	if p.Slug == "" {
		return "index.html"
	}
	return filepath.Join(p.Slug, "index.html")
}

// view is the data passed to every template execution.
type view struct {
	Site *Site
	Page *Page
	Data map[string]any
	Nav  []*Page
}

func main() {
	out := flag.String("out", "dist", "output directory")
	serve := flag.String("serve", "", "after building, serve the output on this address (for example :8080)")
	evidence := flag.Bool("evidence", false, "regenerate data/evidence-e1.json from repository artefacts, then exit")
	repo := flag.String("repo", "..", "path to the repository root, for -evidence")
	flag.Parse()

	if *evidence {
		if err := generateEvidence(*repo, filepath.Join("data", "evidence-e1.json")); err != nil {
			log.Fatalf("web: evidence: %v", err)
		}
		return
	}
	if err := run(*out, *serve); err != nil {
		log.Fatalf("web: %v", err)
	}
}

func run(out, serve string) error {
	site, err := loadSite()
	if err != nil {
		return err
	}
	if err := build(site, out); err != nil {
		return err
	}
	problems, err := check(site, out)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(os.Stderr, "  FAIL  %s\n", p)
		}
		return fmt.Errorf("%d check failure(s)", len(problems))
	}
	fmt.Printf("built %d pages into %s/ — all checks passed\n", len(site.Pages), out)

	if serve != "" {
		fmt.Printf("serving %s on http://localhost%s\n", out, serve)
		return http.ListenAndServe(serve, http.FileServer(http.Dir(out)))
	}
	return nil
}

// ---------------------------------------------------------------- loading ---

func loadSite() (*Site, error) {
	var site Site
	if err := readJSON("site.json", &site); err != nil {
		return nil, err
	}

	site.Data = map[string]any{}
	entries, err := os.ReadDir("data")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		var v any
		if err := readJSON(filepath.Join("data", e.Name()), &v); err != nil {
			return nil, err
		}
		// Templates address data by field name, so the key is the file's base
		// name with hyphens folded to underscores: project-status.json becomes
		// {{.Data.project_status}}.
		key := strings.ReplaceAll(strings.TrimSuffix(e.Name(), ".json"), "-", "_")
		site.Data[key] = v
	}
	return &site, nil
}

// readJSON decodes with UseNumber so that numeric facts render exactly as they
// were written. Evidence figures must not be reformatted by a float printer.
func readJSON(name string, v any) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.UseNumber()
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// ---------------------------------------------------------------- building ---

func build(site *Site, out string) error {
	if err := os.RemoveAll(out); err != nil {
		return err
	}

	nav := make([]*Page, 0, len(site.Pages))
	for _, p := range site.Pages {
		if p.Nav != "" {
			nav = append(nav, p)
		}
	}

	layout, err := template.New("layout").Funcs(funcs()).ParseGlob("templates/*.html")
	if err != nil {
		return err
	}

	for _, p := range site.Pages {
		tmpl, err := layout.Clone()
		if err != nil {
			return err
		}
		if _, err := tmpl.ParseFiles(filepath.Join("pages", p.File)); err != nil {
			return err
		}
		dest := filepath.Join(out, p.OutPath())
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		err = tmpl.ExecuteTemplate(f, "base", view{Site: site, Page: p, Data: site.Data, Nav: nav})
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fmt.Errorf("%s: %w", p.File, err)
		}
	}

	if err := copyTree("static", out); err != nil {
		return err
	}
	// Host configuration that must sit in the publish directory itself.
	// Netlify and Cloudflare Pages read _headers and _redirects from the root
	// of the deployed output; on any other host these are inert text files.
	for _, name := range []string{"_headers", "_redirects"} {
		b, err := os.ReadFile(filepath.Join("deploy", name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, name), b, 0o644); err != nil {
			return err
		}
	}
	// The machine-readable facts are published, not merely consumed, so that a
	// reviewer can diff what the pages claim against a stable JSON surface.
	if err := copyTree("data", filepath.Join(out, "data")); err != nil {
		return err
	}
	if err := writeSitemap(site, out); err != nil {
		return err
	}
	return writeRobots(site, out)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func writeSitemap(site *Site, out string) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range site.Pages {
		fmt.Fprintf(&b, "  <url><loc>%s%s</loc></url>\n", strings.TrimSuffix(site.BaseURL, "/"), p.URL())
	}
	b.WriteString("</urlset>\n")
	return os.WriteFile(filepath.Join(out, "sitemap.xml"), []byte(b.String()), 0o644)
}

func writeRobots(site *Site, out string) error {
	body := fmt.Sprintf(`User-agent: *
Allow: /

# This site is meant to be read by software agents as well as by people.
# The project's facts are published as machine-readable files beside the pages:
#   /data/project-status.json
#   /data/funding-status.json
#   /data/implementation-matrix.json
#   /data/gates.json

Sitemap: %s/sitemap.xml
`, strings.TrimSuffix(site.BaseURL, "/"))
	return os.WriteFile(filepath.Join(out, "robots.txt"), []byte(body), 0o644)
}

// ------------------------------------------------------------- template api ---

// statusLabel maps an implementation-status key to its label and its
// non-colour marker. Status is never communicated by colour alone.
var statusLabel = map[string][2]string{
	"implemented": {"Implemented", "●"},
	"specified":   {"Specified", "◐"},
	"planned":     {"Planned", "○"},
	"research":    {"Research", "◌"},
	"normative":   {"Normative", "§"},
	"explanatory": {"Explanatory", "¶"},
	"complete":    {"Complete", "●"},
	"in-progress": {"In progress", "◐"},
	"next":        {"Next", "▸"},
	"not-proven":  {"Not proven", "×"},
	"unadmitted":  {"Implemented — not admitted", "◒"},
}

func funcs() template.FuncMap {
	return template.FuncMap{
		// chip renders an implementation-status or document-authority label.
		"chip": func(kind string) (template.HTML, error) {
			l, ok := statusLabel[kind]
			if !ok {
				return "", fmt.Errorf("unknown status %q", kind)
			}
			return template.HTML(fmt.Sprintf(
				`<span class="chip chip--%s"><span class="chip__mark" aria-hidden="true">%s</span>%s</span>`,
				kind, l[1], l[0])), nil
		},
		// commas groups an integer for display without altering its value.
		"commas": func(v any) string { return commas(fmt.Sprint(v)) },
		// pct renders part as a share of total, for budget tables.
		"pct": func(part, total any) (string, error) {
			p, err := strconv.ParseFloat(fmt.Sprint(part), 64)
			if err != nil {
				return "", err
			}
			t, err := strconv.ParseFloat(fmt.Sprint(total), 64)
			if err != nil || t == 0 {
				return "", fmt.Errorf("pct: bad total %v", total)
			}
			return strconv.FormatFloat(p/t*100, 'f', 1, 64) + "%", nil
		},
		// repo links a path inside the source repository.
		"repo": func(site *Site, p string) string {
			return strings.TrimSuffix(site.RepoURL, "/") + "/blob/main/" + p
		},
		"upper": strings.ToUpper,
		// pick finds the first item in a JSON array whose key equals want.
		"pick": func(items any, key, want string) (any, error) {
			list, ok := items.([]any)
			if !ok {
				return nil, fmt.Errorf("pick: not a list")
			}
			for _, it := range list {
				m, _ := it.(map[string]any)
				if fmt.Sprint(m[key]) == want {
					return m, nil
				}
			}
			return nil, fmt.Errorf("pick: no item with %s=%q", key, want)
		},
		// short truncates a digest for display. The full value is always
		// present elsewhere on the page; this is for scanning, not for citing.
		"short": func(n int, v any) string {
			s := fmt.Sprint(v)
			prefix := ""
			if i := strings.Index(s, ":"); i >= 0 && i < 12 {
				prefix, s = s[:i+1], s[i+1:]
			}
			if len(s) <= n {
				return prefix + s
			}
			return prefix + s[:n] + "…"
		},
	}
}

func commas(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, hasFrac := strings.Cut(s, ".")
	var out []byte
	for i, c := range []byte(whole) {
		if i > 0 && (len(whole)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	res := string(out)
	if hasFrac {
		res += "." + frac
	}
	if neg {
		res = "-" + res
	}
	return res
}

// ---------------------------------------------------------------- checking ---

var (
	reHref       = regexp.MustCompile(`href="([^"]*)"`)
	reSubresrc   = regexp.MustCompile(`(?:src|srcset)="([^"]*)"`)
	reStylesheet = regexp.MustCompile(`<link[^>]*rel="stylesheet"[^>]*href="([^"]*)"`)
	reCSSURL     = regexp.MustCompile(`url\(\s*['"]?([^'")]+)`)
	reScript     = regexp.MustCompile(`(?is)<script\b[^>]*>`)
	reRemoteJS   = regexp.MustCompile(`https?://|\bimport\s*\(`)
	reInlineOn   = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*["']`)
	reTitle      = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	reDesc       = regexp.MustCompile(`<meta name="description" content="([^"]*)"`)
	reH1         = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)
	reExternal   = regexp.MustCompile(`^(https?:)?//`)
	reID         = regexp.MustCompile(`\sid="([^"]+)"`)
	reAria       = regexp.MustCompile(`aria-labelledby="([^"]+)"`)
	reHeading    = regexp.MustCompile(`<h([1-6])[\s>]`)
)

// check enforces the acceptance rules that are cheap to verify mechanically:
// every internal link resolves, no client JavaScript ships, no subresource is
// fetched from a third party, and every page carries unique metadata.
// verifyEvidence re-derives the part of the E1 proof that can be checked from
// the published artefact alone: that the raw origin bytes still hash to the
// digest the observation recorded, and that every field's provenance points at
// that same observation, body, and adapter. The site therefore re-runs one link
// of the protocol's evidence chain on every build, and refuses to publish a
// proof it cannot reproduce.
func verifyEvidence(data map[string]any) []string {
	var problems []string
	fail := func(f string, a ...any) { problems = append(problems, "evidence-e1.json: "+fmt.Sprintf(f, a...)) }

	root, ok := data["evidence_e1"].(map[string]any)
	if !ok {
		return []string{"evidence-e1.json: generated proof data is missing; run `go run . -evidence`"}
	}
	origin, _ := root["origin"].(map[string]any)
	observation, _ := root["observation"].(map[string]any)
	adapter, _ := root["adapter"].(map[string]any)
	fields, _ := root["fields"].([]any)

	bodyText, _ := origin["body_text"].(string)
	bodyDigest, _ := origin["body_digest"].(string)
	if bodyText == "" || bodyDigest == "" {
		fail("origin body or digest is absent")
		return problems
	}
	sum := sha256.Sum256([]byte(bodyText))
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != bodyDigest {
		fail("published origin bytes hash to %s but the observation records %s", got, bodyDigest)
	}
	if size := fmt.Sprint(origin["body_size"]); size != fmt.Sprint(len(bodyText)) {
		fail("published origin bytes are %d long but the observation records %s", len(bodyText), size)
	}

	envelope, _ := observation["envelope_hash"].(string)
	adapterDigest, _ := adapter["digest"].(string)
	if envelope == "" || adapterDigest == "" {
		fail("observation envelope hash or adapter digest is absent")
	}
	// The adapter digest must equal the recorded hash of the adapter file that
	// was actually read, not merely whatever the result envelope asserted.
	adapterPath, _ := adapter["path"].(string)
	var recorded string
	for _, s := range toSlice(root["sources"]) {
		sm, _ := s.(map[string]any)
		if p, _ := sm["path"].(string); p == adapterPath {
			h, _ := sm["sha256"].(string)
			recorded = "sha256:" + h
		}
	}
	if recorded == "" {
		fail("adapter file %q is not among the recorded sources", adapterPath)
	} else if recorded != adapterDigest {
		fail("adapter digest %s does not match the hash of %s (%s)", adapterDigest, adapterPath, recorded)
	}
	if len(fields) == 0 {
		fail("no result fields")
	}
	for i, f := range fields {
		fm, _ := f.(map[string]any)
		id, _ := fm["id"].(string)
		prov, _ := fm["provenance"].(map[string]any)
		if prov == nil {
			fail("field %d (%s) carries no provenance", i, id)
			continue
		}
		if v, _ := prov["observation_hash"].(string); v != envelope {
			fail("field %q cites observation %s, not %s", id, v, envelope)
		}
		if v, _ := prov["body_digest"].(string); v != bodyDigest {
			fail("field %q cites body %s, not %s", id, v, bodyDigest)
		}
		if v, _ := prov["adapter_digest"].(string); v != adapterDigest {
			fail("field %q cites adapter %s, not %s", id, v, adapterDigest)
		}
	}
	return problems
}

// verifyMetrics enforces that no proof metric is published without naming the
// report it comes from and the scope it was measured at.
func verifyMetrics(data map[string]any) []string {
	var problems []string
	status, _ := data["project_status"].(map[string]any)
	metrics, _ := status["proof_metrics"].([]any)
	if len(metrics) == 0 {
		return []string{"project-status.json: no proof_metrics defined"}
	}
	for i, m := range metrics {
		mm, _ := m.(map[string]any)
		label, _ := mm["label"].(string)
		for _, key := range []string{"label", "value", "source", "scope"} {
			if s, _ := mm[key].(string); s == "" {
				if key == "value" && mm[key] != nil {
					continue
				}
				problems = append(problems, fmt.Sprintf(
					"project-status.json: proof metric %d (%q) has no %s; every metric must name its report and measured scope",
					i, label, key))
			}
		}
	}
	return problems
}

func check(site *Site, out string) ([]string, error) {
	var problems []string
	problems = append(problems, verifyEvidence(site.Data)...)
	problems = append(problems, verifyMetrics(site.Data)...)

	if u, err := url.Parse(site.BaseURL); err != nil || u.Host != site.CanonicalHost {
		problems = append(problems, fmt.Sprintf(
			"site.json: base_url host is %q but canonical_host is %q", hostOf(site.BaseURL), site.CanonicalHost))
	}

	// Every recorded unresolved risk must appear in the published risk set.
	// Omitting an inconvenient one is the specific failure this guards against.
	proofHTML, err := os.ReadFile(filepath.Join(out, "proof", "index.html"))
	if err != nil {
		return nil, err
	}
	status, _ := site.Data["project_status"].(map[string]any)
	risks, _ := status["unresolved_risks"].([]any)
	if len(risks) == 0 {
		problems = append(problems, "project-status.json: no unresolved risks recorded")
	}
	for i, r := range risks {
		text, _ := r.(string)
		if !strings.Contains(string(proofHTML), htmlEscape(firstWords(text, 8))) {
			problems = append(problems, fmt.Sprintf("/proof/: recorded risk R%d is not published", i+1))
		}
	}

	built := map[string]bool{}
	err = filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(out, p)
		if err != nil {
			return err
		}
		built["/"+filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	titles := map[string]string{}
	descs := map[string]string{}

	for _, page := range site.Pages {
		body, err := os.ReadFile(filepath.Join(out, page.OutPath()))
		if err != nil {
			return nil, err
		}
		html := string(body)
		where := page.URL()

		if !strings.Contains(html, `<html lang="`) {
			problems = append(problems, where+": <html> has no lang attribute")
		}
		if strings.Contains(html, `target="_blank"`) {
			problems = append(problems, where+`: uses target="_blank"; outbound links open in the same tab by design`)
		}
		// JavaScript policy: none at all, except a first-party module on the
		// pages explicitly listed in site.json. Inline script and inline event
		// handlers are forbidden everywhere so the CSP can stay script-src
		// 'self' with no unsafe-inline.
		scripts := reScript.FindAllStringSubmatch(html, -1)
		allowed := false
		for _, s := range site.ScriptsAllowedOn {
			if s == page.Slug {
				allowed = true
			}
		}
		for _, m := range scripts {
			switch {
			case !allowed:
				problems = append(problems, where+": loads a script, but this page is not in scripts_allowed_on")
			case !strings.Contains(m[0], `src="/`):
				problems = append(problems, where+": inline or non-first-party script; only a same-origin src is permitted")
			}
		}
		if strings.Contains(html, "</script>") && !strings.Contains(html, `></script>`) {
			problems = append(problems, where+": script element has inline content")
		}
		if m := reInlineOn.FindString(html); m != "" {
			problems = append(problems, where+": contains an inline event handler ("+strings.TrimSpace(m)+")")
		}

		for _, m := range reSubresrc.FindAllStringSubmatch(html, -1) {
			if reExternal.MatchString(m[1]) {
				problems = append(problems, where+": third-party subresource "+m[1])
			}
		}
		for _, m := range reStylesheet.FindAllStringSubmatch(html, -1) {
			if reExternal.MatchString(m[1]) {
				problems = append(problems, where+": third-party stylesheet "+m[1])
			}
		}

		title := firstGroup(reTitle, html)
		desc := firstGroup(reDesc, html)
		switch {
		case title == "":
			problems = append(problems, where+": missing <title>")
		case titles[title] != "":
			problems = append(problems, where+": duplicate <title> also used by "+titles[title])
		default:
			titles[title] = where
		}
		switch {
		case desc == "":
			problems = append(problems, where+": missing meta description")
		case descs[desc] != "":
			problems = append(problems, where+": duplicate meta description also used by "+descs[desc])
		default:
			descs[desc] = where
		}
		if n := len(reH1.FindAllString(html, -1)); n != 1 {
			problems = append(problems, fmt.Sprintf("%s: expected exactly one <h1>, found %d", where, n))
		}

		// Anchors, ARIA references, and heading order. These are the parts of
		// accessibility that can be checked mechanically; the rest is in the
		// Gate 2 website report.
		ids := map[string]int{}
		for _, m := range reID.FindAllStringSubmatch(html, -1) {
			ids[m[1]]++
			if ids[m[1]] == 2 {
				problems = append(problems, where+": duplicate id "+m[1])
			}
		}
		for _, m := range reAria.FindAllStringSubmatch(html, -1) {
			for _, ref := range strings.Fields(m[1]) {
				if ids[ref] == 0 {
					problems = append(problems, where+": aria-labelledby points at missing id "+ref)
				}
			}
		}
		prev := 0
		for _, m := range reHeading.FindAllStringSubmatch(html, -1) {
			level, _ := strconv.Atoi(m[1])
			switch {
			case prev == 0 && level != 1:
				problems = append(problems, fmt.Sprintf("%s: first heading is h%d, not h1", where, level))
			case prev != 0 && level > prev+1:
				problems = append(problems, fmt.Sprintf("%s: heading level jumps h%d to h%d", where, prev, level))
			}
			prev = level
		}

		for _, m := range reHref.FindAllStringSubmatch(html, -1) {
			ref := m[1]
			if ref == "" || reExternal.MatchString(ref) || strings.HasPrefix(ref, "mailto:") {
				continue
			}
			if strings.HasPrefix(ref, "#") {
				if ids[ref[1:]] == 0 {
					problems = append(problems, where+": fragment link "+ref+" has no target")
				}
				continue
			}
			if !strings.HasPrefix(ref, "/") {
				problems = append(problems, where+": relative link "+ref+"; use absolute site paths")
				continue
			}
			target, _, _ := strings.Cut(ref, "#")
			if !resolves(built, target) {
				problems = append(problems, where+": dead internal link "+ref)
			}
		}
	}

	css, err := os.ReadFile(filepath.Join(out, "styles.css"))
	if err != nil {
		return nil, err
	}
	for _, m := range reCSSURL.FindAllStringSubmatch(string(css), -1) {
		if reExternal.MatchString(m[1]) {
			problems = append(problems, "styles.css: third-party asset "+m[1])
		}
	}

	// First-party JavaScript budget, measured compressed because that is what
	// a visitor actually pays.
	total := 0
	err = filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".js" {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if reRemoteJS.Match(b) {
			problems = append(problems, filepath.Base(p)+": references a remote URL or a dynamic import")
		}
		total += gzipLen(b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if site.ScriptBudgetGzipBytes > 0 && total > site.ScriptBudgetGzipBytes {
		problems = append(problems, fmt.Sprintf(
			"first-party JavaScript is %d bytes gzipped, over the %d byte budget", total, site.ScriptBudgetGzipBytes))
	}

	sort.Strings(problems)
	return problems, nil
}

func gzipLen(b []byte) int {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Len()
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Host
}

// firstWords returns the opening n words of s, used to locate a recorded risk
// inside the rendered page without depending on its exact HTML escaping.
func firstWords(s string, n int) string {
	f := strings.Fields(s)
	if len(f) > n {
		f = f[:n]
	}
	return strings.Join(f, " ")
}

func htmlEscape(s string) string { return template.HTMLEscapeString(s) }

func resolves(built map[string]bool, target string) bool {
	if built[target] {
		return true
	}
	return built[path.Join(target, "index.html")]
}

func firstGroup(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}
