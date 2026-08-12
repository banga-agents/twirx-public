// Package universesnapshot implements dependency-free E4 storage prototypes.
// These layouts are derived indexes over canonical Semantic Frame bytes; they
// do not redefine frame identity or constitute an admitted public snapshot.
package universesnapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/jsonbounded"
)

const (
	NativeFormat   = "tw.universe-native-prototype/0.1"
	ColumnarFormat = "tw.universe-columnar-prototype/0.1"
	MaxFrames      = 200000
	MaxBytes       = 1 << 30
)

var ErrInvalid = errors.New("universesnapshot: invalid prototype")

type SourceFrame struct {
	Digest dataplane.Digest
	CBOR   []byte
	Frame  dataplane.Frame
}

type Query struct {
	UniverseID string
	FrameType  string
	NativeKey  string
	SlotRole   string
	SlotValue  *dataplane.TypedValue
	Limit      uint32
}

type NativeEntry struct {
	Digest    string `json:"digest"`
	CBOR      []byte `json:"cbor"`
	Universe  uint32 `json:"universe"`
	FrameType uint32 `json:"frame_type"`
	NativeKey string `json:"native_key"`
	TrustLane string `json:"trust_lane"`
	Lifecycle string `json:"lifecycle"`
}

type Posting struct {
	Key      string   `json:"key"`
	FrameIDs []uint32 `json:"frame_ids"`
}

type NativeArtifact struct {
	Format     string        `json:"format"`
	Universes  []string      `json:"universes"`
	FrameTypes []string      `json:"frame_types"`
	Entries    []NativeEntry `json:"entries"`
	Postings   []Posting     `json:"postings"`
}

type ColumnSlot struct {
	Role   string                 `json:"role"`
	Status string                 `json:"status"`
	Values []dataplane.TypedValue `json:"values"`
}

type ColumnarArtifact struct {
	Format     string         `json:"format"`
	Digests    []string       `json:"digests"`
	CBOR       [][]byte       `json:"cbor"`
	Universes  []string       `json:"universes"`
	FrameTypes []string       `json:"frame_types"`
	NativeKeys []string       `json:"native_keys"`
	TrustLanes []string       `json:"trust_lanes"`
	Lifecycles []string       `json:"lifecycles"`
	Slots      [][]ColumnSlot `json:"slots"`
}

type NativeRuntime struct {
	artifact NativeArtifact
	postings map[string][]uint32
	byDigest map[dataplane.Digest]uint32
}

type ColumnarRuntime struct {
	artifact ColumnarArtifact
	byDigest map[dataplane.Digest]uint32
}

func (runtime *NativeRuntime) Layout() string   { return "native_posting_index" }
func (runtime *ColumnarRuntime) Layout() string { return "columnar_scan" }
func (runtime *NativeRuntime) FrameCount() uint64 {
	return uint64(len(runtime.artifact.Entries))
}
func (runtime *ColumnarRuntime) FrameCount() uint64 {
	return uint64(len(runtime.artifact.Digests))
}

func BuildNative(source []SourceFrame) ([]byte, dataplane.Digest, error) {
	ordered, err := validateAndOrder(source)
	if err != nil {
		return nil, dataplane.Digest{}, err
	}
	universes, universeIDs := dictionary(ordered, func(frame dataplane.Frame) string { return frame.UniverseID })
	frameTypes, typeIDs := dictionary(ordered, func(frame dataplane.Frame) string { return frame.FrameType })
	postingMap := make(map[string][]uint32)
	artifact := NativeArtifact{Format: NativeFormat, Universes: universes, FrameTypes: frameTypes, Entries: make([]NativeEntry, len(ordered))}
	for index, sourceFrame := range ordered {
		id := uint32(index)
		frame := sourceFrame.Frame
		artifact.Entries[index] = NativeEntry{Digest: digestText(sourceFrame.Digest), CBOR: append([]byte(nil), sourceFrame.CBOR...), Universe: universeIDs[frame.UniverseID], FrameType: typeIDs[frame.FrameType], NativeKey: frame.NativeKey, TrustLane: frame.Epistemic.Lane, Lifecycle: frame.Lifecycle.State}
		keys, keyErr := framePostingKeys(frame)
		if keyErr != nil {
			return nil, dataplane.Digest{}, keyErr
		}
		for _, key := range keys {
			postingMap[key] = append(postingMap[key], id)
		}
	}
	keys := make([]string, 0, len(postingMap))
	for key := range postingMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		artifact.Postings = append(artifact.Postings, Posting{Key: key, FrameIDs: postingMap[key]})
	}
	encoded, err := marshal(artifact)
	if err != nil {
		return nil, dataplane.Digest{}, err
	}
	return encoded, dataplane.DigestBytes(encoded), nil
}

