package dataplane

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
)

const (
	PacketVersion = "tw.semantic-packet/0.1"
	BatchVersion  = "tw.packet-batch-manifest/0.1"
	DeltaVersion  = "tw.semantic-delta/0.1"
)

type PacketSubject struct {
	Native              string
	CanonicalCandidates []string
}

type PacketPredicate struct {
	Native   string
	Semantic OptionalText
}

type PacketObject struct {
	NativeStatus  string
	NativeLexical string
	MediaType     OptionalText
	Language      OptionalText
	Typed         *TypedValue
}

type ContextDimension struct {
	Key   string
	Value TypedValue
}

type PacketContext struct {
	Dimensions   []ContextDimension
	Jurisdiction OptionalText
	Language     OptionalText
	Scope        OptionalText
}

type PacketTime struct {
	ObservedAt     string
	AssertedAt     OptionalText
	ValidFrom      OptionalText
	ValidUntil     OptionalText
	SourceModified OptionalText
}

type PacketSource struct {
	OriginID             string
	RepresentationDigest Digest
	Locator              string
	NativeSchemaRef      OptionalText
}

type PacketDerivation struct {
	ObservationDigest      Digest
	TransportDigest        OptionalDigest
	AdapterDigest          Digest
	ExtractionPlanDigest   Digest
	TransformationIDs      []string
	MappingIDs             []string
	SemanticClosureDigest  OptionalDigest
	CompilerContractDigest Digest
	CompilerVersion        string
}

type PacketEpistemic struct {
	Lane                 string
	ExtractionStatus     string
	MappingStatus        string
	ConfidenceMillionths *uint64
	AuthorityClass       string
	FreshnessStatus      string
}

type PacketLifecycle struct {
	State            string
	SupersedesDigest OptionalDigest
}

type Packet struct {
	Version    string
	Kind       string
	Subject    PacketSubject
	Predicate  PacketPredicate
	Object     PacketObject
	Context    PacketContext
	Time       PacketTime
	Source     PacketSource
	Derivation PacketDerivation
	Epistemic  PacketEpistemic
	Lifecycle  PacketLifecycle
	Retention  string
	Disclosure string
}

