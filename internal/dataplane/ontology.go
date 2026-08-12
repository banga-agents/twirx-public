package dataplane

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/cborlite"
)

const (
	FrameVersion    = "tw.semantic-frame/0.1"
	MappingVersion  = "tw.mapping-claim/0.1"
	ModuleVersion   = "tw.ontology-module/0.1"
	UniverseVersion = "tw.semantic-universe/0.1"
)

var semanticVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type FrameSlot struct {
	RoleID        string
	Status        string
	Cardinality   string
	Values        []TypedValue
	PacketDigests []Digest
	MappingIDs    []string
	Conflict      string
}

type FrameTime struct {
	ComposedAt string
	ValidFrom  OptionalText
	ValidUntil OptionalText
}

type FrameEpistemic struct {
	Lane                   string
	CompletenessMillionths uint64
	ConflictStatus         string
}

type FrameDerivation struct {
	PacketDigests          []Digest
	ModuleSetDigest        Digest
	MappingIDs             []string
	CompilerContractDigest Digest
	CompilerVersion        string
}

type FrameLifecycle struct {
	State            string
	SupersedesDigest OptionalDigest
}

type Frame struct {
	Version             string
	UniverseID          string
	FrameType           string
	NativeKey           string
	CanonicalCandidates []string
	Slots               []FrameSlot
	Context             PacketContext
	Time                FrameTime
	Epistemic           FrameEpistemic
	Derivation          FrameDerivation
	Lifecycle           FrameLifecycle
}

func (f Frame) Validate() error {
	if f.Version != FrameVersion {
		return fmt.Errorf("%w: frame version must be %q", ErrInvalid, FrameVersion)
	}
	for name, value := range map[string]string{
		"universe ID": f.UniverseID,
		"frame type":  f.FrameType,
		"native key":  f.NativeKey,
	} {
		if err := validateIdentifier(name, value); err != nil {
			return err
		}
	}
	if err := validateSortedUniqueText("canonical candidates", f.CanonicalCandidates, 32); err != nil {
		return err
	}
	if len(f.Slots) < 1 || len(f.Slots) > 256 {
		return fmt.Errorf("%w: frame slot count outside 1..256", ErrInvalid)
	}
	if err := validateFrameContext(f.Context); err != nil {
		return err
	}
	if err := validateUTCSecond("frame composed at", f.Time.ComposedAt); err != nil {
		return err
	}
	if err := validateOptionalUTCSecond("frame valid from", f.Time.ValidFrom); err != nil {
		return err
	}
	if err := validateOptionalUTCSecond("frame valid until", f.Time.ValidUntil); err != nil {
		return err
	}
	if f.Time.ValidFrom.Present && f.Time.ValidUntil.Present && f.Time.ValidFrom.Value > f.Time.ValidUntil.Value {
		return fmt.Errorf("%w: frame valid-from is after valid-until", ErrInvalid)
	}
	if err := validateEnum("frame trust lane", f.Epistemic.Lane, "observed_native", "provisional_semantic", "attested_semantic"); err != nil {
		return err
	}
	if f.Epistemic.CompletenessMillionths > 1000000 {
		return fmt.Errorf("%w: frame completeness outside 0..1000000", ErrInvalid)
	}
	if err := validateEnum("frame conflict status", f.Epistemic.ConflictStatus, "none", "preserved", "unresolved"); err != nil {
		return err
	}
	if err := validateSortedUniqueDigests("frame packet digests", f.Derivation.PacketDigests, 4096, 1); err != nil {
		return err
	}
	if err := validateSortedUniqueText("frame mapping IDs", f.Derivation.MappingIDs, 256); err != nil {
		return err
	}
	if err := validateIdentifier("frame compiler version", f.Derivation.CompilerVersion); err != nil {
		return err
	}
	if f.Epistemic.Lane == "observed_native" && len(f.Derivation.MappingIDs) != 0 {
		return fmt.Errorf("%w: observed-native frame cannot carry mapping IDs", ErrInvalid)
	}
	if f.Epistemic.Lane != "observed_native" && len(f.Derivation.MappingIDs) == 0 {
		return fmt.Errorf("%w: semantic frame requires mapping IDs", ErrInvalid)
	}

	packetUse := make(map[Digest]struct{}, len(f.Derivation.PacketDigests))
	allowedPackets := make(map[Digest]struct{}, len(f.Derivation.PacketDigests))
	for _, digest := range f.Derivation.PacketDigests {
		allowedPackets[digest] = struct{}{}
	}
	mappingUse := make(map[string]struct{}, len(f.Derivation.MappingIDs))
	for _, id := range f.Derivation.MappingIDs {
		mappingUse[id] = struct{}{}
	}
	for i, slot := range f.Slots {
		if err := slot.Validate(); err != nil {
			return fmt.Errorf("frame slot %d: %w", i, err)
		}
		if i > 0 && strings.Compare(f.Slots[i-1].RoleID, slot.RoleID) >= 0 {
			return fmt.Errorf("%w: frame slots must be strictly sorted by role ID", ErrInvalid)
		}
		for _, digest := range slot.PacketDigests {
			if _, ok := allowedPackets[digest]; !ok {
				return fmt.Errorf("%w: frame slot references packet outside derivation", ErrInvalid)
			}
			packetUse[digest] = struct{}{}
		}
		for _, id := range slot.MappingIDs {
			if _, ok := mappingUse[id]; !ok {
				return fmt.Errorf("%w: frame slot references mapping outside derivation", ErrInvalid)
			}
		}
	}
	if len(packetUse) != len(allowedPackets) {
		return fmt.Errorf("%w: frame derivation contains packet unused by slots", ErrInvalid)
	}
	if err := validateEnum("frame lifecycle", f.Lifecycle.State, "current", "superseded", "withdrawn", "stale", "retracted", "invalid"); err != nil {
		return err
	}
	if err := validateOptionalDigest("supersedes frame digest", f.Lifecycle.SupersedesDigest); err != nil {
		return err
	}
	if f.Lifecycle.State == "superseded" && !f.Lifecycle.SupersedesDigest.Present {
		return fmt.Errorf("%w: superseded frame requires prior digest", ErrInvalid)
	}
	if f.Lifecycle.State == "current" && f.Lifecycle.SupersedesDigest.Present {
		return fmt.Errorf("%w: current frame cannot supersede another frame", ErrInvalid)
	}
	return nil
}

