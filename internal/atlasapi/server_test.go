package atlasapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/atlas"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
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
	server, err := New(selection, registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestStatusKeepsOrthogonalCountsAndExcludesFixtures(t *testing.T) {
	recorder := httptest.NewRecorder()
	testServer(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/atlas/status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatal(recorder.Code)
	}
	var response atlas.Metrics
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Atlas.SelectedCandidates != 500 || response.Atlas.PolicyRecords != 3 || response.Atlas.GenesisPublicRecords != 3 || response.Atlas.TestFixtureRecords != 1 || response.Atlas.CatalogState["cataloged"] != 3 || response.Atlas.PolicyReviewState["completed"] != 3 || response.Atlas.TechnicalStage["semantically_linked"] != 2 {
		t.Fatalf("overstated or conflated status: %#v", response.Atlas)
	}
	if response.Capabilities.AdmittedPublicReadOperations != 4 || response.Capabilities.CommercialOfferCandidates != 0 || response.Capabilities.InterfaceKinds["browser"] != 0 || response.Capabilities.InterfaceKinds["webmcp"] != 0 {
		t.Fatalf("overstated executable or commercial surface: %#v", response.Capabilities)
	}
}

func TestOriginListIsBoundedFilteredAndPaginated(t *testing.T) {
	recorder := httptest.NewRecorder()
	testServer(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins?family=government_law_public_data&limit=10&cursor=5&catalog_state=candidate&technical_stage=unprofiled", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Origins []OriginView `json:"origins"`
		Total   int          `json:"total"`
		Next    *int         `json:"next_cursor"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Origins) != 10 || response.Total != 100 || response.Next == nil || *response.Next != 15 {
		t.Fatalf("unexpected page: %#v", response)
	}
	for _, origin := range response.Origins {
		if origin.Catalog.State != atlas.CatalogCandidate || origin.Policy.ReviewState != atlas.PolicyPending || origin.Technical.Stage != atlas.TechnicalUnprofiled || origin.Publisher.Status != atlas.PublisherUnclaimed || origin.Health.Status != atlas.HealthUnknown {
			t.Fatalf("candidate promoted: %#v", origin)
		}
	}
}

func TestDescribeTWIRXShowsIndependentE2AndPolicyStates(t *testing.T) {
	recorder := httptest.NewRecorder()
	testServer(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins/twirx-org", nil))
	if recorder.Code != http.StatusOK {
		t.Fatal(recorder.Code)
	}
	var origin OriginView
	if err := json.Unmarshal(recorder.Body.Bytes(), &origin); err != nil {
		t.Fatal(err)
	}
	if origin.Catalog.State != atlas.CatalogCataloged || origin.Policy.ReviewState != atlas.PolicyCompleted || origin.Policy.Decision != atlas.DecisionPermitLive || origin.Technical.Stage != atlas.TechnicalSemanticallyLinked || origin.Publisher.Status != atlas.PublisherApproved || origin.Health.Status != atlas.HealthUnknown || origin.CanonicalOrigin != "https://twirx.org" {
		t.Fatalf("unexpected origin: %#v", origin)
	}
	if len(origin.Interfaces) != 4 || len(origin.Capabilities) != 3 || origin.Capabilities[0].EffectClass != atlas.EffectPublicRead || origin.Capabilities[0].Status != atlas.CapabilityAdmitted {
		t.Fatalf("unexpected descriptive capability surface: %#v", origin)
	}
}

func TestControlledFixtureIsExcludedFromPublicOriginSurface(t *testing.T) {
	recorder := httptest.NewRecorder()
	testServer(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins/controlled-origin-lab-fixture", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("fixture entered public origin surface: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestAPIRejectsWritesAndUnknownOrLinearQueries(t *testing.T) {
	server := testServer(t)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v1/atlas/origins", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins?url=https://example.com", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins?maturity=A2", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins?policy_review_state=review_required", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/atlas/origins?limit=1000", nil),
	} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		if recorder.Code < 400 {
			t.Fatalf("unsafe request accepted: %s %s", request.Method, request.URL)
		}
	}
}