func (p Packet) Validate() error {
	if p.Version != PacketVersion {
		return fmt.Errorf("%w: packet version must be %q", ErrInvalid, PacketVersion)
	}
	if err := validateEnum("packet kind", p.Kind, "claim", "state", "capability", "offer", "relationship", "event", "measurement", "document"); err != nil {
		return err
	}
	if err := validateIdentifier("native subject", p.Subject.Native); err != nil {
		return err
	}
	if err := validateSortedUniqueText("canonical candidates", p.Subject.CanonicalCandidates, 32); err != nil {
		return err
	}
	if err := validateIdentifier("native predicate", p.Predicate.Native); err != nil {
		return err
	}
	if err := validateOptionalIdentifier("semantic predicate", p.Predicate.Semantic); err != nil {
		return err
	}
	if err := validateEnum("native status", p.Object.NativeStatus, "resolved", "unknown", "not_observed", "not_provided", "not_applicable", "withheld", "redacted", "unresolved", "contradictory", "invalid", "confirmed_absent"); err != nil {
		return err
	}
	if p.Object.NativeStatus == "resolved" {
		if err := validateText("native lexical", p.Object.NativeLexical, MaxLexical, true); err != nil {
			return err
		}
	} else if p.Object.NativeLexical != "" || p.Object.Typed != nil {
		return fmt.Errorf("%w: non-resolved object must omit lexical and typed values", ErrInvalid)
	}
	if p.Object.Typed != nil {
		if p.Object.NativeStatus != "resolved" {
			return fmt.Errorf("%w: typed value requires resolved native status", ErrInvalid)
		}
		if err := p.Object.Typed.Validate(); err != nil {
			return err
		}
	}
	if p.Object.MediaType.Present {
		if err := validateText("media type", p.Object.MediaType.Value, MaxShortText, false); err != nil {
			return err
		}
	} else if p.Object.MediaType.Value != "" {
		return fmt.Errorf("%w: absent media type carries a value", ErrInvalid)
	}
	if err := validateLanguage("object language", p.Object.Language); err != nil {
		return err
	}
	if len(p.Context.Dimensions) > 32 {
		return fmt.Errorf("%w: too many context dimensions", ErrInvalid)
	}
	for i, dimension := range p.Context.Dimensions {
		if err := validateIdentifier("context dimension key", dimension.Key); err != nil {
			return err
		}
		if i > 0 && strings.Compare(p.Context.Dimensions[i-1].Key, dimension.Key) >= 0 {
			return fmt.Errorf("%w: context dimensions must be strictly sorted by key", ErrInvalid)
		}
		if err := dimension.Value.Validate(); err != nil {
			return err
		}
	}
	if err := validateOptionalIdentifier("jurisdiction", p.Context.Jurisdiction); err != nil {
		return err
	}
	if err := validateLanguage("context language", p.Context.Language); err != nil {
		return err
	}
	if err := validateOptionalIdentifier("scope", p.Context.Scope); err != nil {
		return err
	}
	if err := validateUTCSecond("observed at", p.Time.ObservedAt); err != nil {
		return err
	}
	times := []struct {
		name  string
		value OptionalText
	}{
		{"asserted at", p.Time.AssertedAt},
		{"valid from", p.Time.ValidFrom},
		{"valid until", p.Time.ValidUntil},
		{"source modified at", p.Time.SourceModified},
	}
	for _, field := range times {
		if err := validateOptionalUTCSecond(field.name, field.value); err != nil {
			return err
		}
	}
	if p.Time.ValidFrom.Present && p.Time.ValidUntil.Present && p.Time.ValidFrom.Value > p.Time.ValidUntil.Value {
		return fmt.Errorf("%w: valid-from is after valid-until", ErrInvalid)
	}
	if err := validateIdentifier("origin ID", p.Source.OriginID); err != nil {
		return err
	}
	if err := validateText("source locator", p.Source.Locator, MaxLocator, false); err != nil {
		return err
	}
	if err := validateOptionalIdentifier("native schema reference", p.Source.NativeSchemaRef); err != nil {
		return err
	}
	if err := validateOptionalDigest("transport digest", p.Derivation.TransportDigest); err != nil {
		return err
	}
	if len(p.Derivation.TransformationIDs) > 32 {
		return fmt.Errorf("%w: too many transformations", ErrInvalid)
	}
	seenTransforms := make(map[string]struct{}, len(p.Derivation.TransformationIDs))
	for _, value := range p.Derivation.TransformationIDs {
		if err := validateIdentifier("transformation ID", value); err != nil {
			return err
		}
		if _, exists := seenTransforms[value]; exists {
			return fmt.Errorf("%w: duplicate transformation ID", ErrInvalid)
		}
		seenTransforms[value] = struct{}{}
	}
	if err := validateSortedUniqueText("mapping IDs", p.Derivation.MappingIDs, 32); err != nil {
		return err
	}
	if err := validateOptionalDigest("semantic closure digest", p.Derivation.SemanticClosureDigest); err != nil {
		return err
	}
	if err := validateIdentifier("compiler version", p.Derivation.CompilerVersion); err != nil {
		return err
	}
	if err := validateEnum("trust lane", p.Epistemic.Lane, "observed_native", "provisional_semantic", "attested_semantic"); err != nil {
		return err
	}
	if err := validateEnum("extraction status", p.Epistemic.ExtractionStatus, "deterministic", "publisher_attested"); err != nil {
		return err
	}
	if err := validateEnum("mapping status", p.Epistemic.MappingStatus, "none", "candidate", "reviewed", "disputed", "revoked"); err != nil {
		return err
	}
	if p.Epistemic.ConfidenceMillionths != nil && *p.Epistemic.ConfidenceMillionths > 1000000 {
		return fmt.Errorf("%w: confidence outside 0..1000000", ErrInvalid)
	}
	if err := validateIdentifier("authority class", p.Epistemic.AuthorityClass); err != nil {
		return err
	}
	if err := validateEnum("freshness status", p.Epistemic.FreshnessStatus, "current", "stale", "unknown"); err != nil {
		return err
	}
	semantic := p.Predicate.Semantic.Present
	hasMappings := len(p.Derivation.MappingIDs) > 0
	hasClosure := p.Derivation.SemanticClosureDigest.Present
	switch p.Epistemic.Lane {
	case "observed_native":
		if semantic || hasMappings || hasClosure || p.Epistemic.MappingStatus != "none" || p.Epistemic.ConfidenceMillionths != nil {
			return fmt.Errorf("%w: observed-native lane cannot carry semantic promotion", ErrInvalid)
		}
	case "provisional_semantic":
		if !semantic || !hasMappings || !hasClosure || p.Epistemic.MappingStatus != "candidate" {
			return fmt.Errorf("%w: provisional lane requires candidate mapping and closure evidence", ErrInvalid)
		}
	case "attested_semantic":
		if !semantic || !hasMappings || !hasClosure || p.Epistemic.MappingStatus != "reviewed" || p.Epistemic.ConfidenceMillionths != nil {
			return fmt.Errorf("%w: attested lane requires reviewed mapping and closure evidence", ErrInvalid)
		}
	}
	if p.Epistemic.ConfidenceMillionths != nil && p.Epistemic.MappingStatus != "candidate" {
		return fmt.Errorf("%w: confidence is allowed only for a candidate mapping", ErrInvalid)
	}
	if err := validateEnum("lifecycle state", p.Lifecycle.State, "current", "superseded", "withdrawn", "stale", "retracted", "invalid"); err != nil {
		return err
	}
	if err := validateOptionalDigest("supersedes digest", p.Lifecycle.SupersedesDigest); err != nil {
		return err
	}
	if p.Lifecycle.State == "superseded" && !p.Lifecycle.SupersedesDigest.Present {
		return fmt.Errorf("%w: superseded packet requires prior digest", ErrInvalid)
	}
	if p.Lifecycle.State == "current" && p.Lifecycle.SupersedesDigest.Present {
		return fmt.Errorf("%w: current packet cannot supersede another packet", ErrInvalid)
	}
	if err := validateEnum("retention", p.Retention, "public_transient", "public_versioned", "public_archival"); err != nil {
		return err
	}
	if p.Disclosure != "public" {
		return fmt.Errorf("%w: Genesis disclosure must be public", ErrInvalid)
	}
	return nil
}

