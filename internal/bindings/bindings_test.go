package bindings

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/origincatalog"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

func testContracts(t *testing.T) *twircontract.Set {
	t.Helper()
	set, err := twircontract.Load(filepath.Join("..", "..", "contracts", "e2", "contracts.json"))
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestAllBindingsDeriveFromContract(t *testing.T) {
	set := testContracts(t)
	tools, err := Tools(set)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != len(set.Operations) || len(tools) < 5 {
		t.Fatalf("tools %d operations %d", len(tools), len(set.Operations))
	}
	openapi, err := OpenAPI(set)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range set.Operations {
		if !bytes.Contains(openapi, []byte(op.ID)) {
			t.Fatalf("OpenAPI missing %s", op.ID)
		}
	}
	if bytes.Contains(openapi, []byte(`"fresh"`)) || !bytes.Contains(openapi, []byte(`"const": "replay"`)) {
		t.Fatal("public OpenAPI must describe the replay-only HTTP surface")
	}
	dir := t.TempDir()
	if err := Write(dir, set); err != nil {
		t.Fatal(err)
	}
	catalog, err := origincatalog.Load(filepath.Join("..", "..", "origins", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WritePublicProof(dir, set, catalog); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"openapi.json", "mcp-tools.json", "cli.txt", "public-proof.json", "json-schema/project.getStatus.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
}
