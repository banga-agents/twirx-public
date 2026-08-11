package atlas

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAtlasControlPlaneHasNoNetworkOrExecutionClient(t *testing.T) {
	root := repositoryRoot(t)
	paths := []string{filepath.Join(root, "internal", "atlas"), filepath.Join(root, "internal", "atlasapi"), filepath.Join(root, "cmd", "twirx-atlas")}
	for _, path := range paths {
		entries, err := os.ReadDir(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			inspectBoundaryFile(t, filepath.Join(path, entry.Name()), path == paths[0])
		}
	}
}

func inspectBoundaryFile(t *testing.T, path string, core bool) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range file.Imports {
		name, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if name == "os/exec" || name == "plugin" || name == "unsafe" || core && (name == "net" || name == "net/http") {
			t.Fatalf("forbidden Atlas control-plane import %q in %s", name, path)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok && identifier.Name == "http" {
			switch selector.Sel.Name {
			case "Client", "DefaultClient", "DefaultTransport", "Get", "Head", "NewRequest", "NewRequestWithContext", "Post", "PostForm", "Transport":
				t.Fatalf("HTTP client capability %s in %s", selector.Sel.Name, path)
			}
		}
		return true
	})
}