func validateLanguage(name string, value OptionalText) error {
	if !value.Present {
		if value.Value != "" {
			return fmt.Errorf("%w: absent %s carries a value", ErrInvalid, name)
		}
		return nil
	}
	if len(value.Value) < 2 || len(value.Value) > 63 {
		return fmt.Errorf("%w: %s length outside 2..63", ErrInvalid, name)
	}
	for _, r := range value.Value {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return fmt.Errorf("%w: %s is not a bounded language tag", ErrInvalid, name)
		}
	}
	return nil
}

func MarshalPacket(p Packet) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(14)
	enc.Text(p.Version)
	enc.Text(p.Kind)
	enc.Array(2)
	enc.Text(p.Subject.Native)
	encodeTextSet(&enc, p.Subject.CanonicalCandidates)
	enc.Array(2)
	enc.Text(p.Predicate.Native)
	encodeOptionalText(&enc, p.Predicate.Semantic)
	enc.Array(5)
	enc.Text(p.Object.NativeStatus)
	enc.Text(p.Object.NativeLexical)
	encodeOptionalText(&enc, p.Object.MediaType)
	encodeOptionalText(&enc, p.Object.Language)
	if p.Object.Typed == nil {
		enc.Nil()
	} else {
		encodeTypedValue(&enc, *p.Object.Typed)
	}
	enc.Array(4)
	enc.Array(uint64(len(p.Context.Dimensions)))
	for _, dimension := range p.Context.Dimensions {
		enc.Array(2)
		enc.Text(dimension.Key)
		encodeTypedValue(&enc, dimension.Value)
	}
	encodeOptionalText(&enc, p.Context.Jurisdiction)
	encodeOptionalText(&enc, p.Context.Language)
	encodeOptionalText(&enc, p.Context.Scope)
	enc.Array(5)
	enc.Text(p.Time.ObservedAt)
	encodeOptionalText(&enc, p.Time.AssertedAt)
	encodeOptionalText(&enc, p.Time.ValidFrom)
	encodeOptionalText(&enc, p.Time.ValidUntil)
	encodeOptionalText(&enc, p.Time.SourceModified)
	enc.Array(4)
	enc.Text(p.Source.OriginID)
	encodeDigest(&enc, p.Source.RepresentationDigest)
	enc.Text(p.Source.Locator)
	encodeOptionalText(&enc, p.Source.NativeSchemaRef)
	enc.Array(9)
	encodeDigest(&enc, p.Derivation.ObservationDigest)
	encodeOptionalDigest(&enc, p.Derivation.TransportDigest)
	encodeDigest(&enc, p.Derivation.AdapterDigest)
	encodeDigest(&enc, p.Derivation.ExtractionPlanDigest)
	enc.Array(uint64(len(p.Derivation.TransformationIDs)))
	for _, value := range p.Derivation.TransformationIDs {
		enc.Text(value)
	}
	encodeTextSet(&enc, p.Derivation.MappingIDs)
	encodeOptionalDigest(&enc, p.Derivation.SemanticClosureDigest)
	encodeDigest(&enc, p.Derivation.CompilerContractDigest)
	enc.Text(p.Derivation.CompilerVersion)
	enc.Array(6)
	enc.Text(p.Epistemic.Lane)
	enc.Text(p.Epistemic.ExtractionStatus)
	enc.Text(p.Epistemic.MappingStatus)
	if p.Epistemic.ConfidenceMillionths == nil {
		enc.Nil()
	} else {
		enc.Uint(*p.Epistemic.ConfidenceMillionths)
	}
	enc.Text(p.Epistemic.AuthorityClass)
	enc.Text(p.Epistemic.FreshnessStatus)
	enc.Array(2)
	enc.Text(p.Lifecycle.State)
	encodeOptionalDigest(&enc, p.Lifecycle.SupersedesDigest)
	enc.Text(p.Retention)
	enc.Text(p.Disclosure)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalPacket(data []byte) (Packet, error) {
	var p Packet
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 14 {
		return p, fmt.Errorf("%w: packet array", ErrInvalid)
	}
	if p.Version, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Kind, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
		return p, fmt.Errorf("%w: subject array", ErrInvalid)
	}
	if p.Subject.Native, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Subject.CanonicalCandidates, err = decodeTextSet(dec, 32, 0); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
		return p, fmt.Errorf("%w: predicate array", ErrInvalid)
	}
	if p.Predicate.Native, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Predicate.Semantic, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 5 {
		return p, fmt.Errorf("%w: object array", ErrInvalid)
	}
	if p.Object.NativeStatus, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Object.NativeLexical, err = dec.Text(MaxLexical); err != nil {
		return p, err
	}
	if p.Object.MediaType, err = decodeOptionalText(dec, MaxShortText); err != nil {
		return p, err
	}
	if p.Object.Language, err = decodeOptionalText(dec, 63); err != nil {
		return p, err
	}
	if nilValue, nilErr := dec.TryNil(); nilErr != nil {
		return p, nilErr
	} else if !nilValue {
		value, valueErr := decodeTypedValue(dec)
		if valueErr != nil {
			return p, valueErr
		}
		p.Object.Typed = &value
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 4 {
		return p, fmt.Errorf("%w: context array", ErrInvalid)
	}
	dimensionCount, countErr := dec.Array()
	if countErr != nil || dimensionCount > 32 {
		return p, fmt.Errorf("%w: context dimension count", ErrInvalid)
	}
	for i := uint64(0); i < dimensionCount; i++ {
		if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
			return p, fmt.Errorf("%w: context dimension array", ErrInvalid)
		}
		var dimension ContextDimension
		if dimension.Key, err = dec.Text(MaxIdentifier); err != nil {
			return p, err
		}
		if dimension.Value, err = decodeTypedValue(dec); err != nil {
			return p, err
		}
		p.Context.Dimensions = append(p.Context.Dimensions, dimension)
	}
	if p.Context.Jurisdiction, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return p, err
	}
	if p.Context.Language, err = decodeOptionalText(dec, 63); err != nil {
		return p, err
	}
	if p.Context.Scope, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 5 {
		return p, fmt.Errorf("%w: time array", ErrInvalid)
	}
	if p.Time.ObservedAt, err = dec.Text(20); err != nil {
		return p, err
	}
	if p.Time.AssertedAt, err = decodeOptionalText(dec, 20); err != nil {
		return p, err
	}
	if p.Time.ValidFrom, err = decodeOptionalText(dec, 20); err != nil {
		return p, err
	}
	if p.Time.ValidUntil, err = decodeOptionalText(dec, 20); err != nil {
		return p, err
	}
	if p.Time.SourceModified, err = decodeOptionalText(dec, 20); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 4 {
		return p, fmt.Errorf("%w: source array", ErrInvalid)
	}
	if p.Source.OriginID, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Source.RepresentationDigest, err = decodeDigest(dec); err != nil {
		return p, err
	}
	if p.Source.Locator, err = dec.Text(MaxLocator); err != nil {
		return p, err
	}
	if p.Source.NativeSchemaRef, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 9 {
		return p, fmt.Errorf("%w: derivation array", ErrInvalid)
	}
	if p.Derivation.ObservationDigest, err = decodeDigest(dec); err != nil {
		return p, err
	}
	if p.Derivation.TransportDigest, err = decodeOptionalDigest(dec); err != nil {
		return p, err
	}
	if p.Derivation.AdapterDigest, err = decodeDigest(dec); err != nil {
		return p, err
	}
	if p.Derivation.ExtractionPlanDigest, err = decodeDigest(dec); err != nil {
		return p, err
	}
	transformCount, countErr := dec.Array()
	if countErr != nil || transformCount > 32 {
		return p, fmt.Errorf("%w: transformation count", ErrInvalid)
	}
	for i := uint64(0); i < transformCount; i++ {
		value, textErr := dec.Text(MaxIdentifier)
		if textErr != nil {
			return p, textErr
		}
		p.Derivation.TransformationIDs = append(p.Derivation.TransformationIDs, value)
	}
	if p.Derivation.MappingIDs, err = decodeTextSet(dec, 32, 0); err != nil {
		return p, err
	}
	if p.Derivation.SemanticClosureDigest, err = decodeOptionalDigest(dec); err != nil {
		return p, err
	}
	if p.Derivation.CompilerContractDigest, err = decodeDigest(dec); err != nil {
		return p, err
	}
	if p.Derivation.CompilerVersion, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 6 {
		return p, fmt.Errorf("%w: epistemic array", ErrInvalid)
	}
	if p.Epistemic.Lane, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Epistemic.ExtractionStatus, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Epistemic.MappingStatus, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if nilValue, nilErr := dec.TryNil(); nilErr != nil {
		return p, nilErr
	} else if !nilValue {
		value, valueErr := dec.Uint()
		if valueErr != nil {
			return p, valueErr
		}
		p.Epistemic.ConfidenceMillionths = &value
	}
	if p.Epistemic.AuthorityClass, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Epistemic.FreshnessStatus, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
		return p, fmt.Errorf("%w: lifecycle array", ErrInvalid)
	}
	if p.Lifecycle.State, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Lifecycle.SupersedesDigest, err = decodeOptionalDigest(dec); err != nil {
		return p, err
	}
	if p.Retention, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if p.Disclosure, err = dec.Text(MaxIdentifier); err != nil {
		return p, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return p, err
	}
	if err = finish(dec); err != nil {
		return p, err
	}
	return p, p.Validate()
}