func (s FrameSlot) Validate() error {
	if err := validateIdentifier("frame role ID", s.RoleID); err != nil {
		return err
	}
	if err := validateEnum("frame slot status", s.Status, "resolved", "unknown", "not_observed", "not_provided", "not_applicable", "withheld", "redacted", "unresolved", "contradictory", "invalid", "confirmed_absent"); err != nil {
		return err
	}
	if err := validateEnum("frame slot cardinality", s.Cardinality, "one", "many"); err != nil {
		return err
	}
	if len(s.Values) > 64 {
		return fmt.Errorf("%w: frame slot value count exceeds 64", ErrInvalid)
	}
	if s.Status == "resolved" {
		if len(s.Values) == 0 {
			return fmt.Errorf("%w: resolved frame slot requires a value", ErrInvalid)
		}
	} else if len(s.Values) != 0 {
		return fmt.Errorf("%w: non-resolved frame slot cannot carry values", ErrInvalid)
	}
	if s.Cardinality == "one" && len(s.Values) > 1 {
		return fmt.Errorf("%w: single-cardinality frame slot has multiple values", ErrInvalid)
	}
	var prior []byte
	for i, value := range s.Values {
		if err := value.Validate(); err != nil {
			return err
		}
		encoded := encodedTypedValue(value)
		if i > 0 && bytes.Compare(prior, encoded) >= 0 {
			return fmt.Errorf("%w: frame slot values must be canonically sorted and unique", ErrInvalid)
		}
		prior = encoded
	}
	if err := validateSortedUniqueDigests("frame slot packet digests", s.PacketDigests, 256, 1); err != nil {
		return err
	}
	if err := validateSortedUniqueText("frame slot mapping IDs", s.MappingIDs, 32); err != nil {
		return err
	}
	if err := validateEnum("frame slot conflict", s.Conflict, "none", "preserved", "unresolved"); err != nil {
		return err
	}
	if s.Status == "contradictory" && s.Conflict == "none" {
		return fmt.Errorf("%w: contradictory slot must preserve or leave conflict unresolved", ErrInvalid)
	}
	return nil
}

func encodedTypedValue(value TypedValue) []byte {
	var enc cborlite.Encoder
	encodeTypedValue(&enc, value)
	return enc.Bytes()
}

