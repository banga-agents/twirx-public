package atlasagent

import (
	"crypto/sha256"
	"os"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/universeimport"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

func TestCuratedAgentExecutesOnlyTypedImmutableQueries(t *testing.T) {
	runtime := testRuntime(t)
	engine, err := New(runtime)
	if err != nil {
		t.Fatal(err)
	}

	world, err := engine.Execute("world-state.controlled-development")
	if err != nil {
		t.Fatal(err)
	}
	if world.Status != "resolved" || world.ResultCount != 1 || !world.ProofLinked || len(world.Results[0].Slots) != 5 {
		t.Fatalf("world execution = %+v", world)
	}
	if world.Plan.NetworkRequests != 0 || world.Plan.BrowserExecutions != 0 || world.Plan.LiveSourceCalls != 0 || world.Plan.ModelAuthority != "none" || world.Plan.ExecutionAuthority != "validated_typed_query_over_immutable_snapshot" {
		t.Fatalf("world plan crossed authority boundary: %+v", world.Plan)
	}
	for _, slot := range world.Results[0].Slots {
		if len(slot.PacketDigests) == 0 {
			t.Fatalf("slot %s lost packet proof", slot.RoleID)
		}
	}

	opportunity, err := engine.Execute("opportunity.controlled-funding")
	if err != nil {
		t.Fatal(err)
	}
	if opportunity.Status != "resolved" || opportunity.ResultCount != 1 {
		t.Fatalf("opportunity execution = %+v", opportunity)
	}
	deadlineFound := false
	for _, slot := range opportunity.Results[0].Slots {
		if slot.RoleID == "opportunity:deadline" {
			deadlineFound = true
			if slot.Status != "unresolved" || len(slot.Values) != 0 || len(slot.PacketDigests) != 1 {
				t.Fatalf("deadline semantics were guessed or lost: %+v", slot)
			}
		}
	}
	if !deadlineFound {
		t.Fatal("opportunity result omitted explicit deadline state")
	}

	research, err := engine.Execute("research.evidence-discovery")
	if err != nil {
		t.Fatal(err)
	}
	if research.Status != "unresolved" || research.ResultCount != 0 || research.ProofLinked {
		t.Fatalf("unavailable universe became a claim: %+v", research)
	}
	if _, err := engine.Execute("unknown"); err != ErrUnknownScenario {
		t.Fatalf("unknown scenario error = %v", err)
	}
	publicOpportunity, err := engine.Execute("opportunity.source-records-nsf")
	if err != nil || publicOpportunity.Status != "unresolved" {
		t.Fatalf("source scenario guessed absent NSF records: %+v, %v", publicOpportunity, err)
	}
}

func TestCuratedScenarioRegistryIsStable(t *testing.T) {
	scenarios := CuratedScenarios()
	if len(scenarios) != 7 {
		t.Fatalf("scenario count = %d", len(scenarios))
	}
	for index := 1; index < len(scenarios); index++ {
		if scenarios[index-1].ID >= scenarios[index].ID {
			t.Fatal("scenario registry is not deterministic")
		}
	}
}

func TestControlledInvestigationCoordinatesWithoutInventingJoin(t *testing.T) {
	engine, err := New(testRuntime(t))
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.ExecuteInvestigation("utility.controlled-world-and-opportunity")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "resolved" || len(result.Executions) != 2 || len(result.Universes) != 2 || result.ResultCount != 2 || result.ProofLinkedCount != 2 {
		t.Fatalf("investigation = %+v", result)
	}
	if result.NetworkRequests != 0 || result.BrowserExecutions != 0 || result.LiveSourceCalls != 0 || result.ModelAuthority != "none" {
		t.Fatalf("investigation crossed an authority boundary: %+v", result)
	}
	if _, err := engine.ExecuteInvestigation("unknown"); err != ErrUnknownInvestigation {
		t.Fatalf("unknown investigation error = %v", err)
	}
}

func testRuntime(t *testing.T) *universesnapshot.NativeRuntime {
	t.Helper()
	worldBytes, err := os.ReadFile("../../origins/fixtures/world-bank-chl-population-2024.json")
	if err != nil {
		t.Fatal(err)
	}
	worldRecords, err := universeimport.CompileWorldBank(worldBytes, importerConfig(universeimport.WorldBankOriginID, worldBytes, "world"))
	if err != nil {
		t.Fatal(err)
	}
	grantBytes, err := os.ReadFile("../../conformance/e4-importers/grants-fetch-controlled.json")
	if err != nil {
		t.Fatal(err)
	}
	grantRecords, err := universeimport.CompileGrantsFetch(grantBytes, importerConfig(universeimport.GrantsGovOriginID, grantBytes, "grant"))
	if err != nil {
		t.Fatal(err)
	}
	source := []universesnapshot.SourceFrame{
		{Digest: worldRecords[0].FrameDigest, CBOR: worldRecords[0].FrameCBOR, Frame: worldRecords[0].Frame},
		{Digest: grantRecords[0].FrameDigest, CBOR: grantRecords[0].FrameCBOR, Frame: grantRecords[0].Frame},
	}
	data, digest, err := universesnapshot.BuildNative(source)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := universesnapshot.OpenNative(data, digest)
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func importerConfig(origin string, representation []byte, label string) universeimport.Config {
	return universeimport.Config{
		OriginID:             origin,
		ObservedAt:           "2026-08-12T00:00:00Z",
		RepresentationDigest: dataplane.DigestBytes(representation),
		ObservationDigest:    sha256.Sum256([]byte(label + "/observation")),
		ModuleSetDigest:      sha256.Sum256([]byte(label + "/modules")),
		EvidenceClass:        "test_fixture",
		EvidenceRef:          "controlled-test-fixture",
		EvidenceStored:       true,
	}
}