type DigestSizeEntry struct {
	Digest Digest
	Size   uint64
}

type NamedArtifact struct {
	Name   string
	Digest Digest
	Size   uint64
}

type BatchManifest struct {
	Version                string
	OriginID               string
	CompilerContractDigest Digest
	PolicyDecisionDigest   Digest
	StartedAt              string
	CompletedAt            string
	PreviousBatchID        OptionalDigest
	Observations           []Digest
	Packets                []DigestSizeEntry
	Deltas                 []DigestSizeEntry
	RejectionReportDigest  Digest
	MetricsDigest          Digest
	Artifacts              []NamedArtifact
}

func (m BatchManifest) Validate() error {
	if m.Version != BatchVersion {
		return fmt.Errorf("%w: batch version", ErrInvalid)
	}
	if err := validateIdentifier("origin ID", m.OriginID); err != nil {
		return err
	}
	if err := validateUTCSecond("started at", m.StartedAt); err != nil {
		return err
	}
	if err := validateUTCSecond("completed at", m.CompletedAt); err != nil {
		return err
	}
	if m.StartedAt > m.CompletedAt {
		return fmt.Errorf("%w: batch completion precedes start", ErrInvalid)
	}
	if err := validateOptionalDigest("previous batch ID", m.PreviousBatchID); err != nil {
		return err
	}
	if err := validateSortedUniqueDigests("observations", m.Observations, 1024, 1); err != nil {
		return err
	}
	if err := validateDigestEntries("packets", m.Packets, 32768); err != nil {
		return err
	}
	if err := validateDigestEntries("deltas", m.Deltas, 32768); err != nil {
		return err
	}
	if len(m.Artifacts) == 0 || len(m.Artifacts) > 64 {
		return fmt.Errorf("%w: artifact count outside 1..64", ErrInvalid)
	}
	for i, artifact := range m.Artifacts {
		if err := validateText("artifact name", artifact.Name, MaxShortText, false); err != nil {
			return err
		}
		if artifact.Size == 0 || artifact.Size > 4<<20 {
			return fmt.Errorf("%w: artifact size outside bounds", ErrInvalid)
		}
		if i > 0 && strings.Compare(m.Artifacts[i-1].Name, artifact.Name) >= 0 {
			return fmt.Errorf("%w: artifacts must be strictly sorted", ErrInvalid)
		}
	}
	return nil
}

