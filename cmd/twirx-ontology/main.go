package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/ontologyfabric"
)

const maxGeneratedFile = 8 << 20

type compiledItem struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Version  string `json:"version"`
	Digest   string `json:"digest"`
	CBORPath string `json:"cbor_path"`
	Source   string `json:"source"`
}

type compileIndex struct {
	Format string         `json:"format"`
	Items  []compiledItem `json:"items"`
	Status string         `json:"status"`
}

func main() {
	if len(os.Args) < 2 {
		fatal("usage: twirx-ontology validate|compile|diff [flags]")
	}
	var err error
	switch os.Args[1] {
	case "validate":
		err = validateCommand(os.Args[2:])
	case "compile":
		err = compileCommand(os.Args[2:])
	case "diff":
		err = diffCommand(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fatal(err.Error())
	}
}

func validateCommand(args []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	modules, universes, err := loadTree(*root)
	if err != nil {
		return err
	}
	if err := ontologyfabric.ValidateModuleSet(moduleSources(modules)); err != nil {
		return err
	}
	for _, universe := range universes {
		if err := validateUniverseModules(universe.source, modules); err != nil {
			return fmt.Errorf("%s: %w", universe.path, err)
		}
	}
	fmt.Printf("ontology: %d modules and %d universes valid; no network used\n", len(modules), len(universes))
	return nil
}

func compileCommand(args []string) error {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	out := flags.String("out", "generated/e4/ontology", "output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	modules, universes, err := loadTree(*root)
	if err != nil {
		return err
	}
	if err := ontologyfabric.ValidateModuleSet(moduleSources(modules)); err != nil {
		return err
	}
	outputRoot := *out
	if !filepath.IsAbs(outputRoot) {
		outputRoot = filepath.Join(*root, outputRoot)
	}
	var index compileIndex
	index.Format = "tw.ontology-compile-index/0.1"
	index.Status = "draft_implementation_candidate"
	for _, module := range modules {
		compiled, compileErr := ontologyfabric.CompileModule(module.data)
		if compileErr != nil {
			return fmt.Errorf("%s: %w", module.path, compileErr)
		}
		name := safeName(compiled.Manifest.ModuleID + "@" + compiled.Manifest.SemanticVersion)
		rel := filepath.ToSlash(filepath.Join("modules", name+".cbor"))
		if err := atomicfile.Write(filepath.Join(outputRoot, filepath.FromSlash(rel)), compiled.CBOR, maxGeneratedFile, 0o640); err != nil {
			return err
		}
		index.Items = append(index.Items, compiledItem{Kind: "ontology_module", ID: compiled.Manifest.ModuleID, Version: compiled.Manifest.SemanticVersion, Digest: digestText(compiled.Digest), CBORPath: rel, Source: filepath.ToSlash(module.path)})
	}
	for _, universe := range universes {
		if err := validateUniverseModules(universe.source, modules); err != nil {
			return fmt.Errorf("%s: %w", universe.path, err)
		}
		compiled, compileErr := ontologyfabric.CompileUniverse(universe.data)
		if compileErr != nil {
			return fmt.Errorf("%s: %w", universe.path, compileErr)
		}
		name := safeName(compiled.Universe.UniverseID + "@" + compiled.Universe.SemanticVersion)
		rel := filepath.ToSlash(filepath.Join("universes", name+".cbor"))
		if err := atomicfile.Write(filepath.Join(outputRoot, filepath.FromSlash(rel)), compiled.CBOR, maxGeneratedFile, 0o640); err != nil {
			return err
		}
		index.Items = append(index.Items, compiledItem{Kind: "semantic_universe", ID: compiled.Universe.UniverseID, Version: compiled.Universe.SemanticVersion, Digest: digestText(compiled.Digest), CBORPath: rel, Source: filepath.ToSlash(universe.path)})
	}
	sort.Slice(index.Items, func(i, j int) bool {
		if index.Items[i].Kind != index.Items[j].Kind {
			return index.Items[i].Kind < index.Items[j].Kind
		}
		return index.Items[i].ID < index.Items[j].ID
	})
	indexBytes, err := ontologyfabric.MarshalReport(index)
	if err != nil {
		return err
	}
	if err := atomicfile.Write(filepath.Join(outputRoot, "index.json"), indexBytes, maxGeneratedFile, 0o640); err != nil {
		return err
	}
	fmt.Printf("ontology: compiled %d modules and %d universes to %s\n", len(modules), len(universes), outputRoot)
	return nil
}

func diffCommand(args []string) error {
	flags := flag.NewFlagSet("diff", flag.ContinueOnError)
	beforePath := flags.String("before", "", "prior module source")
	afterPath := flags.String("after", "", "new module source")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *beforePath == "" || *afterPath == "" {
		return fmt.Errorf("diff requires --before and --after")
	}
	beforeBytes, err := os.ReadFile(*beforePath)
	if err != nil {
		return err
	}
	afterBytes, err := os.ReadFile(*afterPath)
	if err != nil {
		return err
	}
	before, err := ontologyfabric.ParseModuleSource(beforeBytes)
	if err != nil {
		return err
	}
	after, err := ontologyfabric.ParseModuleSource(afterBytes)
	if err != nil {
		return err
	}
	report, err := ontologyfabric.Diff(before, after)
	if err != nil {
		return err
	}
	data, err := ontologyfabric.MarshalReport(report)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

type moduleFile struct {
	path   string
	data   []byte
	source ontologyfabric.ModuleSource
}

type universeFile struct {
	path   string
	data   []byte
	source ontologyfabric.UniverseSource
}

func loadTree(root string) ([]moduleFile, []universeFile, error) {
	modulePaths, err := filepath.Glob(filepath.Join(root, "ontology", "modules", "*", "module.json"))
	if err != nil {
		return nil, nil, err
	}
	universePaths, err := filepath.Glob(filepath.Join(root, "ontology", "universes", "*.json"))
	if err != nil {
		return nil, nil, err
	}
	if len(modulePaths) == 0 || len(universePaths) == 0 {
		return nil, nil, fmt.Errorf("ontology tree requires modules and universes")
	}
	sort.Strings(modulePaths)
	sort.Strings(universePaths)
	modules := make([]moduleFile, 0, len(modulePaths))
	for _, path := range modulePaths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, nil, readErr
		}
		source, parseErr := ontologyfabric.ParseModuleSource(data)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, parseErr)
		}
		modules = append(modules, moduleFile{path: relative(root, path), data: data, source: source})
	}
	universes := make([]universeFile, 0, len(universePaths))
	for _, path := range universePaths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, nil, readErr
		}
		source, parseErr := ontologyfabric.ParseUniverseSource(data)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, parseErr)
		}
		universes = append(universes, universeFile{path: relative(root, path), data: data, source: source})
	}
	return modules, universes, nil
}

