// twirx-e4-agent runs bounded curated queries over either controlled fixtures
// or a manifest-identified, immutable public Utility Universe release. It has
// no origin-network, browser, model-authority, or write path.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/typed-web-commons/typed-web/internal/atlasagent"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/opportunityrelease"
	"github.com/typed-web-commons/typed-web/internal/universeimport"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

type output struct {
	Format             string               `json:"format"`
	EvidenceClass      string               `json:"evidence_class"`
	CurrentClaimsMade  bool                 `json:"current_claims_made"`
	FixtureCountPublic bool                 `json:"fixture_counted_public"`
	Execution          atlasagent.Execution `json:"execution"`
}

type investigationOutput struct {
	Format             string                   `json:"format"`
	EvidenceClass      string                   `json:"evidence_class"`
	CurrentClaimsMade  bool                     `json:"current_claims_made"`
	FixtureCountPublic bool                     `json:"fixture_counted_public"`
	Investigation      atlasagent.Investigation `json:"investigation"`
}

type publicOutput struct {
	Format                    string               `json:"format"`
	EvidenceClass             string               `json:"evidence_class"`
	ReleaseManifestDigest     string               `json:"release_manifest_digest"`
	SourceRecordsAccepted     uint64               `json:"source_records_accepted"`
	SemanticPackets           uint64               `json:"semantic_packets"`
	OpportunityFrames         uint64               `json:"opportunity_frames"`
	CombinedFrames            uint64               `json:"combined_frames"`
	EligibilityTextWithheld   bool                 `json:"eligibility_text_withheld"`
	PublisherEndorsement      bool                 `json:"publisher_endorsement"`
	RuntimeOriginNetworkCalls uint64               `json:"runtime_origin_network_calls"`
	Execution                 atlasagent.Execution `json:"execution"`
}

type publicInvestigationOutput struct {
	Format                    string                   `json:"format"`
	EvidenceClass             string                   `json:"evidence_class"`
	ReleaseManifestDigest     string                   `json:"release_manifest_digest"`
	SemanticPackets           uint64                   `json:"semantic_packets"`
	CombinedFrames            uint64                   `json:"combined_frames"`
	EligibilityTextWithheld   bool                     `json:"eligibility_text_withheld"`
	PublisherEndorsement      bool                     `json:"publisher_endorsement"`
	RuntimeOriginNetworkCalls uint64                   `json:"runtime_origin_network_calls"`
	Investigation             atlasagent.Investigation `json:"investigation"`
}

type benchmarkOutput struct {
	Format                     string        `json:"format"`
	EvidenceClass              string        `json:"evidence_class"`
	ReleaseManifestDigest      string        `json:"release_manifest_digest"`
	ScenarioID                 string        `json:"scenario_id"`
	Iterations                 uint32        `json:"iterations"`
	ResultsPerIteration        uint64        `json:"results_per_iteration"`
	FramesAvailable            uint64        `json:"frames_available"`
	RuntimeAdmissionNanosecond int64         `json:"runtime_admission_nanoseconds"`
	Execution                  durationStats `json:"execution_nanoseconds"`
	NetworkRequests            uint64        `json:"network_requests"`
	BrowserExecutions          uint64        `json:"browser_executions"`
	LiveSourceCalls            uint64        `json:"live_source_calls"`
	ModelAuthority             string        `json:"model_authority"`
}

