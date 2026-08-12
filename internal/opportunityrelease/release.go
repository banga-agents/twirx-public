// Package opportunityrelease compiles the verified private Grants.gov
// projection into a privacy-safe, immutable public Opportunity Universe
// release and a combined World State + Opportunity query segment.
package opportunityrelease

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/typed-web-commons/typed-web/internal/artifactsegment"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
	"github.com/typed-web-commons/typed-web/internal/opportunitypilot"
	"github.com/typed-web-commons/typed-web/internal/universeimport"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
	"github.com/typed-web-commons/typed-web/internal/worldstatepilot"
)

const (
	ReleaseFormat    = "tw.e4-opportunity-release/0.1"
	PrivacyFormat    = "tw.e4-opportunity-privacy-report/0.1"
	CompilationBatch = 4096
	maxManifest      = 4 << 20
	maxPrivacy       = 1 << 20
)

var (
	emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	urlPattern   = regexp.MustCompile(`(?i)https?://[^\s<>]+`)
	phonePattern = regexp.MustCompile(`\+?[0-9][0-9() .\-]{7,}[0-9]`)
)

type Artifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Digest  string `json:"digest"`
	Size    uint64 `json:"size"`
	Entries uint64 `json:"entries,omitempty"`
}

type PrivacyReport struct {
	Format                            string `json:"format"`
	ProjectionPublic                  bool   `json:"projection_public"`
	RawEvidencePublic                 bool   `json:"raw_evidence_public"`
	ContactFieldsExcluded             uint64 `json:"contact_fields_excluded"`
	DescriptionFieldsExcluded         uint64 `json:"description_fields_excluded"`
	EligibilityFieldsWithheld         uint64 `json:"eligibility_fields_withheld"`
	EmailLikeOccurrencesInProjection  uint64 `json:"email_like_occurrences_in_private_projection"`
	URLLikeOccurrencesInProjection    uint64 `json:"url_like_occurrences_in_private_projection"`
	PhoneLikeOccurrencesInProjection  uint64 `json:"phone_like_occurrences_in_private_projection"`
	EligibilityLexicalValuesInRelease uint64 `json:"eligibility_lexical_values_in_release"`
	PublisherNonEndorsementNotice     string `json:"publisher_non_endorsement_notice"`
	ProjectionWithholdingReason       string `json:"projection_withholding_reason"`
}

type Manifest struct {
	Format                    string     `json:"format"`
	OriginID                  string     `json:"origin_id"`
	UniverseID                string     `json:"universe_id"`
	CompiledAt                string     `json:"compiled_at"`
	EvidenceClass             string     `json:"evidence_class"`
	WorkOrderDigest           string     `json:"work_order_digest"`
	PolicyDecisionDigest      string     `json:"policy_decision_digest"`
	AcquisitionManifestDigest string     `json:"acquisition_manifest_digest"`
	ArchiveDigest             string     `json:"archive_digest"`
	XMLDigest                 string     `json:"xml_digest"`
	PrivateProjectionDigest   string     `json:"private_projection_digest"`
	ModuleSetDigest           string     `json:"module_set_digest"`
	WorldStateReleaseDigest   string     `json:"world_state_release_digest"`
	SourceRecordsSeen         uint64     `json:"source_records_seen"`
	SourceRecordsAccepted     uint64     `json:"source_records_accepted"`
	SourceRecordsRejected     uint64     `json:"source_records_rejected"`
	Packets                   uint64     `json:"packets"`
	MappingClaims             uint64     `json:"mapping_claims"`
	Frames                    uint64     `json:"frames"`
	WorldStateFrames          uint64     `json:"world_state_frames"`
	CombinedFrames            uint64     `json:"combined_frames"`
	ArtifactSegments          uint64     `json:"artifact_segments"`
	TrustLane                 string     `json:"trust_lane"`
	MappingStatus             string     `json:"mapping_status"`
	SchedulerEnabled          bool       `json:"scheduler_enabled"`
	RawEvidencePublic         bool       `json:"raw_evidence_public"`
	PrivateProjectionPublic   bool       `json:"private_projection_public"`
	EligibilityTextWithheld   bool       `json:"eligibility_text_withheld"`
	RuntimeOriginNetworkCalls uint64     `json:"runtime_origin_network_calls"`
	RuntimeBrowserExecutions  uint64     `json:"runtime_browser_executions"`
	RuntimeModelAuthority     string     `json:"runtime_model_authority"`
	Artifacts                 []Artifact `json:"artifacts"`
}

