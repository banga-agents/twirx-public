package origincatalog

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/atlas"
	"github.com/typed-web-commons/typed-web/internal/twircontract"
)

func TestCatalogBuildsOnlyReviewedEndpoints(t *testing.T) {
	catalog, err := Load(filepath.Join("..", "..", "origins", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	contracts, err := twircontract.Load(filepath.Join("..", "..", "contracts", "e2", "contracts.json"))
	if err != nil {
		t.Fatal(err)
	}
	op, err := contracts.Find("development.getIndicator")
	if err != nil {
		t.Fatal(err)
	}
	origin, err := catalog.ForOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	requestURL, err := origin.RequestURL(op, map[string]string{"country": "CHL", "indicator": "SP.POP.TOTL", "year": "2024"})
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://api.worldbank.org/v2/country/CHL/indicator/SP.POP.TOTL?date=2024&format=json&per_page=1" {
		t.Fatalf("unexpected URL %s", requestURL)
	}
	for _, country := range []string{"127.0.0.1", "example.com/../../", "CHL@evil.example"} {
		if _, err := origin.RequestURL(op, map[string]string{"country": country, "indicator": "SP.POP.TOTL", "year": "2024"}); err == nil {
			t.Fatalf("accepted %q", country)
		}
	}
}

func TestCatalogBindsCanonicalRegistryIdentityAndFixtureScope(t *testing.T) {
	root := filepath.Join("..", "..")
	selection, err := atlas.LoadSelection(filepath.Join(root, "atlas", "genesis-500", "selection.json"))
	if err != nil {
		t.Fatal(err)
	}
	policies, err := atlas.LoadPolicySet(filepath.Join(root, "atlas", "policies.json"), selection)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := atlas.LoadRegistry(filepath.Join(root, "atlas", "registry.json"), selection, policies)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(filepath.Join(root, "origins", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.ValidateRegistry(registry); err != nil {
		t.Fatal(err)
	}
	catalog.Origins[0].RegistryID = "twirx-org"
	if err := catalog.ValidateRegistry(registry); err == nil {
		t.Fatal("controlled fixture accepted under a public-origin canonical identity")
	}
}

func TestEveryContractHasExactlyOneCatalogOwner(t *testing.T) {
	catalog, err := Load(filepath.Join("..", "..", "origins", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	contracts, err := twircontract.Load(filepath.Join("..", "..", "contracts", "e2", "contracts.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range contracts.Operations {
		origin, findErr := catalog.ForOperation(op.ID)
		if findErr != nil {
			t.Fatal(findErr)
		}
		if origin.ID != op.OriginID {
			t.Fatalf("operation %s owner mismatch", op.ID)
		}
	}
}

func TestCatalogRejectsUnsafeFixture(t *testing.T) {
	catalog, err := Load(filepath.Join("..", "..", "origins", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog.Origins[0].ReplayFixture = "../private"
	if err := catalog.Validate(); err == nil || !strings.Contains(err.Error(), "unsafe fixture") {
		t.Fatalf("got %v", err)
	}
}