type durationStats struct {
	Minimum int64 `json:"minimum"`
	Median  int64 `json:"median"`
	P95     int64 `json:"p95"`
	Maximum int64 `json:"maximum"`
	Mean    int64 `json:"mean"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("twirx-e4-agent", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "repository root")
	release := flags.String("release", "", "immutable Opportunity Utility release")
	releaseManifestDigest := flags.String("release-manifest-digest", "", "trusted SHA-256 identity of the release manifest")
	scenarioID := flags.String("scenario", "", "curated scenario ID")
	investigationID := flags.String("investigation", "", "curated multi-universe investigation ID")
	list := flags.Bool("list", false, "list curated scenarios")
	listInvestigations := flags.Bool("list-investigations", false, "list curated investigations")
	benchmarkIterations := flags.Uint("benchmark-iterations", 0, "benchmark a public curated scenario after one runtime admission (1-10000)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if *list {
		return encoder.Encode(atlasagent.CuratedScenarios())
	}
	if *listInvestigations {
		return encoder.Encode(atlasagent.CuratedInvestigations())
	}
	if *scenarioID != "" && *investigationID != "" {
		return errors.New("choose either --scenario or --investigation")
	}
	if *benchmarkIterations > 10000 || *benchmarkIterations > 0 && *investigationID != "" {
		return errors.New("benchmark iterations must be at most 10000 and cannot be combined with an investigation")
	}
	if (*release == "") != (*releaseManifestDigest == "") {
		return errors.New("--release and --release-manifest-digest must be supplied together")
	}
	public := *release != ""
	var engine *atlasagent.Engine
	var manifest opportunityrelease.Manifest
	var admissionDuration time.Duration
	if public {
		admissionStarted := time.Now()
		runtime, admitted, err := opportunityrelease.OpenPublicRuntime(filepath.Clean(*release), *releaseManifestDigest)
		if err != nil {
			return err
		}
		admissionDuration = time.Since(admissionStarted)
		defer runtime.Close()
		manifest = admitted
		engine, err = atlasagent.New(runtime)
		if err != nil {
			return err
		}
	} else {
		if *benchmarkIterations > 0 {
			return errors.New("benchmarking requires an immutable public release")
		}
		var err error
		engine, err = controlledEngine(filepath.Clean(*root))
		if err != nil {
			return err
		}
	}
	if *investigationID != "" {
		investigation, err := engine.ExecuteInvestigation(*investigationID)
		if err != nil {
			if errors.Is(err, atlasagent.ErrUnknownInvestigation) {
				return fmt.Errorf("unknown curated investigation %q", *investigationID)
			}
			return err
		}
		if public {
			return encoder.Encode(publicInvestigationOutput{Format: "tw.e4-agent-public-investigation/0.1", EvidenceClass: manifest.EvidenceClass, ReleaseManifestDigest: *releaseManifestDigest, SemanticPackets: manifest.Packets, CombinedFrames: manifest.CombinedFrames, EligibilityTextWithheld: manifest.EligibilityTextWithheld, PublisherEndorsement: false, RuntimeOriginNetworkCalls: 0, Investigation: investigation})
		}
		return encoder.Encode(investigationOutput{Format: "tw.e4-agent-controlled-investigation/0.1", EvidenceClass: "test_fixture", CurrentClaimsMade: false, FixtureCountPublic: false, Investigation: investigation})
	}
	selected := *scenarioID
	if selected == "" {
		if public {
			selected = "opportunity.source-records-nsf"
		} else {
			selected = "world-state.controlled-development"
		}
	}
	if *benchmarkIterations > 0 {
		durations := make([]time.Duration, *benchmarkIterations)
		var last atlasagent.Execution
		for index := range durations {
			started := time.Now()
			execution, err := engine.Execute(selected)
			durations[index] = time.Since(started)
			if err != nil {
				return err
			}
			last = execution
		}
		return encoder.Encode(benchmarkOutput{Format: "tw.e4-agent-public-benchmark/0.1", EvidenceClass: manifest.EvidenceClass, ReleaseManifestDigest: *releaseManifestDigest, ScenarioID: selected, Iterations: uint32(*benchmarkIterations), ResultsPerIteration: last.ResultCount, FramesAvailable: last.Plan.FramesAvailable, RuntimeAdmissionNanosecond: admissionDuration.Nanoseconds(), Execution: summarizeDurations(durations), NetworkRequests: 0, BrowserExecutions: 0, LiveSourceCalls: 0, ModelAuthority: "none"})
	}
	execution, err := engine.Execute(selected)
	if err != nil {
		if errors.Is(err, atlasagent.ErrUnknownScenario) {
			return fmt.Errorf("unknown curated scenario %q", selected)
		}
		return err
	}
	if public {
		return encoder.Encode(publicOutput{Format: "tw.e4-agent-public-query/0.1", EvidenceClass: manifest.EvidenceClass, ReleaseManifestDigest: *releaseManifestDigest, SourceRecordsAccepted: manifest.SourceRecordsAccepted, SemanticPackets: manifest.Packets, OpportunityFrames: manifest.Frames, CombinedFrames: manifest.CombinedFrames, EligibilityTextWithheld: manifest.EligibilityTextWithheld, PublisherEndorsement: false, RuntimeOriginNetworkCalls: 0, Execution: execution})
	}
	return encoder.Encode(output{Format: "tw.e4-agent-controlled-demo/0.1", EvidenceClass: "test_fixture", CurrentClaimsMade: false, FixtureCountPublic: false, Execution: execution})
}

func summarizeDurations(source []time.Duration) durationStats {
	ordered := append([]time.Duration(nil), source...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	var total int64
	for _, duration := range ordered {
		total += duration.Nanoseconds()
	}
	p95 := (len(ordered)*95 + 99) / 100
	if p95 == 0 {
		p95 = 1
	}
	return durationStats{Minimum: ordered[0].Nanoseconds(), Median: ordered[len(ordered)/2].Nanoseconds(), P95: ordered[p95-1].Nanoseconds(), Maximum: ordered[len(ordered)-1].Nanoseconds(), Mean: total / int64(len(ordered))}
}

func controlledEngine(root string) (*atlasagent.Engine, error) {
	worldPath := filepath.Join(root, "origins", "fixtures", "world-bank-chl-population-2024.json")
	worldBytes, err := os.ReadFile(worldPath)
	if err != nil {
		return nil, err
	}
	worldRecords, err := universeimport.CompileWorldBank(worldBytes, importerConfig(universeimport.WorldBankOriginID, worldPath, worldBytes, "world"))
	if err != nil {
		return nil, err
	}
	grantPath := filepath.Join(root, "conformance", "e4-importers", "grants-fetch-controlled.json")
	grantBytes, err := os.ReadFile(grantPath)
	if err != nil {
		return nil, err
	}
	grantRecords, err := universeimport.CompileGrantsFetch(grantBytes, importerConfig(universeimport.GrantsGovOriginID, grantPath, grantBytes, "grant"))
	if err != nil {
		return nil, err
	}
	source := []universesnapshot.SourceFrame{
		{Digest: worldRecords[0].FrameDigest, CBOR: worldRecords[0].FrameCBOR, Frame: worldRecords[0].Frame},
		{Digest: grantRecords[0].FrameDigest, CBOR: grantRecords[0].FrameCBOR, Frame: grantRecords[0].Frame},
	}
	data, digest, err := universesnapshot.BuildNative(source)
	if err != nil {
		return nil, err
	}
	runtime, err := universesnapshot.OpenNative(data, digest)
	if err != nil {
		return nil, err
	}
	return atlasagent.New(runtime)
}

func importerConfig(origin, path string, representation []byte, label string) universeimport.Config {
	return universeimport.Config{
		OriginID:             origin,
		ObservedAt:           "2026-08-12T00:00:00Z",
		RepresentationDigest: dataplane.DigestBytes(representation),
		ObservationDigest:    sha256.Sum256([]byte(label + "/controlled-observation")),
		ModuleSetDigest:      sha256.Sum256([]byte(label + "/draft-module-set")),
		EvidenceClass:        "test_fixture",
		EvidenceRef:          path,
		EvidenceStored:       true,
	}
}
