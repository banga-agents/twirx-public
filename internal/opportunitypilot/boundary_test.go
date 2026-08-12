package opportunitypilot

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpportunityBoundaryHasNoProcessPluginUnsafeOrTrustedParserNetwork(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{"os/exec": true, "plugin": true, "unsafe": true, "C": true}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(".", entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			name := strings.Trim(imported.Path.Value, `"`)
			if forbidden[name] {
				t.Fatalf("%s imports forbidden package %q", entry.Name(), name)
			}
			if entry.Name() == "project.go" && (name == "net" || name == "net/http" || name == "github.com/typed-web-commons/typed-web/internal/safefetch") {
				t.Fatalf("offline projection parser imports network package %q", name)
			}
		}
	}
}