func OpenNative(data []byte, expected dataplane.Digest) (*NativeRuntime, error) {
	if dataplane.DigestBytes(data) != expected {
		return nil, fmt.Errorf("%w: native artifact digest mismatch", ErrInvalid)
	}
	var artifact NativeArtifact
	if err := decode(data, &artifact); err != nil {
		return nil, err
	}
	if artifact.Format != NativeFormat || len(artifact.Entries) == 0 || len(artifact.Entries) > MaxFrames || !sortedUniqueText(artifact.Universes) || !sortedUniqueText(artifact.FrameTypes) {
		return nil, fmt.Errorf("%w: native metadata", ErrInvalid)
	}
	runtime := &NativeRuntime{artifact: artifact, postings: make(map[string][]uint32, len(artifact.Postings)), byDigest: make(map[dataplane.Digest]uint32, len(artifact.Entries))}
	var prior dataplane.Digest
	for index, entry := range artifact.Entries {
		digest, err := parseDigest(entry.Digest)
		if err != nil || dataplane.DigestBytes(entry.CBOR) != digest || (index > 0 && bytes.Compare(prior[:], digest[:]) >= 0) || int(entry.Universe) >= len(artifact.Universes) || int(entry.FrameType) >= len(artifact.FrameTypes) {
			return nil, fmt.Errorf("%w: native entry %d", ErrInvalid, index)
		}
		frame, err := dataplane.UnmarshalFrame(entry.CBOR)
		if err != nil || frame.UniverseID != artifact.Universes[entry.Universe] || frame.FrameType != artifact.FrameTypes[entry.FrameType] || frame.NativeKey != entry.NativeKey || frame.Epistemic.Lane != entry.TrustLane || frame.Lifecycle.State != entry.Lifecycle {
			return nil, fmt.Errorf("%w: native entry %d does not reconcile", ErrInvalid, index)
		}
		runtime.byDigest[digest] = uint32(index)
		prior = digest
	}
	for index, posting := range artifact.Postings {
		if posting.Key == "" || len(posting.FrameIDs) == 0 || (index > 0 && artifact.Postings[index-1].Key >= posting.Key) || !sortedUniqueIDs(posting.FrameIDs, uint32(len(artifact.Entries))) {
			return nil, fmt.Errorf("%w: posting %d", ErrInvalid, index)
		}
		runtime.postings[posting.Key] = posting.FrameIDs
	}
	if err := runtime.reconcilePostings(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func BuildColumnar(source []SourceFrame) ([]byte, dataplane.Digest, error) {
	ordered, err := validateAndOrder(source)
	if err != nil {
		return nil, dataplane.Digest{}, err
	}
	artifact := ColumnarArtifact{Format: ColumnarFormat}
	for _, sourceFrame := range ordered {
		frame := sourceFrame.Frame
		artifact.Digests = append(artifact.Digests, digestText(sourceFrame.Digest))
		artifact.CBOR = append(artifact.CBOR, append([]byte(nil), sourceFrame.CBOR...))
		artifact.Universes = append(artifact.Universes, frame.UniverseID)
		artifact.FrameTypes = append(artifact.FrameTypes, frame.FrameType)
		artifact.NativeKeys = append(artifact.NativeKeys, frame.NativeKey)
		artifact.TrustLanes = append(artifact.TrustLanes, frame.Epistemic.Lane)
		artifact.Lifecycles = append(artifact.Lifecycles, frame.Lifecycle.State)
		row := make([]ColumnSlot, len(frame.Slots))
		for i, slot := range frame.Slots {
			row[i] = ColumnSlot{Role: slot.RoleID, Status: slot.Status, Values: append([]dataplane.TypedValue(nil), slot.Values...)}
		}
		artifact.Slots = append(artifact.Slots, row)
	}
	encoded, err := marshal(artifact)
	if err != nil {
		return nil, dataplane.Digest{}, err
	}
	return encoded, dataplane.DigestBytes(encoded), nil
}

func OpenColumnar(data []byte, expected dataplane.Digest) (*ColumnarRuntime, error) {
	if dataplane.DigestBytes(data) != expected {
		return nil, fmt.Errorf("%w: columnar artifact digest mismatch", ErrInvalid)
	}
	var artifact ColumnarArtifact
	if err := decode(data, &artifact); err != nil {
		return nil, err
	}
	count := len(artifact.Digests)
	if artifact.Format != ColumnarFormat || count == 0 || count > MaxFrames || len(artifact.CBOR) != count || len(artifact.Universes) != count || len(artifact.FrameTypes) != count || len(artifact.NativeKeys) != count || len(artifact.TrustLanes) != count || len(artifact.Lifecycles) != count || len(artifact.Slots) != count {
		return nil, fmt.Errorf("%w: column counts", ErrInvalid)
	}
	runtime := &ColumnarRuntime{artifact: artifact, byDigest: make(map[dataplane.Digest]uint32, count)}
	var prior dataplane.Digest
	for index := 0; index < count; index++ {
		digest, err := parseDigest(artifact.Digests[index])
		if err != nil || dataplane.DigestBytes(artifact.CBOR[index]) != digest || (index > 0 && bytes.Compare(prior[:], digest[:]) >= 0) {
			return nil, fmt.Errorf("%w: column row %d", ErrInvalid, index)
		}
		frame, err := dataplane.UnmarshalFrame(artifact.CBOR[index])
		if err != nil || frame.UniverseID != artifact.Universes[index] || frame.FrameType != artifact.FrameTypes[index] || frame.NativeKey != artifact.NativeKeys[index] || frame.Epistemic.Lane != artifact.TrustLanes[index] || frame.Lifecycle.State != artifact.Lifecycles[index] || !sameColumnSlots(frame.Slots, artifact.Slots[index]) {
			return nil, fmt.Errorf("%w: column row %d does not reconcile", ErrInvalid, index)
		}
		runtime.byDigest[digest] = uint32(index)
		prior = digest
	}
	return runtime, nil
}

func (runtime *NativeRuntime) Query(query Query) ([]dataplane.Digest, error) {
	keys, err := queryKeys(query)
	if err != nil {
		return nil, err
	}
	sets := make([][]uint32, 0, len(keys))
	for _, key := range keys {
		ids, ok := runtime.postings[key]
		if !ok {
			return []dataplane.Digest{}, nil
		}
		sets = append(sets, ids)
	}
	ids := intersect(sets)
	return nativeResults(runtime.artifact, ids, query.Limit)
}

func (runtime *ColumnarRuntime) Query(query Query) ([]dataplane.Digest, error) {
	if _, err := queryKeys(query); err != nil {
		return nil, err
	}
	results := make([]dataplane.Digest, 0)
	for index := range runtime.artifact.Digests {
		if query.UniverseID != "" && runtime.artifact.Universes[index] != query.UniverseID || query.FrameType != "" && runtime.artifact.FrameTypes[index] != query.FrameType || query.NativeKey != "" && runtime.artifact.NativeKeys[index] != query.NativeKey {
			continue
		}
		if query.SlotRole != "" && !columnHasSlot(runtime.artifact.Slots[index], query.SlotRole, *query.SlotValue) {
			continue
		}
		digest, _ := parseDigest(runtime.artifact.Digests[index])
		results = append(results, digest)
		if query.Limit > 0 && len(results) >= int(query.Limit) {
			break
		}
	}
	return results, nil
}

func (runtime *NativeRuntime) Trace(digest dataplane.Digest) ([]byte, error) {
	id, ok := runtime.byDigest[digest]
	if !ok {
		return nil, fmt.Errorf("%w: frame not found", ErrInvalid)
	}
	return append([]byte(nil), runtime.artifact.Entries[id].CBOR...), nil
}

func (runtime *ColumnarRuntime) Trace(digest dataplane.Digest) ([]byte, error) {
	id, ok := runtime.byDigest[digest]
	if !ok {
		return nil, fmt.Errorf("%w: frame not found", ErrInvalid)
	}
	return append([]byte(nil), runtime.artifact.CBOR[id]...), nil
}

func (runtime *NativeRuntime) reconcilePostings() error {
	expected := make(map[string][]uint32)
	for index, entry := range runtime.artifact.Entries {
		frame, _ := dataplane.UnmarshalFrame(entry.CBOR)
		keys, _ := framePostingKeys(frame)
		for _, key := range keys {
			expected[key] = append(expected[key], uint32(index))
		}
	}
	if len(expected) != len(runtime.postings) {
		return fmt.Errorf("%w: posting set mismatch", ErrInvalid)
	}
	for key, ids := range expected {
		if !sameIDs(ids, runtime.postings[key]) {
			return fmt.Errorf("%w: posting content mismatch", ErrInvalid)
		}
	}
	return nil
}

func validateAndOrder(source []SourceFrame) ([]SourceFrame, error) {
	if len(source) == 0 || len(source) > MaxFrames {
		return nil, fmt.Errorf("%w: frame count outside 1..%d", ErrInvalid, MaxFrames)
	}
	ordered := append([]SourceFrame(nil), source...)
	for index := range ordered {
		if len(ordered[index].CBOR) == 0 || len(ordered[index].CBOR) > dataplane.MaxDocumentBytes || dataplane.DigestBytes(ordered[index].CBOR) != ordered[index].Digest {
			return nil, fmt.Errorf("%w: source frame %d digest", ErrInvalid, index)
		}
		decoded, err := dataplane.UnmarshalFrame(ordered[index].CBOR)
		if err != nil {
			return nil, fmt.Errorf("%w: source frame %d: %v", ErrInvalid, index, err)
		}
		if decoded.NativeKey != ordered[index].Frame.NativeKey || decoded.UniverseID != ordered[index].Frame.UniverseID || decoded.FrameType != ordered[index].Frame.FrameType {
			return nil, fmt.Errorf("%w: source frame %d decoded identity", ErrInvalid, index)
		}
		ordered[index].Frame = decoded
	}
	sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(ordered[i].Digest[:], ordered[j].Digest[:]) < 0 })
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].Digest == ordered[i].Digest {
			return nil, fmt.Errorf("%w: duplicate frame", ErrInvalid)
		}
	}
	return ordered, nil
}