// CanonicalTypedValueBytes returns the deterministic-CBOR encoding used to
// order values inside a Semantic Frame slot. It is an implementation helper;
// the repository CDDL and prose specification remain normative.
func CanonicalTypedValueBytes(value TypedValue) ([]byte, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	return encodedTypedValue(value), nil
}

func validateFrameContext(context PacketContext) error {
	if len(context.Dimensions) > 32 {
		return fmt.Errorf("%w: too many frame context dimensions", ErrInvalid)
	}
	for i, dimension := range context.Dimensions {
		if err := validateIdentifier("frame context dimension", dimension.Key); err != nil {
			return err
		}
		if i > 0 && strings.Compare(context.Dimensions[i-1].Key, dimension.Key) >= 0 {
			return fmt.Errorf("%w: frame context dimensions must be sorted", ErrInvalid)
		}
		if err := dimension.Value.Validate(); err != nil {
			return err
		}
	}
	if err := validateOptionalIdentifier("frame jurisdiction", context.Jurisdiction); err != nil {
		return err
	}
	if err := validateLanguage("frame language", context.Language); err != nil {
		return err
	}
	return validateOptionalIdentifier("frame scope", context.Scope)
}

func MarshalFrame(f Frame) ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(12)
	enc.Text(f.Version)
	enc.Text(f.UniverseID)
	enc.Text(f.FrameType)
	enc.Text(f.NativeKey)
	encodeTextSet(&enc, f.CanonicalCandidates)
	enc.Array(5)
	encodeDigestSet(&enc, f.Derivation.PacketDigests)
	encodeDigest(&enc, f.Derivation.ModuleSetDigest)
	encodeTextSet(&enc, f.Derivation.MappingIDs)
	encodeDigest(&enc, f.Derivation.CompilerContractDigest)
	enc.Text(f.Derivation.CompilerVersion)
	enc.Array(uint64(len(f.Slots)))
	for _, slot := range f.Slots {
		encodeFrameSlot(&enc, slot)
	}
	encodeFrameContext(&enc, f.Context)
	enc.Array(3)
	enc.Text(f.Time.ComposedAt)
	encodeOptionalText(&enc, f.Time.ValidFrom)
	encodeOptionalText(&enc, f.Time.ValidUntil)
	enc.Array(3)
	enc.Text(f.Epistemic.Lane)
	enc.Uint(f.Epistemic.CompletenessMillionths)
	enc.Text(f.Epistemic.ConflictStatus)
	enc.Array(2)
	enc.Text(f.Lifecycle.State)
	encodeOptionalDigest(&enc, f.Lifecycle.SupersedesDigest)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func encodeFrameSlot(enc *cborlite.Encoder, slot FrameSlot) {
	enc.Array(7)
	enc.Text(slot.RoleID)
	enc.Text(slot.Status)
	enc.Text(slot.Cardinality)
	enc.Array(uint64(len(slot.Values)))
	for _, value := range slot.Values {
		encodeTypedValue(enc, value)
	}
	encodeDigestSet(enc, slot.PacketDigests)
	encodeTextSet(enc, slot.MappingIDs)
	enc.Text(slot.Conflict)
}

func encodeFrameContext(enc *cborlite.Encoder, context PacketContext) {
	enc.Array(4)
	enc.Array(uint64(len(context.Dimensions)))
	for _, dimension := range context.Dimensions {
		enc.Array(2)
		enc.Text(dimension.Key)
		encodeTypedValue(enc, dimension.Value)
	}
	encodeOptionalText(enc, context.Jurisdiction)
	encodeOptionalText(enc, context.Language)
	encodeOptionalText(enc, context.Scope)
}