// OpenPublicRuntime opens only the immutable combined query segment from a
// release whose manifest identity is supplied by the caller. It deliberately
// does not read private acquisition or projection state and grants no origin
// network authority.
func OpenPublicRuntime(releaseRoot, expectedManifestDigest string) (*universesnapshot.CompactRuntime, Manifest, error) {
	var manifest Manifest
	expected, err := parseDigest(expectedManifestDigest)
	if err != nil {
		return nil, manifest, err
	}
	manifestBytes, err := readRegular(filepath.Join(releaseRoot, "release-manifest.json"), maxManifest)
	if err != nil || dataplane.DigestBytes(manifestBytes) != expected {
		return nil, manifest, errors.New("opportunity release: public manifest identity mismatch")
	}
	policy := jsonbounded.Policy{MaxBytes: maxManifest, MaxDepth: 12, MaxScalarBytes: 64 << 10, MaxContainerEntries: 20000, MaxTokens: 100000}
	if err := jsonbounded.Decode(manifestBytes, &manifest, policy, true); err != nil {
		return nil, Manifest{}, err
	}
	combined, privacy, err := validatePublicManifest(manifest)
	if err != nil {
		return nil, Manifest{}, err
	}
	if _, err := os.Stat(filepath.Join(releaseRoot, "approved-projection.json")); !errors.Is(err, os.ErrNotExist) {
		return nil, Manifest{}, errors.New("opportunity release: private projection entered public release")
	}
	privacyBytes, err := readRegular(filepath.Join(releaseRoot, privacy.Path), privacy.Size)
	if err != nil || uint64(len(privacyBytes)) != privacy.Size {
		return nil, Manifest{}, errors.New("opportunity release: public privacy report unavailable")
	}
	privacyDigest, err := parseDigest(privacy.Digest)
	if err != nil || dataplane.DigestBytes(privacyBytes) != privacyDigest {
		return nil, Manifest{}, errors.New("opportunity release: public privacy report identity mismatch")
	}
	var report PrivacyReport
	if err := jsonbounded.Decode(privacyBytes, &report, jsonbounded.Policy{MaxBytes: maxPrivacy, MaxDepth: 4, MaxScalarBytes: 4096, MaxContainerEntries: 64, MaxTokens: 256}, true); err != nil || report.Format != PrivacyFormat || report.ProjectionPublic || report.RawEvidencePublic || report.EligibilityLexicalValuesInRelease != 0 || report.EligibilityFieldsWithheld == 0 {
		return nil, Manifest{}, errors.New("opportunity release: public privacy report is invalid")
	}
	combinedDigest, err := parseDigest(combined.Digest)
	if err != nil {
		return nil, Manifest{}, err
	}
	path := filepath.Join(releaseRoot, combined.Path)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != int64(combined.Size) {
		return nil, Manifest{}, errors.New("opportunity release: combined public segment unavailable")
	}
	runtime, err := universesnapshot.OpenCompactFile(path, combinedDigest)
	if err != nil {
		return nil, Manifest{}, err
	}
	if runtime.FrameCount() != manifest.CombinedFrames {
		_ = runtime.Close()
		return nil, Manifest{}, errors.New("opportunity release: combined public frame count mismatch")
	}
	return runtime, manifest, nil
}

