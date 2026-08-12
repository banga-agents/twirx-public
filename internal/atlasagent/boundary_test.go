package atlasagent

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestTrustedAgentBoundaryHasNoNetworkProcessOrModelImports(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "agent.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{"net": true, "net/http": true, "os/exec": true, "plugin": true, "unsafe": true}
	for _, imported := range file.Imports {
		path := imported.Path.Value[1 : len(imported.Path.Value)-1]
		if forbidden[path] {
			t.Fatalf("trusted agent imports forbidden package %q", path)
		}
	}
}