func validateDigestEntries(name string, values []DigestSizeEntry, maximum int) error {
	if len(values) > maximum {
		return fmt.Errorf("%w: %s count exceeds %d", ErrInvalid, name, maximum)
	}
	for i, value := range values {
		if value.Size == 0 || value.Size > 4<<20 {
			return fmt.Errorf("%w: %s size outside bounds", ErrInvalid, name)
		}
		if i > 0 && bytes.Compare(values[i-1].Digest[:], value.Digest[:]) >= 0 {
			return fmt.Errorf("%w: %s must be strictly sorted", ErrInvalid, name)
		}
	}
	return nil
}

func encodeDigestEntries(enc *cborlite.Encoder, values []DigestSizeEntry) {
	enc.Array(uint64(len(values)))
	for _, value := range values {
		enc.Array(2)
		encodeDigest(enc, value.Digest)
		enc.Uint(value.Size)
	}
}

func decodeDigestEntries(dec *cborlite.Decoder, maximum int) ([]DigestSizeEntry, error) {
	n, err := dec.Array()
	if err != nil || n > uint64(maximum) {
		return nil, fmt.Errorf("%w: digest entries count", ErrInvalid)
	}
	values := make([]DigestSizeEntry, 0, n)
	for i := uint64(0); i < n; i++ {
		if length, arrayErr := dec.Array(); arrayErr != nil || length != 2 {
			return nil, fmt.Errorf("%w: digest entry array", ErrInvalid)
		}
		var value DigestSizeEntry
		if value.Digest, err = decodeDigest(dec); err != nil {
			return nil, err
		}
		if value.Size, err = dec.Uint(); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func MarshalBatchManifest(m BatchManifest) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(14)
	enc.Text(m.Version)
	enc.Text(m.OriginID)
	encodeDigest(&enc, m.CompilerContractDigest)
	encodeDigest(&enc, m.PolicyDecisionDigest)
	enc.Text(m.StartedAt)
	enc.Text(m.CompletedAt)
	encodeOptionalDigest(&enc, m.PreviousBatchID)
	encodeDigestSet(&enc, m.Observations)
	encodeDigestEntries(&enc, m.Packets)
	encodeDigestEntries(&enc, m.Deltas)
	encodeDigest(&enc, m.RejectionReportDigest)
	encodeDigest(&enc, m.MetricsDigest)
	enc.Array(uint64(len(m.Artifacts)))
	for _, artifact := range m.Artifacts {
		enc.Array(3)
		enc.Text(artifact.Name)
		encodeDigest(&enc, artifact.Digest)
		enc.Uint(artifact.Size)
	}
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalBatchManifest(data []byte) (BatchManifest, error) {
	var m BatchManifest
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return m, err
	}
	if n, e := dec.Array(); e != nil || n != 14 {
		return m, fmt.Errorf("%w: batch array", ErrInvalid)
	}
	if m.Version, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.OriginID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.CompilerContractDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.PolicyDecisionDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.StartedAt, err = dec.Text(20); err != nil {
		return m, err
	}
	if m.CompletedAt, err = dec.Text(20); err != nil {
		return m, err
	}
	if m.PreviousBatchID, err = decodeOptionalDigest(dec); err != nil {
		return m, err
	}
	if m.Observations, err = decodeDigestSet(dec, 1024, 1); err != nil {
		return m, err
	}
	if m.Packets, err = decodeDigestEntries(dec, 32768); err != nil {
		return m, err
	}
	if m.Deltas, err = decodeDigestEntries(dec, 32768); err != nil {
		return m, err
	}
	if m.RejectionReportDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.MetricsDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	artifactCount, countErr := dec.Array()
	if countErr != nil || artifactCount == 0 || artifactCount > 64 {
		return m, fmt.Errorf("%w: artifact count", ErrInvalid)
	}
	for i := uint64(0); i < artifactCount; i++ {
		if n, e := dec.Array(); e != nil || n != 3 {
			return m, fmt.Errorf("%w: artifact array", ErrInvalid)
		}
		var artifact NamedArtifact
		if artifact.Name, err = dec.Text(MaxShortText); err != nil {
			return m, err
		}
		if artifact.Digest, err = decodeDigest(dec); err != nil {
			return m, err
		}
		if artifact.Size, err = dec.Uint(); err != nil {
			return m, err
		}
		m.Artifacts = append(m.Artifacts, artifact)
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return m, err
	}
	if err = finish(dec); err != nil {
		return m, err
	}
	return m, m.Validate()
}

type Delta struct {
	Version                    string
	Class                      string
	Kind                       string
	SemanticKeyDigest          Digest
	BeforePacketDigest         OptionalDigest
	AfterPacketDigest          OptionalDigest
	BeforeSourceEvidenceDigest OptionalDigest
	AfterSourceEvidenceDigest  OptionalDigest
	OriginID                   string
	OccurredAt                 string
	BatchID                    Digest
	CanonVersion               string
	ReasonCode                 string
}

func (d Delta) Validate() error {
	if d.Version != DeltaVersion {
		return fmt.Errorf("%w: delta version", ErrInvalid)
	}
	if err := validateEnum("delta class", d.Class, "origin", "semantic", "canon"); err != nil {
		return err
	}
	if err := validateIdentifier("origin ID", d.OriginID); err != nil {
		return err
	}
	if err := validateUTCSecond("occurred at", d.OccurredAt); err != nil {
		return err
	}
	if err := validateIdentifier("canon version", d.CanonVersion); err != nil {
		return err
	}
	if err := validateIdentifier("reason code", d.ReasonCode); err != nil {
		return err
	}
	digests := []struct {
		name  string
		value OptionalDigest
	}{
		{"before packet", d.BeforePacketDigest},
		{"after packet", d.AfterPacketDigest},
		{"before evidence", d.BeforeSourceEvidenceDigest},
		{"after evidence", d.AfterSourceEvidenceDigest},
	}
	for _, field := range digests {
		if err := validateOptionalDigest(field.name, field.value); err != nil {
			return err
		}
	}
	originKinds := []string{"added", "modified", "withdrawn", "restored", "source_retracted"}
	semanticKinds := []string{"mapped", "remapped", "narrowed", "broadened", "disputed", "attested", "de_attested"}
	canonKinds := []string{"module_added", "module_superseded", "mapping_superseded", "closure_changed"}
	allowed := originKinds
	if d.Class == "semantic" {
		allowed = semanticKinds
	} else if d.Class == "canon" {
		allowed = canonKinds
	}
	if err := validateEnum("delta kind", d.Kind, allowed...); err != nil {
		return err
	}
	if d.Class == "origin" {
		switch d.Kind {
		case "added":
			if d.BeforePacketDigest.Present || d.BeforeSourceEvidenceDigest.Present || !d.AfterPacketDigest.Present || !d.AfterSourceEvidenceDigest.Present {
				return fmt.Errorf("%w: invalid added topology", ErrInvalid)
			}
		case "modified", "restored":
			if !allPresent(d.BeforePacketDigest, d.AfterPacketDigest, d.BeforeSourceEvidenceDigest, d.AfterSourceEvidenceDigest) || d.BeforeSourceEvidenceDigest.Value == d.AfterSourceEvidenceDigest.Value {
				return fmt.Errorf("%w: origin change requires changed source evidence", ErrInvalid)
			}
		case "withdrawn", "source_retracted":
			if !d.BeforePacketDigest.Present || d.AfterPacketDigest.Present || !d.BeforeSourceEvidenceDigest.Present {
				return fmt.Errorf("%w: invalid removal topology", ErrInvalid)
			}
		}
	} else {
		if !allPresent(d.BeforePacketDigest, d.AfterPacketDigest, d.BeforeSourceEvidenceDigest, d.AfterSourceEvidenceDigest) || d.BeforeSourceEvidenceDigest.Value != d.AfterSourceEvidenceDigest.Value {
			return fmt.Errorf("%w: semantic/canon change must preserve source evidence", ErrInvalid)
		}
		if d.BeforePacketDigest.Value == d.AfterPacketDigest.Value {
			return fmt.Errorf("%w: reinterpretation requires distinct packet identities", ErrInvalid)
		}
	}
	return nil
}

func allPresent(values ...OptionalDigest) bool {
	for _, value := range values {
		if !value.Present {
			return false
		}
	}
	return true
}

func MarshalDelta(d Delta) ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(14)
	enc.Text(d.Version)
	enc.Text(d.Class)
	enc.Text(d.Kind)
	encodeDigest(&enc, d.SemanticKeyDigest)
	encodeOptionalDigest(&enc, d.BeforePacketDigest)
	encodeOptionalDigest(&enc, d.AfterPacketDigest)
	encodeOptionalDigest(&enc, d.BeforeSourceEvidenceDigest)
	encodeOptionalDigest(&enc, d.AfterSourceEvidenceDigest)
	enc.Text(d.OriginID)
	enc.Text(d.OccurredAt)
	encodeDigest(&enc, d.BatchID)
	enc.Text(d.CanonVersion)
	enc.Text(d.ReasonCode)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalDelta(data []byte) (Delta, error) {
	var d Delta
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return d, err
	}
	if n, e := dec.Array(); e != nil || n != 14 {
		return d, fmt.Errorf("%w: delta array", ErrInvalid)
	}
	if d.Version, err = dec.Text(MaxIdentifier); err != nil {
		return d, err
	}
	if d.Class, err = dec.Text(MaxIdentifier); err != nil {
		return d, err
	}
	if d.Kind, err = dec.Text(MaxIdentifier); err != nil {
		return d, err
	}
	if d.SemanticKeyDigest, err = decodeDigest(dec); err != nil {
		return d, err
	}
	if d.BeforePacketDigest, err = decodeOptionalDigest(dec); err != nil {
		return d, err
	}
	if d.AfterPacketDigest, err = decodeOptionalDigest(dec); err != nil {
		return d, err
	}
	if d.BeforeSourceEvidenceDigest, err = decodeOptionalDigest(dec); err != nil {
		return d, err
	}
	if d.AfterSourceEvidenceDigest, err = decodeOptionalDigest(dec); err != nil {
		return d, err
	}
	if d.OriginID, err = dec.Text(MaxIdentifier); err != nil {
		return d, err
	}
	if d.OccurredAt, err = dec.Text(20); err != nil {
		return d, err
	}
	if d.BatchID, err = decodeDigest(dec); err != nil {
		return d, err
	}
	if d.CanonVersion, err = dec.Text(MaxIdentifier); err != nil {
		return d, err
	}
	if d.ReasonCode, err = dec.Text(MaxIdentifier); err != nil {
		return d, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return d, err
	}
	if err = finish(dec); err != nil {
		return d, err
	}
	return d, d.Validate()
}