func validatePublicManifest(manifest Manifest) (Artifact, Artifact, error) {
	var combined, privacy Artifact
	var opportunity Artifact
	if manifest.Format != ReleaseFormat || manifest.OriginID != universeimport.GrantsGovOriginID || manifest.UniverseID != "tw:opportunity" || manifest.EvidenceClass != "current_observation" || manifest.SourceRecordsSeen == 0 || manifest.SourceRecordsAccepted == 0 || manifest.SourceRecordsAccepted+manifest.SourceRecordsRejected != manifest.SourceRecordsSeen || manifest.Frames != manifest.SourceRecordsAccepted || manifest.WorldStateFrames == 0 || manifest.CombinedFrames != manifest.Frames+manifest.WorldStateFrames || manifest.Packets == 0 || manifest.MappingClaims == 0 || manifest.ArtifactSegments == 0 || manifest.SchedulerEnabled || manifest.RawEvidencePublic || manifest.PrivateProjectionPublic || !manifest.EligibilityTextWithheld || manifest.RuntimeOriginNetworkCalls != 0 || manifest.RuntimeBrowserExecutions != 0 || manifest.RuntimeModelAuthority != "none" || manifest.TrustLane != "provisional_semantic" || manifest.MappingStatus != "candidate" || len(manifest.Artifacts) == 0 {
		return combined, privacy, errors.New("opportunity release: public manifest invariants are invalid")
	}
	compiledAt, err := time.Parse(time.RFC3339, manifest.CompiledAt)
	if err != nil || compiledAt.Location() != time.UTC {
		return combined, privacy, errors.New("opportunity release: public manifest time is invalid")
	}
	for _, value := range []string{manifest.WorkOrderDigest, manifest.PolicyDecisionDigest, manifest.AcquisitionManifestDigest, manifest.ArchiveDigest, manifest.XMLDigest, manifest.PrivateProjectionDigest, manifest.ModuleSetDigest, manifest.WorldStateReleaseDigest} {
		if _, err := parseDigest(value); err != nil {
			return combined, privacy, err
		}
	}
	artifactSegments := uint64(0)
	packetEntries := uint64(0)
	mappingEntries := uint64(0)
	for index, artifact := range manifest.Artifacts {
		if artifact.Path == "" || filepath.IsAbs(artifact.Path) || filepath.Clean(artifact.Path) != artifact.Path || strings.Contains(artifact.Path, "..") || index > 0 && manifest.Artifacts[index-1].Path >= artifact.Path || artifact.Size == 0 {
			return combined, privacy, errors.New("opportunity release: unsafe or unsorted public artifact")
		}
		if _, err := parseDigest(artifact.Digest); err != nil {
			return combined, privacy, err
		}
		switch artifact.Kind {
		case "packet_segment", "mapping_segment":
			if artifact.Entries == 0 || artifact.Entries > artifactsegment.MaxEntries || artifact.Size > artifactsegment.MaxSegmentBytes {
				return combined, privacy, errors.New("opportunity release: public artifact-segment bounds are invalid")
			}
			artifactSegments++
			if artifact.Kind == "packet_segment" {
				packetEntries += artifact.Entries
			} else {
				mappingEntries += artifact.Entries
			}
		case "opportunity_frame_segment":
			if opportunity.Path != "" || artifact.Entries != manifest.Frames || artifact.Size > universesnapshot.MaxBytes {
				return combined, privacy, errors.New("opportunity release: Opportunity frame artifact is invalid")
			}
			opportunity = artifact
		case "combined_frame_segment":
			if combined.Path != "" || artifact.Entries != manifest.CombinedFrames || artifact.Size > universesnapshot.MaxBytes {
				return combined, privacy, errors.New("opportunity release: combined frame artifact is invalid")
			}
			combined = artifact
		case "privacy_report":
			if privacy.Path != "" || artifact.Entries != 0 || artifact.Size > maxPrivacy {
				return combined, privacy, errors.New("opportunity release: privacy artifact is invalid")
			}
			privacy = artifact
		default:
			return combined, privacy, errors.New("opportunity release: unknown public artifact kind")
		}
	}
	if artifactSegments != manifest.ArtifactSegments || packetEntries != manifest.Packets || mappingEntries != manifest.MappingClaims || opportunity.Path == "" || combined.Path == "" || privacy.Path == "" {
		return combined, privacy, errors.New("opportunity release: required public artifacts are missing")
	}
	return combined, privacy, nil
}