func UnmarshalFrame(data []byte) (Frame, error) {
	var f Frame
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return f, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 12 {
		return f, fmt.Errorf("%w: frame array", ErrInvalid)
	}
	if f.Version, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if f.UniverseID, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if f.FrameType, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if f.NativeKey, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if f.CanonicalCandidates, err = decodeTextSet(dec, 32, 0); err != nil {
		return f, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 5 {
		return f, fmt.Errorf("%w: frame derivation array", ErrInvalid)
	}
	if f.Derivation.PacketDigests, err = decodeDigestSet(dec, 4096, 1); err != nil {
		return f, err
	}
	if f.Derivation.ModuleSetDigest, err = decodeDigest(dec); err != nil {
		return f, err
	}
	if f.Derivation.MappingIDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return f, err
	}
	if f.Derivation.CompilerContractDigest, err = decodeDigest(dec); err != nil {
		return f, err
	}
	if f.Derivation.CompilerVersion, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	count, countErr := dec.Array()
	if countErr != nil || count < 1 || count > 256 {
		return f, fmt.Errorf("%w: frame slot count", ErrInvalid)
	}
	for i := uint64(0); i < count; i++ {
		slot, slotErr := decodeFrameSlot(dec)
		if slotErr != nil {
			return f, slotErr
		}
		f.Slots = append(f.Slots, slot)
	}
	if f.Context, err = decodeFrameContext(dec); err != nil {
		return f, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 3 {
		return f, fmt.Errorf("%w: frame time array", ErrInvalid)
	}
	if f.Time.ComposedAt, err = dec.Text(20); err != nil {
		return f, err
	}
	if f.Time.ValidFrom, err = decodeOptionalText(dec, 20); err != nil {
		return f, err
	}
	if f.Time.ValidUntil, err = decodeOptionalText(dec, 20); err != nil {
		return f, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 3 {
		return f, fmt.Errorf("%w: frame epistemic array", ErrInvalid)
	}
	if f.Epistemic.Lane, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if f.Epistemic.CompletenessMillionths, err = dec.Uint(); err != nil {
		return f, err
	}
	if f.Epistemic.ConflictStatus, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
		return f, fmt.Errorf("%w: frame lifecycle array", ErrInvalid)
	}
	if f.Lifecycle.State, err = dec.Text(MaxIdentifier); err != nil {
		return f, err
	}
	if f.Lifecycle.SupersedesDigest, err = decodeOptionalDigest(dec); err != nil {
		return f, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return f, err
	}
	if err = finish(dec); err != nil {
		return f, err
	}
	return f, f.Validate()
}

func decodeFrameSlot(dec *cborlite.Decoder) (FrameSlot, error) {
	var slot FrameSlot
	n, err := dec.Array()
	if err != nil || n != 7 {
		return slot, fmt.Errorf("%w: frame slot array", ErrInvalid)
	}
	if slot.RoleID, err = dec.Text(MaxIdentifier); err != nil {
		return slot, err
	}
	if slot.Status, err = dec.Text(MaxIdentifier); err != nil {
		return slot, err
	}
	if slot.Cardinality, err = dec.Text(MaxIdentifier); err != nil {
		return slot, err
	}
	valueCount, countErr := dec.Array()
	if countErr != nil || valueCount > 64 {
		return slot, fmt.Errorf("%w: frame slot value count", ErrInvalid)
	}
	for i := uint64(0); i < valueCount; i++ {
		value, valueErr := decodeTypedValue(dec)
		if valueErr != nil {
			return slot, valueErr
		}
		slot.Values = append(slot.Values, value)
	}
	if slot.PacketDigests, err = decodeDigestSet(dec, 256, 1); err != nil {
		return slot, err
	}
	if slot.MappingIDs, err = decodeTextSet(dec, 32, 0); err != nil {
		return slot, err
	}
	if slot.Conflict, err = dec.Text(MaxIdentifier); err != nil {
		return slot, err
	}
	return slot, nil
}

func decodeFrameContext(dec *cborlite.Decoder) (PacketContext, error) {
	var context PacketContext
	n, err := dec.Array()
	if err != nil || n != 4 {
		return context, fmt.Errorf("%w: frame context array", ErrInvalid)
	}
	count, countErr := dec.Array()
	if countErr != nil || count > 32 {
		return context, fmt.Errorf("%w: frame context dimension count", ErrInvalid)
	}
	for i := uint64(0); i < count; i++ {
		if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
			return context, fmt.Errorf("%w: frame context dimension array", ErrInvalid)
		}
		var dimension ContextDimension
		if dimension.Key, err = dec.Text(MaxIdentifier); err != nil {
			return context, err
		}
		if dimension.Value, err = decodeTypedValue(dec); err != nil {
			return context, err
		}
		context.Dimensions = append(context.Dimensions, dimension)
	}
	if context.Jurisdiction, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return context, err
	}
	if context.Language, err = decodeOptionalText(dec, 63); err != nil {
		return context, err
	}
	if context.Scope, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return context, err
	}
	return context, nil
}

