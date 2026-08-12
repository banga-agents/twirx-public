package opportunityrelease_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/opportunityrelease"
	"github.com/typed-web-commons/typed-web/internal/universeimport"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

func TestOpenPublicRuntimeBindsManifestPrivacyAndSegment(t *testing.T) {
	repository := repositoryRoot(t)
	grantPath := filepath.Join(repository, "conformance", "e4-importers", "grants-fetch-controlled.json")
	worldPath := filepath.Join(repository, "origins", "fixtures", "world-bank-chl-population-2024.json")
	grantData := mustRead(t, grantPath)
	worldData := mustRead(t, worldPath)
	grantRecords, err := universeimport.CompileGrantsFetch(grantData, config(universeimport.GrantsGovOriginID, grantPath, grantData, "grant"))
	if err != nil {
		t.Fatal(err)
	}
	worldRecords, err := universeimport.CompileWorldBank(worldData, config(universeimport.WorldBankOriginID, worldPath, worldData, "world"))
	if err != nil {
		t.Fatal(err)
	}
	source := []universesnapshot.SourceFrame{
		{Digest: grantRecords[0].FrameDigest, CBOR: grantRecords[0].FrameCBOR, Frame: grantRecords[0].Frame},
		{Digest: worldRecords[0].FrameDigest, CBOR: worldRecords[0].FrameCBOR, Frame: worldRecords[0].Frame},
	}
	segment, segmentDigest, err := universesnapshot.BuildCompact(source)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "segments"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "reports"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "segments", "combined.twux"), segment, 0o440); err != nil {
		t.Fatal(err)
	}
	privacy := opportunityrelease.PrivacyReport{
		Format: opportunityrelease.PrivacyFormat, EligibilityFieldsWithheld: 1,
		PublisherNonEndorsementNotice: "not endorsed", ProjectionWithholdingReason: "private text withheld",
	}
	privacyBytes, err := json.MarshalIndent(privacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	privacyBytes = append(privacyBytes, '\n')
	if err := os.WriteFile(filepath.Join(root, "reports", "privacy.json"), privacyBytes, 0o440); err != nil {
		t.Fatal(err)
	}
	fixed := digestText(sha256.Sum256([]byte("fixed")))
	manifest := opportunityrelease.Manifest{
		Format: opportunityrelease.ReleaseFormat, OriginID: universeimport.GrantsGovOriginID, UniverseID: "tw:opportunity", CompiledAt: "2026-08-12T04:49:03Z", EvidenceClass: "current_observation",
		WorkOrderDigest: fixed, PolicyDecisionDigest: fixed, AcquisitionManifestDigest: fixed, ArchiveDigest: fixed, XMLDigest: fixed, PrivateProjectionDigest: fixed, ModuleSetDigest: fixed, WorldStateReleaseDigest: fixed,
		SourceRecordsSeen: 1, SourceRecordsAccepted: 1, Packets: 1, MappingClaims: 1, Frames: 1, WorldStateFrames: 1, CombinedFrames: 2, ArtifactSegments: 2,
		TrustLane: "provisional_semantic", MappingStatus: "candidate", EligibilityTextWithheld: true, RuntimeModelAuthority: "none",
		Artifacts: []opportunityrelease.Artifact{
			{Kind: "privacy_report", Path: "reports/privacy.json", Digest: digestText(dataplane.DigestBytes(privacyBytes)), Size: uint64(len(privacyBytes))},
			{Kind: "combined_frame_segment", Path: "segments/combined.twux", Digest: fixed, Size: 1, Entries: 2},
			{Kind: "mapping_segment", Path: "segments/mapping.twas", Digest: fixed, Size: 1, Entries: 1},
			{Kind: "opportunity_frame_segment", Path: "segments/opportunity.twux", Digest: fixed, Size: 1, Entries: 1},
			{Kind: "packet_segment", Path: "segments/packet.twas", Digest: fixed, Size: 1, Entries: 1},
		},
	}
	manifest.Artifacts[1] = opportunityrelease.Artifact{Kind: "combined_frame_segment", Path: "segments/combined.twux", Digest: digestText(segmentDigest), Size: uint64(len(segment)), Entries: 2}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(root, "release-manifest.json"), manifestBytes, 0o440); err != nil {
		t.Fatal(err)
	}
	identity := digestText(dataplane.DigestBytes(manifestBytes))
	opened, admitted, err := opportunityrelease.OpenPublicRuntime(root, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if admitted.CombinedFrames != 2 || opened.FrameCount() != 2 {
		t.Fatal("public runtime did not preserve manifest counts")
	}
	result, err := opened.Query(universesnapshot.Query{UniverseID: "tw:opportunity", FrameType: "opportunity:GrantOpportunity", Limit: 1})
	if err != nil || len(result) != 1 {
		t.Fatalf("public Opportunity query = %x, %v", result, err)
	}
	if _, _, err := opportunityrelease.OpenPublicRuntime(root, fixed); err == nil {
		t.Fatal("public runtime accepted an untrusted manifest identity")
	}
	if err := os.Remove(filepath.Join(root, "reports", "privacy.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := opportunityrelease.OpenPublicRuntime(root, identity); err == nil {
		t.Fatal("public runtime opened without its privacy report")
	}
}

func config(origin, path string, representation []byte, label string) universeimport.Config {
	return universeimport.Config{
		OriginID: origin, ObservedAt: "2026-08-12T00:00:00Z", RepresentationDigest: dataplane.DigestBytes(representation),
		ObservationDigest: sha256.Sum256([]byte(label + "/observation")), ModuleSetDigest: sha256.Sum256([]byte(label + "/module")),
		EvidenceClass: "test_fixture", EvidenceRef: path, EvidenceStored: true,
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func digestText(digest [32]byte) string { return "sha256:" + hex.EncodeToString(digest[:]) }