func Build(root, acquisitionRoot, projectionRoot, worldReleaseRoot, output string) (Manifest, error) {
	var release Manifest
	loaded, err := opportunitypilot.LoadWorkOrder(filepath.Join(root, "atlas", "e4-plans", "grants-gov-20260811.json"))
	if err != nil || opportunitypilot.VerifyAuthority(root, loaded) != nil {
		return release, errors.New("opportunity release: exact founder authority is unavailable")
	}
	acquisition, err := opportunitypilot.VerifyAcquisition(acquisitionRoot, loaded)
	if err != nil {
		return release, err
	}
	projectionManifest, projection, err := opportunitypilot.VerifyProjection(projectionRoot, acquisitionRoot, loaded)
	if err != nil {
		return release, err
	}
	worldRelease, err := worldstatepilot.VerifyRelease(root, worldReleaseRoot)
	if err != nil {
		return release, fmt.Errorf("opportunity release: verify World State release: %w", err)
	}
	moduleSet, err := loadModuleSet(root)
	if err != nil {
		return release, err
	}
	projectionDigest, err := parseDigest(projectionManifest.ProjectionDigest)
	if err != nil {
		return release, err
	}
	xmlDigest, err := parseDigest(projectionManifest.XMLDigest)
	if err != nil {
		return release, err
	}
	policyDigest, err := parseDigest(acquisition.PolicyDecisionDigest)
	if err != nil {
		return release, err
	}
	acquisitionManifestBytes, err := os.ReadFile(filepath.Join(acquisitionRoot, "acquisition-manifest.json"))
	if err != nil || len(acquisitionManifestBytes) == 0 || len(acquisitionManifestBytes) > maxManifest {
		return release, errors.New("opportunity release: acquisition manifest unavailable")
	}
	observationDigest := dataplane.DigestBytes(acquisitionManifestBytes)
	worldManifestBytes, err := os.ReadFile(filepath.Join(worldReleaseRoot, "release-manifest.json"))
	if err != nil || len(worldManifestBytes) == 0 || len(worldManifestBytes) > maxManifest {
		return release, errors.New("opportunity release: World State manifest unavailable")
	}
	rootOut, err := createOutput(output)
	if err != nil {
		return release, err
	}
	config := universeimport.Config{
		OriginID: universeimport.GrantsGovOriginID, ObservedAt: acquisition.CompletedAt,
		RepresentationDigest: projectionDigest, ObservationDigest: observationDigest,
		ModuleSetDigest: moduleSet, PolicyDecisionDigest: dataplane.OptionalDigest{Present: true, Value: policyDigest},
		EvidenceClass: "current_observation", EvidenceRef: "private-projection:" + projectionManifest.ProjectionDigest, EvidenceStored: true,
	}
	var opportunityFrames []universesnapshot.SourceFrame
	var total uint64
	segmentSequence := uint64(0)
	withheld := uint64(0)
	for start := uint64(0); total == 0 || start < total; start += CompilationBatch {
		records, discovered, compileErr := universeimport.CompileGrantsBulkProjectionWindow(projection, xmlDigest, config, start, CompilationBatch)
		if compileErr != nil {
			return Manifest{}, fmt.Errorf("opportunity release: compile window %d: %w", start, compileErr)
		}
		if total == 0 {
			total = discovered
		} else if total != discovered {
			return Manifest{}, errors.New("opportunity release: projection count changed between windows")
		}
		packetEntries := make([]artifactsegment.Entry, 0)
		mappingEntries := make([]artifactsegment.Entry, 0)
		for _, record := range records {
			for _, packet := range record.Packets {
				if packet.Packet.Predicate.Native == "AdditionalInformationOnEligibility" {
					if packet.Packet.Object.NativeStatus == "withheld" {
						withheld++
					}
					if packet.Packet.Object.NativeLexical != "" || packet.Packet.Object.Typed != nil {
						return Manifest{}, errors.New("opportunity release: withheld eligibility lexical value escaped")
					}
				}
				packetEntries = append(packetEntries, artifactsegment.Entry{Digest: packet.Digest, CBOR: packet.CBOR})
			}
			for _, mapping := range record.Mappings {
				mappingEntries = append(mappingEntries, artifactsegment.Entry{Digest: mapping.Digest, CBOR: mapping.CBOR})
			}
			opportunityFrames = append(opportunityFrames, universesnapshot.SourceFrame{Digest: record.FrameDigest, CBOR: record.FrameCBOR, Frame: record.Frame})
		}
		if err := writeSegments(rootOut, "packet_segment", artifactsegment.KindPacket, packetEntries, &segmentSequence, &release.Artifacts); err != nil {
			return Manifest{}, err
		}
		if err := writeSegments(rootOut, "mapping_segment", artifactsegment.KindMapping, mappingEntries, &segmentSequence, &release.Artifacts); err != nil {
			return Manifest{}, err
		}
		release.Packets += uint64(len(packetEntries))
		release.MappingClaims += uint64(len(mappingEntries))
	}
	if total != projectionManifest.Report.RecordsAccepted || uint64(len(opportunityFrames)) != total {
		return Manifest{}, errors.New("opportunity release: compiler count does not reconcile with projection")
	}
	opportunitySegment, opportunitySegmentDigest, err := universesnapshot.BuildCompact(opportunityFrames)
	if err != nil {
		return Manifest{}, err
	}
	if err := writeArtifact(rootOut, "segments/opportunity.twux", opportunitySegment, "opportunity_frame_segment", uint64(len(opportunityFrames)), &release.Artifacts); err != nil {
		return Manifest{}, err
	}
	worldFrames, err := loadWorldFrames(worldReleaseRoot, worldRelease)
	if err != nil {
		return Manifest{}, err
	}
	combinedFrames := make([]universesnapshot.SourceFrame, 0, len(worldFrames)+len(opportunityFrames))
	combinedFrames = append(combinedFrames, worldFrames...)
	combinedFrames = append(combinedFrames, opportunityFrames...)
	combinedSegment, combinedDigest, err := universesnapshot.BuildCompact(combinedFrames)
	if err != nil {
		return Manifest{}, err
	}
	if err := writeArtifact(rootOut, "segments/utility-universes.twux", combinedSegment, "combined_frame_segment", uint64(len(combinedFrames)), &release.Artifacts); err != nil {
		return Manifest{}, err
	}
	privacy := PrivacyReport{
		Format: PrivacyFormat, ProjectionPublic: false, RawEvidencePublic: false,
		ContactFieldsExcluded: projectionManifest.Report.ContactFieldsExcluded, DescriptionFieldsExcluded: projectionManifest.Report.DescriptionFieldsExcluded,
		EligibilityFieldsWithheld: withheld, EmailLikeOccurrencesInProjection: countMatches(emailPattern, projection),
		URLLikeOccurrencesInProjection: countMatches(urlPattern, projection), PhoneLikeOccurrencesInProjection: countMatches(phonePattern, projection),
		EligibilityLexicalValuesInRelease: 0,
		PublisherNonEndorsementNotice:     "This product uses the Grants.gov public data source but is not endorsed or certified by the U.S. Department of Health and Human Services.",
		ProjectionWithholdingReason:       "The approved private projection contains contact-like text inside the allowed eligibility field. The public release emits proof-linked withheld status and no eligibility lexical value.",
	}
	privacyBytes, err := marshal(privacy, maxPrivacy)
	if err != nil {
		return Manifest{}, err
	}
	if err := writeArtifact(rootOut, "reports/privacy.json", privacyBytes, "privacy_report", 0, &release.Artifacts); err != nil {
		return Manifest{}, err
	}
	release.Format = ReleaseFormat
	release.OriginID = universeimport.GrantsGovOriginID
	release.UniverseID = "tw:opportunity"
	release.CompiledAt = acquisition.CompletedAt
	release.EvidenceClass = "current_observation"
	release.WorkOrderDigest = acquisition.WorkOrderDigest
	release.PolicyDecisionDigest = acquisition.PolicyDecisionDigest
	release.AcquisitionManifestDigest = digestText(observationDigest)
	release.ArchiveDigest = acquisition.ArchiveDigest
	release.XMLDigest = projectionManifest.XMLDigest
	release.PrivateProjectionDigest = projectionManifest.ProjectionDigest
	release.ModuleSetDigest = digestText(moduleSet)
	release.WorldStateReleaseDigest = digestText(dataplane.DigestBytes(worldManifestBytes))
	release.SourceRecordsSeen = projectionManifest.Report.SourceRecordsSeen
	release.SourceRecordsAccepted = projectionManifest.Report.RecordsAccepted
	release.SourceRecordsRejected = projectionManifest.Report.RecordsRejected
	release.Frames = uint64(len(opportunityFrames))
	release.WorldStateFrames = uint64(len(worldFrames))
	release.CombinedFrames = uint64(len(combinedFrames))
	release.ArtifactSegments = segmentSequence
	release.TrustLane = "provisional_semantic"
	release.MappingStatus = "candidate"
	release.SchedulerEnabled = false
	release.RawEvidencePublic = false
	release.PrivateProjectionPublic = false
	release.EligibilityTextWithheld = true
	release.RuntimeOriginNetworkCalls = 0
	release.RuntimeBrowserExecutions = 0
	release.RuntimeModelAuthority = "none"
	sort.Slice(release.Artifacts, func(i, j int) bool { return release.Artifacts[i].Path < release.Artifacts[j].Path })
	manifestBytes, err := marshal(release, maxManifest)
	if err != nil {
		return Manifest{}, err
	}
	// Manifest-last publication is the only release completion signal.
	if err := atomicfile.Write(filepath.Join(rootOut, "release-manifest.json"), manifestBytes, maxManifest, 0o440); err != nil {
		return Manifest{}, err
	}
	_ = opportunitySegmentDigest
	_ = combinedDigest
	return release, nil
}