type MappingNative struct {
	OriginID       string
	SchemaRef      OptionalText
	Term           string
	LocatorPattern OptionalText
}

type MappingSemantic struct {
	ConceptID string
	RoleID    OptionalText
}

type MappingScope struct {
	UniverseID    string
	Jurisdictions []string
	Languages     []string
	ConditionIDs  []string
}

type MappingClaim struct {
	Version                 string
	MappingID               string
	Native                  MappingNative
	Semantic                MappingSemantic
	Relation                string
	Scope                   MappingScope
	Status                  string
	EvidencePacketDigests   []Digest
	ModuleID                string
	ReviewDecisionDigest    OptionalDigest
	SupersedesMappingDigest OptionalDigest
}

func (m MappingClaim) Validate() error {
	if m.Version != MappingVersion {
		return fmt.Errorf("%w: mapping version must be %q", ErrInvalid, MappingVersion)
	}
	for name, value := range map[string]string{
		"mapping ID":          m.MappingID,
		"mapping origin ID":   m.Native.OriginID,
		"mapping native term": m.Native.Term,
		"semantic concept ID": m.Semantic.ConceptID,
		"mapping universe ID": m.Scope.UniverseID,
		"mapping module ID":   m.ModuleID,
	} {
		if err := validateIdentifier(name, value); err != nil {
			return err
		}
	}
	if err := validateOptionalIdentifier("mapping native schema", m.Native.SchemaRef); err != nil {
		return err
	}
	if m.Native.LocatorPattern.Present {
		if err := validateText("mapping locator pattern", m.Native.LocatorPattern.Value, MaxLocator, false); err != nil {
			return err
		}
	} else if m.Native.LocatorPattern.Value != "" {
		return fmt.Errorf("%w: absent mapping locator carries a value", ErrInvalid)
	}
	if err := validateOptionalIdentifier("mapping role ID", m.Semantic.RoleID); err != nil {
		return err
	}
	if err := validateEnum("mapping relation", m.Relation, "exact", "close", "broad", "narrow", "contextual", "candidate"); err != nil {
		return err
	}
	if err := validateEnum("mapping status", m.Status, "candidate", "reviewed", "disputed", "revoked"); err != nil {
		return err
	}
	if err := validateSortedUniqueText("mapping jurisdictions", m.Scope.Jurisdictions, 32); err != nil {
		return err
	}
	if err := validateSortedUniqueText("mapping languages", m.Scope.Languages, 32); err != nil {
		return err
	}
	for _, language := range m.Scope.Languages {
		if err := validateLanguage("mapping language", OptionalText{Present: true, Value: language}); err != nil {
			return err
		}
	}
	if err := validateSortedUniqueText("mapping conditions", m.Scope.ConditionIDs, 32); err != nil {
		return err
	}
	if err := validateSortedUniqueDigests("mapping evidence packets", m.EvidencePacketDigests, 256, 0); err != nil {
		return err
	}
	if err := validateOptionalDigest("mapping review decision", m.ReviewDecisionDigest); err != nil {
		return err
	}
	if err := validateOptionalDigest("supersedes mapping", m.SupersedesMappingDigest); err != nil {
		return err
	}
	if m.Status == "candidate" {
		if m.ReviewDecisionDigest.Present {
			return fmt.Errorf("%w: candidate mapping cannot carry a review decision", ErrInvalid)
		}
	} else if !m.ReviewDecisionDigest.Present {
		return fmt.Errorf("%w: non-candidate mapping requires a review decision", ErrInvalid)
	}
	if m.Relation == "candidate" && m.Status != "candidate" {
		return fmt.Errorf("%w: candidate relation cannot be reviewed or revoked", ErrInvalid)
	}
	return nil
}