func dictionary(source []SourceFrame, value func(dataplane.Frame) string) ([]string, map[string]uint32) {
	set := make(map[string]struct{})
	for _, frame := range source {
		set[value(frame.Frame)] = struct{}{}
	}
	values := make([]string, 0, len(set))
	for item := range set {
		values = append(values, item)
	}
	sort.Strings(values)
	ids := make(map[string]uint32, len(values))
	for index, item := range values {
		ids[item] = uint32(index)
	}
	return values, ids
}

func framePostingKeys(frame dataplane.Frame) ([]string, error) {
	keys := []string{postingKey("universe", frame.UniverseID), postingKey("frame", frame.FrameType), postingKey("native", frame.NativeKey)}
	for _, slot := range frame.Slots {
		for _, value := range slot.Values {
			key, err := typedPostingKey(slot.RoleID, value)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	keys = uniqueText(keys)
	return keys, nil
}

func queryKeys(query Query) ([]string, error) {
	if query.Limit < 1 || query.Limit > 1000 {
		return nil, fmt.Errorf("%w: query limit outside 1..1000", ErrInvalid)
	}
	if query.SlotRole == "" && query.SlotValue != nil || query.SlotRole != "" && query.SlotValue == nil {
		return nil, fmt.Errorf("%w: slot role and value must be supplied together", ErrInvalid)
	}
	keys := make([]string, 0, 4)
	if query.UniverseID != "" {
		keys = append(keys, postingKey("universe", query.UniverseID))
	}
	if query.FrameType != "" {
		keys = append(keys, postingKey("frame", query.FrameType))
	}
	if query.NativeKey != "" {
		keys = append(keys, postingKey("native", query.NativeKey))
	}
	if query.SlotRole != "" {
		key, err := typedPostingKey(query.SlotRole, *query.SlotValue)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: unbounded query", ErrInvalid)
	}
	return keys, nil
}

func postingKey(kind, value string) string { return kind + "\x1f" + value }

func typedPostingKey(role string, value dataplane.TypedValue) (string, error) {
	encoded, err := dataplane.CanonicalTypedValueBytes(value)
	if err != nil {
		return "", err
	}
	return postingKey("slot", role+"\x1f"+hex.EncodeToString(encoded)), nil
}

func intersect(sets [][]uint32) []uint32 {
	if len(sets) == 0 {
		return nil
	}
	sort.Slice(sets, func(i, j int) bool { return len(sets[i]) < len(sets[j]) })
	result := append([]uint32(nil), sets[0]...)
	for _, set := range sets[1:] {
		out := result[:0]
		i, j := 0, 0
		for i < len(result) && j < len(set) {
			switch {
			case result[i] == set[j]:
				out = append(out, result[i])
				i++
				j++
			case result[i] < set[j]:
				i++
			default:
				j++
			}
		}
		result = out
	}
	return result
}

func nativeResults(artifact NativeArtifact, ids []uint32, limit uint32) ([]dataplane.Digest, error) {
	result := make([]dataplane.Digest, 0, len(ids))
	for _, id := range ids {
		digest, err := parseDigest(artifact.Entries[id].Digest)
		if err != nil {
			return nil, err
		}
		result = append(result, digest)
		if limit > 0 && len(result) >= int(limit) {
			break
		}
	}
	return result, nil
}

func columnHasSlot(slots []ColumnSlot, role string, value dataplane.TypedValue) bool {
	if err := value.Validate(); err != nil {
		return false
	}
	for _, slot := range slots {
		if slot.Role != role {
			continue
		}
		for _, candidate := range slot.Values {
			if candidate == value {
				return true
			}
		}
	}
	return false
}

func sameColumnSlots(frame []dataplane.FrameSlot, columns []ColumnSlot) bool {
	if len(frame) != len(columns) {
		return false
	}
	for index := range frame {
		if frame[index].RoleID != columns[index].Role || frame[index].Status != columns[index].Status || len(frame[index].Values) != len(columns[index].Values) {
			return false
		}
		for valueIndex := range frame[index].Values {
			if frame[index].Values[valueIndex] != columns[index].Values[valueIndex] {
				return false
			}
		}
	}
	return true
}

func marshal(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) == 0 || len(data) > MaxBytes {
		return nil, fmt.Errorf("%w: serialized artifact exceeds bounds", ErrInvalid)
	}
	return data, nil
}

func decode(data []byte, target any) error {
	if len(data) == 0 || len(data) > MaxBytes {
		return fmt.Errorf("%w: artifact size", ErrInvalid)
	}
	policy := jsonbounded.Policy{MaxBytes: MaxBytes, MaxDepth: 24, MaxScalarBytes: dataplane.MaxDocumentBytes * 2, MaxContainerEntries: MaxFrames * 1024, MaxTokens: MaxFrames * 8192}
	if err := jsonbounded.Decode(data, target, policy, true); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return nil
}

func digestText(digest dataplane.Digest) string { return "sha256:" + hex.EncodeToString(digest[:]) }

func parseDigest(value string) (dataplane.Digest, error) {
	var digest dataplane.Digest
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return digest, fmt.Errorf("%w: digest syntax", ErrInvalid)
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != sha256.Size {
		return digest, fmt.Errorf("%w: digest syntax", ErrInvalid)
	}
	copy(digest[:], decoded)
	return digest, nil
}

func sortedUniqueText(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func uniqueText(values []string) []string {
	if len(values) == 0 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func sortedUniqueIDs(values []uint32, maximum uint32) bool {
	for index, value := range values {
		if value >= maximum || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func sameIDs(left, right []uint32) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