func Verify(root, acquisitionRoot, projectionRoot, worldReleaseRoot, releaseRoot string) (Manifest, error) {
	var manifest Manifest
	loaded, err := opportunitypilot.LoadWorkOrder(filepath.Join(root, "atlas", "e4-plans", "grants-gov-20260811.json"))
	if err != nil || opportunitypilot.VerifyAuthority(root, loaded) != nil {
		return manifest, errors.New("opportunity release: exact founder authority is unavailable")
	}
	acquisition, err := opportunitypilot.VerifyAcquisition(acquisitionRoot, loaded)
	if err != nil {
		return manifest, err
	}
	projectionManifest, _, err := opportunitypilot.VerifyProjection(projectionRoot, acquisitionRoot, loaded)
	if err != nil {
		return manifest, err
	}
	worldRelease, err := worldstatepilot.VerifyRelease(root, worldReleaseRoot)
	if err != nil {
		return manifest, err
	}
	manifestBytes, err := readRegular(filepath.Join(releaseRoot, "release-manifest.json"), maxManifest)
	if err != nil {
		return manifest, err
	}
	policy := jsonbounded.Policy{MaxBytes: maxManifest, MaxDepth: 12, MaxScalarBytes: 64 << 10, MaxContainerEntries: 20000, MaxTokens: 100000}
	if err := jsonbounded.Decode(manifestBytes, &manifest, policy, true); err != nil {
		return Manifest{}, err
	}
	if manifest.Format != ReleaseFormat || manifest.OriginID != universeimport.GrantsGovOriginID || manifest.UniverseID != "tw:opportunity" || manifest.EvidenceClass != "current_observation" || manifest.WorkOrderDigest != acquisition.WorkOrderDigest || manifest.PolicyDecisionDigest != acquisition.PolicyDecisionDigest || manifest.ArchiveDigest != acquisition.ArchiveDigest || manifest.XMLDigest != projectionManifest.XMLDigest || manifest.PrivateProjectionDigest != projectionManifest.ProjectionDigest || manifest.SourceRecordsSeen != projectionManifest.Report.SourceRecordsSeen || manifest.SourceRecordsAccepted != projectionManifest.Report.RecordsAccepted || manifest.SourceRecordsRejected != projectionManifest.Report.RecordsRejected || manifest.Frames != manifest.SourceRecordsAccepted || manifest.CombinedFrames != manifest.Frames+manifest.WorldStateFrames || manifest.SchedulerEnabled || manifest.RawEvidencePublic || manifest.PrivateProjectionPublic || !manifest.EligibilityTextWithheld || manifest.RuntimeOriginNetworkCalls != 0 || manifest.RuntimeBrowserExecutions != 0 || manifest.RuntimeModelAuthority != "none" || manifest.TrustLane != "provisional_semantic" || manifest.MappingStatus != "candidate" || len(manifest.Artifacts) == 0 {
		return Manifest{}, errors.New("opportunity release: manifest identity or authority is invalid")
	}
	if _, err := os.Stat(filepath.Join(releaseRoot, "approved-projection.json")); !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, errors.New("opportunity release: private projection entered public release")
	}
	packetDigests := make(map[dataplane.Digest]struct{}, manifest.Packets)
	mappingDigests := make(map[dataplane.Digest]struct{}, manifest.MappingClaims)
	packetSegments, mappingSegments := uint64(0), uint64(0)
	var opportunityPath, combinedPath string
	var opportunityDigest, combinedDigest dataplane.Digest
	var privacy PrivacyReport
	for index, artifact := range manifest.Artifacts {
		if artifact.Path == "" || filepath.IsAbs(artifact.Path) || filepath.Clean(artifact.Path) != artifact.Path || strings.Contains(artifact.Path, "..") || index > 0 && manifest.Artifacts[index-1].Path >= artifact.Path {
			return Manifest{}, errors.New("opportunity release: unsafe or unsorted artifact path")
		}
		digest, err := parseDigest(artifact.Digest)
		if err != nil {
			return Manifest{}, err
		}
		data, err := readRegular(filepath.Join(releaseRoot, artifact.Path), artifact.Size)
		if err != nil || uint64(len(data)) != artifact.Size || dataplane.DigestBytes(data) != digest {
			return Manifest{}, errors.New("opportunity release: artifact does not rehash")
		}
		switch artifact.Kind {
		case "packet_segment", "mapping_segment":
			segment, err := artifactsegment.Open(data, digest)
			expectedKind := artifactsegment.KindPacket
			if artifact.Kind == "mapping_segment" {
				expectedKind = artifactsegment.KindMapping
			}
			if err != nil || uint64(segment.Count()) != artifact.Entries || segment.Kind() != expectedKind {
				return Manifest{}, errors.New("opportunity release: artifact segment is invalid")
			}
			for entryIndex := uint32(0); entryIndex < segment.Count(); entryIndex++ {
				entry, _ := segment.Entry(entryIndex)
				if artifact.Kind == "packet_segment" {
					packet, err := dataplane.UnmarshalPacket(entry.CBOR)
					if err != nil {
						return Manifest{}, err
					}
					if packet.Predicate.Native == "AdditionalInformationOnEligibility" && (packet.Object.NativeLexical != "" || packet.Object.Typed != nil || packet.Object.NativeStatus != "withheld" && packet.Object.NativeStatus != "not_provided") {
						return Manifest{}, errors.New("opportunity release: eligibility privacy invariant failed")
					}
					packetDigests[entry.Digest] = struct{}{}
				} else {
					if _, err := dataplane.UnmarshalMappingClaim(entry.CBOR); err != nil {
						return Manifest{}, err
					}
					mappingDigests[entry.Digest] = struct{}{}
				}
			}
			if artifact.Kind == "packet_segment" {
				packetSegments++
			} else {
				mappingSegments++
			}
		case "opportunity_frame_segment":
			opportunityPath, opportunityDigest = artifact.Path, digest
		case "combined_frame_segment":
			combinedPath, combinedDigest = artifact.Path, digest
		case "privacy_report":
			if err := jsonbounded.Decode(data, &privacy, jsonbounded.Policy{MaxBytes: maxPrivacy, MaxDepth: 4, MaxScalarBytes: 4096, MaxContainerEntries: 64, MaxTokens: 256}, true); err != nil {
				return Manifest{}, err
			}
		default:
			return Manifest{}, errors.New("opportunity release: unknown artifact kind")
		}
	}
	if uint64(len(packetDigests)) != manifest.Packets || uint64(len(mappingDigests)) != manifest.MappingClaims || packetSegments+mappingSegments != manifest.ArtifactSegments || privacy.Format != PrivacyFormat || privacy.ProjectionPublic || privacy.RawEvidencePublic || privacy.EligibilityLexicalValuesInRelease != 0 || privacy.EligibilityFieldsWithheld == 0 {
		return Manifest{}, errors.New("opportunity release: packet, segment, or privacy counts are invalid")
	}
	opportunityRuntime, err := universesnapshot.OpenCompactFile(filepath.Join(releaseRoot, opportunityPath), opportunityDigest)
	if err != nil {
		return Manifest{}, err
	}
	defer opportunityRuntime.Close()
	if opportunityRuntime.FrameCount() != manifest.Frames {
		return Manifest{}, errors.New("opportunity release: opportunity frame count mismatch")
	}
	used := make(map[dataplane.Digest]struct{}, len(packetDigests))
	opportunityFrameDigests := make(map[dataplane.Digest]struct{}, manifest.Frames)
	if err := opportunityRuntime.VisitFrames(func(frameDigest dataplane.Digest, data []byte) error {
		frame, err := dataplane.UnmarshalFrame(data)
		if err != nil {
			return err
		}
		if frame.UniverseID != "tw:opportunity" || frame.FrameType != "opportunity:GrantOpportunity" {
			return errors.New("opportunity release: opportunity frame classification mismatch")
		}
		for _, packetDigest := range frame.Derivation.PacketDigests {
			if _, exists := packetDigests[packetDigest]; !exists {
				return errors.New("opportunity release: frame references absent packet")
			}
			used[packetDigest] = struct{}{}
		}
		opportunityFrameDigests[frameDigest] = struct{}{}
		return nil
	}); err != nil {
		return Manifest{}, err
	}
	if uint64(len(opportunityFrameDigests)) != manifest.Frames {
		return Manifest{}, errors.New("opportunity release: opportunity frame walk does not cover release")
	}
	if len(used) != len(packetDigests) {
		return Manifest{}, errors.New("opportunity release: public packet is unused by frames")
	}
	combinedRuntime, err := universesnapshot.OpenCompactFile(filepath.Join(releaseRoot, combinedPath), combinedDigest)
	if err != nil {
		return Manifest{}, err
	}
	defer combinedRuntime.Close()
	if combinedRuntime.FrameCount() != manifest.CombinedFrames {
		return Manifest{}, errors.New("opportunity release: combined frame count mismatch")
	}
	combinedOpportunity := uint64(0)
	combinedWorld := uint64(0)
	if err := combinedRuntime.VisitFrames(func(digest dataplane.Digest, data []byte) error {
		frame, err := dataplane.UnmarshalFrame(data)
		if err != nil {
			return err
		}
		switch {
		case frame.UniverseID == "tw:opportunity" && frame.FrameType == "opportunity:GrantOpportunity":
			if _, exists := opportunityFrameDigests[digest]; !exists {
				return errors.New("opportunity release: combined segment contains an unknown Opportunity frame")
			}
			combinedOpportunity++
		case frame.UniverseID == "tw:world-state" && frame.FrameType == "world:IndicatorObservation":
			combinedWorld++
		default:
			return errors.New("opportunity release: combined segment contains an unexpected frame class")
		}
		return nil
	}); err != nil {
		return Manifest{}, err
	}
	if combinedOpportunity != manifest.Frames || combinedWorld != manifest.WorldStateFrames || manifest.WorldStateFrames != uint64(worldRelease.Frames) {
		return Manifest{}, errors.New("opportunity release: World State segment binding failed")
	}
	queryLimit := uint32(manifest.Frames)
	if queryLimit > 1000 {
		queryLimit = 1000
	}
	queryResult, err := combinedRuntime.Query(universesnapshot.Query{UniverseID: "tw:opportunity", FrameType: "opportunity:GrantOpportunity", Limit: queryLimit})
	if err != nil || len(queryResult) != int(queryLimit) {
		return Manifest{}, errors.New("opportunity release: bounded Opportunity query failed")
	}
	return manifest, nil
}