func MarshalMappingClaim(m MappingClaim) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(12)
	enc.Text(m.Version)
	enc.Text(m.MappingID)
	enc.Array(4)
	enc.Text(m.Native.OriginID)
	encodeOptionalText(&enc, m.Native.SchemaRef)
	enc.Text(m.Native.Term)
	encodeOptionalText(&enc, m.Native.LocatorPattern)
	enc.Array(2)
	enc.Text(m.Semantic.ConceptID)
	encodeOptionalText(&enc, m.Semantic.RoleID)
	enc.Text(m.Relation)
	enc.Array(4)
	enc.Text(m.Scope.UniverseID)
	encodeTextSet(&enc, m.Scope.Jurisdictions)
	encodeTextSet(&enc, m.Scope.Languages)
	encodeTextSet(&enc, m.Scope.ConditionIDs)
	enc.Text(m.Status)
	encodeDigestSet(&enc, m.EvidencePacketDigests)
	enc.Text(m.ModuleID)
	encodeOptionalDigest(&enc, m.ReviewDecisionDigest)
	encodeOptionalDigest(&enc, m.SupersedesMappingDigest)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalMappingClaim(data []byte) (MappingClaim, error) {
	var m MappingClaim
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return m, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 12 {
		return m, fmt.Errorf("%w: mapping claim array", ErrInvalid)
	}
	if m.Version, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.MappingID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 4 {
		return m, fmt.Errorf("%w: mapping native array", ErrInvalid)
	}
	if m.Native.OriginID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.Native.SchemaRef, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return m, err
	}
	if m.Native.Term, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.Native.LocatorPattern, err = decodeOptionalText(dec, MaxLocator); err != nil {
		return m, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 2 {
		return m, fmt.Errorf("%w: mapping semantic array", ErrInvalid)
	}
	if m.Semantic.ConceptID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.Semantic.RoleID, err = decodeOptionalText(dec, MaxIdentifier); err != nil {
		return m, err
	}
	if m.Relation, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 4 {
		return m, fmt.Errorf("%w: mapping scope array", ErrInvalid)
	}
	if m.Scope.UniverseID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.Scope.Jurisdictions, err = decodeTextSet(dec, 32, 0); err != nil {
		return m, err
	}
	if m.Scope.Languages, err = decodeTextSet(dec, 32, 0); err != nil {
		return m, err
	}
	if m.Scope.ConditionIDs, err = decodeTextSet(dec, 32, 0); err != nil {
		return m, err
	}
	if m.Status, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.EvidencePacketDigests, err = decodeDigestSet(dec, 256, 0); err != nil {
		return m, err
	}
	if m.ModuleID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.ReviewDecisionDigest, err = decodeOptionalDigest(dec); err != nil {
		return m, err
	}
	if m.SupersedesMappingDigest, err = decodeOptionalDigest(dec); err != nil {
		return m, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return m, err
	}
	if err = finish(dec); err != nil {
		return m, err
	}
	return m, m.Validate()
}

type OntologyModuleManifest struct {
	Version              string
	ModuleID             string
	SemanticVersion      string
	Status               string
	Imports              []string
	ConceptIDs           []string
	FrameTypeIDs         []string
	RoleIDs              []string
	MappingClaimDigests  []Digest
	QueryTemplateIDs     []string
	VisualizationIDs     []string
	SourceArtifactDigest Digest
	ReviewDecisionDigest OptionalDigest
}

func (m OntologyModuleManifest) Validate() error {
	if m.Version != ModuleVersion {
		return fmt.Errorf("%w: module version must be %q", ErrInvalid, ModuleVersion)
	}
	if err := validateIdentifier("module ID", m.ModuleID); err != nil {
		return err
	}
	if !semanticVersionPattern.MatchString(m.SemanticVersion) {
		return fmt.Errorf("%w: module semantic version must be MAJOR.MINOR.PATCH", ErrInvalid)
	}
	if err := validateEnum("module status", m.Status, "draft", "reviewed", "admitted", "deprecated", "superseded"); err != nil {
		return err
	}
	textSets := []struct {
		name string
		set  []string
		max  int
		min  int
	}{
		{"module imports", m.Imports, 64, 0},
		{"module concepts", m.ConceptIDs, 4096, 1},
		{"module frame types", m.FrameTypeIDs, 512, 0},
		{"module roles", m.RoleIDs, 4096, 0},
		{"module query templates", m.QueryTemplateIDs, 256, 0},
		{"module visualizations", m.VisualizationIDs, 256, 0},
	}
	for _, item := range textSets {
		if len(item.set) < item.min {
			return fmt.Errorf("%w: %s requires at least %d item", ErrInvalid, item.name, item.min)
		}
		if err := validateSortedUniqueText(item.name, item.set, item.max); err != nil {
			return err
		}
	}
	if err := validateSortedUniqueDigests("module mapping claims", m.MappingClaimDigests, 4096, 0); err != nil {
		return err
	}
	if err := validateOptionalDigest("module review decision", m.ReviewDecisionDigest); err != nil {
		return err
	}
	if m.Status == "draft" && m.ReviewDecisionDigest.Present {
		return fmt.Errorf("%w: draft module cannot carry a review decision", ErrInvalid)
	}
	if m.Status != "draft" && !m.ReviewDecisionDigest.Present {
		return fmt.Errorf("%w: released module requires a review decision", ErrInvalid)
	}
	return nil
}