func moduleSources(files []moduleFile) []ontologyfabric.ModuleSource {
	out := make([]ontologyfabric.ModuleSource, len(files))
	for i := range files {
		out[i] = files[i].source
	}
	return out
}

func validateUniverseModules(universe ontologyfabric.UniverseSource, modules []moduleFile) error {
	available := make(map[string]moduleFile, len(modules))
	for _, module := range modules {
		available[module.source.ModuleID+"@"+module.source.Version] = module
	}
	frames := make(map[string]struct{})
	for _, ref := range universe.ModuleIDs {
		module, ok := available[ref]
		if !ok {
			return fmt.Errorf("universe references missing module %q", ref)
		}
		for _, frame := range module.source.Frames {
			frames[frame.ID] = struct{}{}
		}
	}
	for _, frameID := range universe.FrameTypeIDs {
		if _, ok := frames[frameID]; !ok {
			return fmt.Errorf("universe references frame %q outside its module set", frameID)
		}
	}
	return nil
}

func digestText(digest dataplane.Digest) string { return "sha256:" + hex.EncodeToString(digest[:]) }

func safeName(value string) string {
	return strings.NewReplacer(":", "-", "@", "-", "/", "-").Replace(value)
}

func relative(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return value
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "twirx-ontology:", message)
	os.Exit(1)
}