func writeSegments(root, kind string, segmentKind uint32, entries []artifactsegment.Entry, sequence *uint64, artifacts *[]Artifact) error {
	for start := 0; start < len(entries); start += artifactsegment.MaxEntries {
		end := start + artifactsegment.MaxEntries
		if end > len(entries) {
			end = len(entries)
		}
		data, digest, err := artifactsegment.Build(segmentKind, entries[start:end])
		if err != nil {
			return err
		}
		path := fmt.Sprintf("segments/%s-%06d.twas", strings.TrimSuffix(kind, "_segment"), *sequence)
		*sequence++
		if err := atomicfile.Write(filepath.Join(root, path), data, artifactsegment.MaxSegmentBytes, 0o440); err != nil {
			return err
		}
		*artifacts = append(*artifacts, Artifact{Kind: kind, Path: path, Digest: digestText(digest), Size: uint64(len(data)), Entries: uint64(end - start)})
	}
	return nil
}

func writeArtifact(root, path string, data []byte, kind string, entries uint64, artifacts *[]Artifact) error {
	maximum := universesnapshot.MaxBytes
	if len(data) <= maxPrivacy {
		maximum = maxPrivacy
	}
	if err := atomicfile.Write(filepath.Join(root, path), data, maximum, 0o440); err != nil {
		return err
	}
	*artifacts = append(*artifacts, Artifact{Kind: kind, Path: path, Digest: digestText(dataplane.DigestBytes(data)), Size: uint64(len(data)), Entries: entries})
	return nil
}