func MarshalOntologyModule(m OntologyModuleManifest) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(14)
	enc.Text(m.Version)
	enc.Text(m.ModuleID)
	enc.Text(m.SemanticVersion)
	enc.Text(m.Status)
	encodeTextSet(&enc, m.Imports)
	encodeTextSet(&enc, m.ConceptIDs)
	encodeTextSet(&enc, m.FrameTypeIDs)
	encodeTextSet(&enc, m.RoleIDs)
	encodeDigestSet(&enc, m.MappingClaimDigests)
	encodeTextSet(&enc, m.QueryTemplateIDs)
	encodeTextSet(&enc, m.VisualizationIDs)
	encodeDigest(&enc, m.SourceArtifactDigest)
	encodeOptionalDigest(&enc, m.ReviewDecisionDigest)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalOntologyModule(data []byte) (OntologyModuleManifest, error) {
	var m OntologyModuleManifest
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return m, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 14 {
		return m, fmt.Errorf("%w: ontology module array", ErrInvalid)
	}
	if m.Version, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.ModuleID, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.SemanticVersion, err = dec.Text(64); err != nil {
		return m, err
	}
	if m.Status, err = dec.Text(MaxIdentifier); err != nil {
		return m, err
	}
	if m.Imports, err = decodeTextSet(dec, 64, 0); err != nil {
		return m, err
	}
	if m.ConceptIDs, err = decodeTextSet(dec, 4096, 1); err != nil {
		return m, err
	}
	if m.FrameTypeIDs, err = decodeTextSet(dec, 512, 0); err != nil {
		return m, err
	}
	if m.RoleIDs, err = decodeTextSet(dec, 4096, 0); err != nil {
		return m, err
	}
	if m.MappingClaimDigests, err = decodeDigestSet(dec, 4096, 0); err != nil {
		return m, err
	}
	if m.QueryTemplateIDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return m, err
	}
	if m.VisualizationIDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return m, err
	}
	if m.SourceArtifactDigest, err = decodeDigest(dec); err != nil {
		return m, err
	}
	if m.ReviewDecisionDigest, err = decodeOptionalDigest(dec); err != nil {
		return m, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return m, err
	}
	if err = finish(dec); err != nil {
		return m, err
	}
	return m, m.Validate()
}

type SemanticUniverse struct {
	Version             string
	UniverseID          string
	SemanticVersion     string
	Title               string
	ModuleIDs           []string
	FrameTypeIDs        []string
	SourceOriginIDs     []string
	MappingClaimDigests []Digest
	MaterializedViewIDs []string
	QueryTemplateIDs    []string
	VisualizationIDs    []string
	UpdatePolicyID      string
	EvaluationSuiteID   string
	ModuleSetDigest     Digest
	CompiledAt          string
}

func (u SemanticUniverse) Validate() error {
	if u.Version != UniverseVersion {
		return fmt.Errorf("%w: universe version must be %q", ErrInvalid, UniverseVersion)
	}
	for name, value := range map[string]string{
		"universe ID":         u.UniverseID,
		"update policy ID":    u.UpdatePolicyID,
		"evaluation suite ID": u.EvaluationSuiteID,
	} {
		if err := validateIdentifier(name, value); err != nil {
			return err
		}
	}
	if !semanticVersionPattern.MatchString(u.SemanticVersion) {
		return fmt.Errorf("%w: universe semantic version must be MAJOR.MINOR.PATCH", ErrInvalid)
	}
	if err := validateText("universe title", u.Title, MaxShortText, false); err != nil {
		return err
	}
	sets := []struct {
		name string
		set  []string
		max  int
		min  int
	}{
		{"universe modules", u.ModuleIDs, 64, 1},
		{"universe frame types", u.FrameTypeIDs, 512, 1},
		{"universe origins", u.SourceOriginIDs, 1024, 0},
		{"universe views", u.MaterializedViewIDs, 64, 0},
		{"universe query templates", u.QueryTemplateIDs, 256, 0},
		{"universe visualizations", u.VisualizationIDs, 256, 0},
	}
	for _, item := range sets {
		if len(item.set) < item.min {
			return fmt.Errorf("%w: %s requires at least %d item", ErrInvalid, item.name, item.min)
		}
		if err := validateSortedUniqueText(item.name, item.set, item.max); err != nil {
			return err
		}
	}
	if err := validateSortedUniqueDigests("universe mappings", u.MappingClaimDigests, 4096, 0); err != nil {
		return err
	}
	return validateUTCSecond("universe compiled at", u.CompiledAt)
}

