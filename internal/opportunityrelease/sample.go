package opportunityrelease

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/artifactsegment"
	"github.com/typed-web-commons/typed-web/internal/atomicfile"
	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

const VerifierSampleFormat = "tw.e4-opportunity-c-verifier-sample/0.1"

type VerifierSampleEntry struct {
	Kind                string `json:"kind"`
	Path                string `json:"path"`
	Digest              string `json:"digest"`
	SourceSegmentDigest string `json:"source_segment_digest"`
	SourceEntryIndex    uint32 `json:"source_entry_index"`
}

type VerifierSampleManifest struct {
	Format                string                `json:"format"`
	ReleaseManifestDigest string                `json:"release_manifest_digest"`
	SelectionPolicy       string                `json:"selection_policy"`
	PacketSamples         uint64                `json:"packet_samples"`
	MappingSamples        uint64                `json:"mapping_samples"`
	FrameSamples          uint64                `json:"frame_samples"`
	Entries               []VerifierSampleEntry `json:"entries"`
}

// ExportVerifierSample publishes a deterministic, privacy-safe sample for the
// independent restricted-C artifact verifier. Go release admission still
// verifies every artifact; this sample makes no full-C-corpus claim.
func ExportVerifierSample(releaseRoot, expectedManifestDigest, output string) (VerifierSampleManifest, error) {
	var sample VerifierSampleManifest
	runtime, manifest, err := OpenPublicRuntime(releaseRoot, expectedManifestDigest)
	if err != nil {
		return sample, err
	}
	defer runtime.Close()
	root, err := createOutput(output)
	if err != nil {
		return sample, err
	}
	sample.Format = VerifierSampleFormat
	sample.ReleaseManifestDigest = expectedManifestDigest
	sample.SelectionPolicy = "first, middle, and last canonical entry from each packet and mapping segment; first three canonical-digest frames from each admitted universe"
	sequence := uint64(0)
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind != "packet_segment" && artifact.Kind != "mapping_segment" {
			continue
		}
		digest, err := parseDigest(artifact.Digest)
		if err != nil {
			return sample, err
		}
		data, err := readRegular(filepath.Join(releaseRoot, artifact.Path), artifact.Size)
		if err != nil || uint64(len(data)) != artifact.Size || dataplane.DigestBytes(data) != digest {
			return sample, errors.New("opportunity release: verifier sample source segment is invalid")
		}
		segment, err := artifactsegment.Open(data, digest)
		if err != nil {
			return sample, err
		}
		kind := "packet"
		expectedKind := artifactsegment.KindPacket
		if artifact.Kind == "mapping_segment" {
			kind = "mapping-claim"
			expectedKind = artifactsegment.KindMapping
		}
		if segment.Kind() != expectedKind {
			return sample, errors.New("opportunity release: verifier sample segment kind mismatch")
		}
		for _, index := range sampleIndexes(segment.Count()) {
			entry, err := segment.Entry(index)
			if err != nil {
				return sample, err
			}
			if kind == "packet" {
				if _, err := dataplane.UnmarshalPacket(entry.CBOR); err != nil {
					return sample, err
				}
				sample.PacketSamples++
			} else {
				if _, err := dataplane.UnmarshalMappingClaim(entry.CBOR); err != nil {
					return sample, err
				}
				sample.MappingSamples++
			}
			path, err := writeVerifierArtifact(root, kind, sequence, entry.Digest, entry.CBOR)
			if err != nil {
				return sample, err
			}
			sequence++
			sample.Entries = append(sample.Entries, VerifierSampleEntry{Kind: kind, Path: path, Digest: digestText(entry.Digest), SourceSegmentDigest: artifact.Digest, SourceEntryIndex: index})
		}
	}
	frameCounts := map[string]uint64{"tw:opportunity": 0, "tw:world-state": 0}
	frameIndex := uint32(0)
	if err := runtime.VisitFrames(func(digest dataplane.Digest, data []byte) error {
		index := frameIndex
		frameIndex++
		frame, err := dataplane.UnmarshalFrame(data)
		if err != nil {
			return err
		}
		count, expected := frameCounts[frame.UniverseID]
		if !expected {
			return errors.New("opportunity release: verifier sample found unexpected universe")
		}
		if count >= 3 {
			return nil
		}
		path, err := writeVerifierArtifact(root, "frame", sequence, digest, data)
		if err != nil {
			return err
		}
		sequence++
		frameCounts[frame.UniverseID]++
		sample.FrameSamples++
		sample.Entries = append(sample.Entries, VerifierSampleEntry{Kind: "frame", Path: path, Digest: digestText(digest), SourceSegmentDigest: combinedArtifactDigest(manifest), SourceEntryIndex: index})
		return nil
	}); err != nil {
		return sample, err
	}
	if frameCounts["tw:opportunity"] != 3 || frameCounts["tw:world-state"] != 3 || sample.PacketSamples == 0 || sample.MappingSamples == 0 {
		return sample, errors.New("opportunity release: verifier sample coverage is incomplete")
	}
	sort.Slice(sample.Entries, func(i, j int) bool { return sample.Entries[i].Path < sample.Entries[j].Path })
	encoded, err := marshal(sample, maxManifest)
	if err != nil {
		return sample, err
	}
	if err := atomicfile.Write(filepath.Join(root, "sample-manifest.json"), encoded, maxManifest, 0o440); err != nil {
		return sample, err
	}
	return sample, nil
}

func sampleIndexes(count uint32) []uint32 {
	candidates := []uint32{0, count / 2, count - 1}
	result := make([]uint32, 0, len(candidates))
	for _, candidate := range candidates {
		if len(result) == 0 || result[len(result)-1] != candidate {
			result = append(result, candidate)
		}
	}
	return result
}

func writeVerifierArtifact(root, kind string, sequence uint64, digest dataplane.Digest, data []byte) (string, error) {
	path := fmt.Sprintf("artifacts/%s-%06d-%s.cbor", kind, sequence, strings.TrimPrefix(digestText(digest), "sha256:")[:16])
	if err := atomicfile.Write(filepath.Join(root, path), data, artifactsegment.MaxArtifactBytes, 0o440); err != nil {
		return "", err
	}
	return path, nil
}

func combinedArtifactDigest(manifest Manifest) string {
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind == "combined_frame_segment" {
			return artifact.Digest
		}
	}
	return ""
}