func loadModuleSet(root string) (dataplane.Digest, error) {
	data, err := os.ReadFile(filepath.Join(root, "generated", "e4", "ontology", "universes", "tw-opportunity-0.1.0.cbor"))
	if err != nil {
		return dataplane.Digest{}, err
	}
	universe, err := dataplane.UnmarshalSemanticUniverse(data)
	if err != nil || universe.UniverseID != "tw:opportunity" {
		return dataplane.Digest{}, errors.New("opportunity release: compiled Opportunity universe is invalid")
	}
	return universe.ModuleSetDigest, nil
}

func loadWorldFrames(root string, release worldstatepilot.ReleaseManifest) ([]universesnapshot.SourceFrame, error) {
	var segment worldstatepilot.ReleaseArtifact
	for _, artifact := range release.Artifacts {
		if artifact.Kind == "compact_frame_segment" {
			segment = artifact
		}
	}
	digest, err := parseDigest(segment.Digest)
	if err != nil {
		return nil, err
	}
	runtime, err := universesnapshot.OpenCompactFile(filepath.Join(root, segment.Path), digest)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	digests, err := runtime.Query(universesnapshot.Query{UniverseID: "tw:world-state", FrameType: "world:IndicatorObservation", Limit: uint32(release.Frames)})
	if err != nil || len(digests) != release.Frames {
		return nil, errors.New("opportunity release: World State compact segment does not reconcile")
	}
	result := make([]universesnapshot.SourceFrame, 0, len(digests))
	for _, digest := range digests {
		data, err := runtime.Trace(digest)
		if err != nil {
			return nil, err
		}
		frame, err := dataplane.UnmarshalFrame(data)
		if err != nil {
			return nil, err
		}
		result = append(result, universesnapshot.SourceFrame{Digest: digest, CBOR: append([]byte(nil), data...), Frame: frame})
	}
	return result, nil
}