func MarshalSemanticUniverse(u SemanticUniverse) ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	var enc cborlite.Encoder
	enc.Array(16)
	enc.Text(u.Version)
	enc.Text(u.UniverseID)
	enc.Text(u.SemanticVersion)
	enc.Text(u.Title)
	encodeTextSet(&enc, u.ModuleIDs)
	encodeTextSet(&enc, u.FrameTypeIDs)
	encodeTextSet(&enc, u.SourceOriginIDs)
	encodeDigestSet(&enc, u.MappingClaimDigests)
	encodeTextSet(&enc, u.MaterializedViewIDs)
	encodeTextSet(&enc, u.QueryTemplateIDs)
	encodeTextSet(&enc, u.VisualizationIDs)
	enc.Text(u.UpdatePolicyID)
	enc.Text(u.EvaluationSuiteID)
	encodeDigest(&enc, u.ModuleSetDigest)
	enc.Text(u.CompiledAt)
	enc.Array(0)
	return boundedEncoding(&enc)
}

func UnmarshalSemanticUniverse(data []byte) (SemanticUniverse, error) {
	var u SemanticUniverse
	dec, err := checkedDocument(data, MaxDocumentBytes)
	if err != nil {
		return u, err
	}
	if n, arrayErr := dec.Array(); arrayErr != nil || n != 16 {
		return u, fmt.Errorf("%w: universe array", ErrInvalid)
	}
	if u.Version, err = dec.Text(MaxIdentifier); err != nil {
		return u, err
	}
	if u.UniverseID, err = dec.Text(MaxIdentifier); err != nil {
		return u, err
	}
	if u.SemanticVersion, err = dec.Text(64); err != nil {
		return u, err
	}
	if u.Title, err = dec.Text(MaxShortText); err != nil {
		return u, err
	}
	if u.ModuleIDs, err = decodeTextSet(dec, 64, 1); err != nil {
		return u, err
	}
	if u.FrameTypeIDs, err = decodeTextSet(dec, 512, 1); err != nil {
		return u, err
	}
	if u.SourceOriginIDs, err = decodeTextSet(dec, 1024, 0); err != nil {
		return u, err
	}
	if u.MappingClaimDigests, err = decodeDigestSet(dec, 4096, 0); err != nil {
		return u, err
	}
	if u.MaterializedViewIDs, err = decodeTextSet(dec, 64, 0); err != nil {
		return u, err
	}
	if u.QueryTemplateIDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return u, err
	}
	if u.VisualizationIDs, err = decodeTextSet(dec, 256, 0); err != nil {
		return u, err
	}
	if u.UpdatePolicyID, err = dec.Text(MaxIdentifier); err != nil {
		return u, err
	}
	if u.EvaluationSuiteID, err = dec.Text(MaxIdentifier); err != nil {
		return u, err
	}
	if u.ModuleSetDigest, err = decodeDigest(dec); err != nil {
		return u, err
	}
	if u.CompiledAt, err = dec.Text(20); err != nil {
		return u, err
	}
	if err = expectEmptyExtensions(dec); err != nil {
		return u, err
	}
	if err = finish(dec); err != nil {
		return u, err
	}
	return u, u.Validate()
}

func sortedDigestUnion(slots []FrameSlot) []Digest {
	set := make(map[Digest]struct{})
	for _, slot := range slots {
		for _, digest := range slot.PacketDigests {
			set[digest] = struct{}{}
		}
	}
	values := make([]Digest, 0, len(set))
	for digest := range set {
		values = append(values, digest)
	}
	sort.Slice(values, func(i, j int) bool { return bytes.Compare(values[i][:], values[j][:]) < 0 })
	return values
}