func createOutput(path string) (string, error) {
	if path == "" {
		return "", errors.New("opportunity release: output is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(filepath.Dir(absolute)); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("opportunity release: output parent must be a real directory")
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", errors.New("opportunity release: immutable output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Mkdir(absolute, 0o750); err != nil {
		return "", err
	}
	return absolute, nil
}

func readRegular(path string, maximum uint64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || uint64(info.Size()) > maximum {
		return nil, errors.New("opportunity release: bounded regular artifact unavailable")
	}
	return os.ReadFile(path)
}

func marshal(value any, maximum int) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) == 0 || len(data) > maximum {
		return nil, errors.New("opportunity release: JSON artifact exceeds bound")
	}
	return data, nil
}

func parseDigest(value string) (dataplane.Digest, error) {
	var result dataplane.Digest
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return result, errors.New("opportunity release: invalid digest")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != len(result) {
		return result, errors.New("opportunity release: invalid digest")
	}
	copy(result[:], decoded)
	return result, nil
}

func digestText(value dataplane.Digest) string { return "sha256:" + hex.EncodeToString(value[:]) }

func countMatches(pattern *regexp.Regexp, data []byte) uint64 {
	count := uint64(0)
	for len(data) > 0 {
		location := pattern.FindIndex(data)
		if location == nil {
			break
		}
		count++
		data = data[location[1]:]
	}
	return count
}
