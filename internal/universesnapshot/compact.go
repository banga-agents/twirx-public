package universesnapshot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"syscall"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

const (
	CompactFormat = "tw.universe-compact-segment/0.1"
	compactMagic  = "TWUXS001"
	compactHeader = 64
	compactEntry  = 52
	maxPostingKey = 16 << 10
	maxDictionary = 4096
)

type compactPosting struct {
	keyOffset     int
	keyLength     int
	payloadOffset int
	payloadLength int
	count         uint32
}

// CompactRuntime is a verified read-only view over one immutable segment. A
// file-backed runtime owns an mmap and must be closed. Canonical frame bodies
// are decoded during admission and copied only when a trace is requested.
type CompactRuntime struct {
	data        []byte
	frameCount  uint32
	dictionary  []string
	entryOffset int
	bodyOffset  int
	postings    []compactPosting
	mapped      bool
}

func (runtime *CompactRuntime) Layout() string     { return "compact_native_segment" }
func (runtime *CompactRuntime) FrameCount() uint64 { return uint64(runtime.frameCount) }

func BuildCompact(source []SourceFrame) ([]byte, dataplane.Digest, error) {
	ordered, err := validateAndOrder(source)
	if err != nil {
		return nil, dataplane.Digest{}, err
	}
	dictionarySet := make(map[string]struct{})
	postingMap := make(map[string][]uint32)
	for index, item := range ordered {
		dictionarySet[item.Frame.UniverseID] = struct{}{}
		dictionarySet[item.Frame.FrameType] = struct{}{}
		keys, keyErr := framePostingKeys(item.Frame)
		if keyErr != nil {
			return nil, dataplane.Digest{}, keyErr
		}
		for _, key := range keys {
			postingMap[key] = append(postingMap[key], uint32(index))
		}
	}
	dictionary := make([]string, 0, len(dictionarySet))
	for value := range dictionarySet {
		dictionary = append(dictionary, value)
	}
	if len(dictionary) == 0 || len(dictionary) > maxDictionary {
		return nil, dataplane.Digest{}, fmt.Errorf("%w: compact dictionary count", ErrInvalid)
	}
	sort.Strings(dictionary)
	dictionaryIDs := make(map[string]uint32, len(dictionary))
	for index, value := range dictionary {
		dictionaryIDs[value] = uint32(index)
	}
	var dictionaryBytes bytes.Buffer
	putU32(&dictionaryBytes, uint32(len(dictionary)))
	for _, value := range dictionary {
		if value == "" || len(value) > maxPostingKey {
			return nil, dataplane.Digest{}, fmt.Errorf("%w: compact dictionary value", ErrInvalid)
		}
		putU32(&dictionaryBytes, uint32(len(value)))
		dictionaryBytes.WriteString(value)
	}
	entryOffset := compactHeader + dictionaryBytes.Len()
	bodyOffset := entryOffset + compactEntry*len(ordered)
	var entries, bodies bytes.Buffer
	for _, item := range ordered {
		entries.Write(item.Digest[:])
		putU64(&entries, uint64(bodyOffset+bodies.Len()))
		putU32(&entries, uint32(len(item.CBOR)))
		putU32(&entries, dictionaryIDs[item.Frame.UniverseID])
		putU32(&entries, dictionaryIDs[item.Frame.FrameType])
		bodies.Write(item.CBOR)
	}
	postingOffset := bodyOffset + bodies.Len()
	postingKeys := make([]string, 0, len(postingMap))
	for key := range postingMap {
		postingKeys = append(postingKeys, key)
	}
	sort.Strings(postingKeys)
	var postings bytes.Buffer
	for _, key := range postingKeys {
		if len(key) == 0 || len(key) > maxPostingKey {
			return nil, dataplane.Digest{}, fmt.Errorf("%w: compact posting key", ErrInvalid)
		}
		ids := postingMap[key]
		var payload bytes.Buffer
		var prior uint32
		for index, id := range ids {
			delta := uint64(id)
			if index > 0 {
				delta = uint64(id - prior)
			}
			var encoded [binary.MaxVarintLen64]byte
			n := binary.PutUvarint(encoded[:], delta)
			payload.Write(encoded[:n])
			prior = id
		}
		putU32(&postings, uint32(len(key)))
		postings.WriteString(key)
		putU32(&postings, uint32(len(ids)))
		putU32(&postings, uint32(payload.Len()))
		postings.Write(payload.Bytes())
	}
	endOffset := postingOffset + postings.Len()
	if endOffset > MaxBytes {
		return nil, dataplane.Digest{}, fmt.Errorf("%w: compact segment exceeds %d bytes", ErrInvalid, MaxBytes)
	}
	result := make([]byte, compactHeader, endOffset)
	copy(result[:8], compactMagic)
	binary.BigEndian.PutUint32(result[8:12], uint32(len(ordered)))
	binary.BigEndian.PutUint32(result[12:16], uint32(len(dictionary)))
	binary.BigEndian.PutUint32(result[16:20], uint32(len(postingKeys)))
	binary.BigEndian.PutUint64(result[24:32], uint64(compactHeader))
	binary.BigEndian.PutUint64(result[32:40], uint64(entryOffset))
	binary.BigEndian.PutUint64(result[40:48], uint64(bodyOffset))
	binary.BigEndian.PutUint64(result[48:56], uint64(postingOffset))
	binary.BigEndian.PutUint64(result[56:64], uint64(endOffset))
	result = append(result, dictionaryBytes.Bytes()...)
	result = append(result, entries.Bytes()...)
	result = append(result, bodies.Bytes()...)
	result = append(result, postings.Bytes()...)
	return result, dataplane.DigestBytes(result), nil
}

func OpenCompact(data []byte, expected dataplane.Digest) (*CompactRuntime, error) {
	return openCompact(data, expected, false)
}

func OpenCompactFile(path string, expected dataplane.Digest) (*CompactRuntime, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: open compact segment: %v", ErrInvalid, err)
	}
	defer syscall.Close(fd)
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Size < compactHeader || stat.Size > int64(MaxBytes) {
		return nil, fmt.Errorf("%w: compact file bounds or type", ErrInvalid)
	}
	data, err := syscall.Mmap(fd, 0, int(stat.Size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("%w: mmap compact segment: %v", ErrInvalid, err)
	}
	runtime, err := openCompact(data, expected, true)
	if err != nil {
		_ = syscall.Munmap(data)
		return nil, err
	}
	return runtime, nil
}

func (runtime *CompactRuntime) Close() error {
	if runtime == nil || !runtime.mapped || runtime.data == nil {
		return nil
	}
	data := runtime.data
	runtime.data = nil
	runtime.mapped = false
	return syscall.Munmap(data)
}

func openCompact(data []byte, expected dataplane.Digest, mapped bool) (*CompactRuntime, error) {
	if len(data) < compactHeader || len(data) > MaxBytes || dataplane.DigestBytes(data) != expected || string(data[:8]) != compactMagic {
		return nil, fmt.Errorf("%w: compact segment identity", ErrInvalid)
	}
	frames := binary.BigEndian.Uint32(data[8:12])
	dictionaryCount := binary.BigEndian.Uint32(data[12:16])
	postingCount := binary.BigEndian.Uint32(data[16:20])
	if frames == 0 || frames > MaxFrames || dictionaryCount == 0 || dictionaryCount > maxDictionary || postingCount == 0 || binary.BigEndian.Uint32(data[20:24]) != 0 {
		return nil, fmt.Errorf("%w: compact header counts", ErrInvalid)
	}
	offsets := []uint64{binary.BigEndian.Uint64(data[24:32]), binary.BigEndian.Uint64(data[32:40]), binary.BigEndian.Uint64(data[40:48]), binary.BigEndian.Uint64(data[48:56]), binary.BigEndian.Uint64(data[56:64])}
	if offsets[0] != compactHeader || offsets[4] != uint64(len(data)) || !(offsets[0] < offsets[1] && offsets[1] < offsets[2] && offsets[2] < offsets[3] && offsets[3] < offsets[4]) || offsets[2] != offsets[1]+uint64(frames)*compactEntry {
		return nil, fmt.Errorf("%w: compact section offsets", ErrInvalid)
	}
	runtime := &CompactRuntime{data: data, frameCount: frames, entryOffset: int(offsets[1]), bodyOffset: int(offsets[2]), mapped: mapped}
	dictionary, err := parseDictionary(data[int(offsets[0]):int(offsets[1])], dictionaryCount)
	if err != nil {
		return nil, err
	}
	runtime.dictionary = dictionary
	expectedPostings, err := runtime.validateEntries(int(offsets[3]))
	if err != nil {
		return nil, err
	}
	postings, memberships, err := parseCompactPostings(data, int(offsets[3]), postingCount, frames)
	if err != nil {
		return nil, err
	}
	runtime.postings = postings
	if err := runtime.reconcileCompact(expectedPostings, memberships); err != nil {
		return nil, err
	}
	return runtime, nil
}

func parseDictionary(data []byte, expected uint32) ([]string, error) {
	if len(data) < 4 || binary.BigEndian.Uint32(data[:4]) != expected {
		return nil, fmt.Errorf("%w: compact dictionary count", ErrInvalid)
	}
	offset := 4
	result := make([]string, 0, expected)
	for index := uint32(0); index < expected; index++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("%w: compact dictionary truncation", ErrInvalid)
		}
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if length < 1 || length > maxPostingKey || offset+length > len(data) {
			return nil, fmt.Errorf("%w: compact dictionary value bounds", ErrInvalid)
		}
		value := string(data[offset : offset+length])
		if len(result) > 0 && result[len(result)-1] >= value {
			return nil, fmt.Errorf("%w: compact dictionary order", ErrInvalid)
		}
		result = append(result, value)
		offset += length
	}
	if offset != len(data) {
		return nil, fmt.Errorf("%w: compact dictionary trailing bytes", ErrInvalid)
	}
	return result, nil
}

func (runtime *CompactRuntime) validateEntries(postingOffset int) (map[string][]uint32, error) {
	var prior dataplane.Digest
	expectedBody := runtime.bodyOffset
	expectedPostings := make(map[string][]uint32)
	for id := uint32(0); id < runtime.frameCount; id++ {
		offset := runtime.entryOffset + int(id)*compactEntry
		entry := runtime.data[offset : offset+compactEntry]
		var digest dataplane.Digest
		copy(digest[:], entry[:32])
		bodyOffset := int(binary.BigEndian.Uint64(entry[32:40]))
		bodyLength := int(binary.BigEndian.Uint32(entry[40:44]))
		universe := binary.BigEndian.Uint32(entry[44:48])
		frameType := binary.BigEndian.Uint32(entry[48:52])
		if (id > 0 && bytes.Compare(prior[:], digest[:]) >= 0) || bodyOffset != expectedBody || bodyLength < 1 || bodyLength > dataplane.MaxDocumentBytes || bodyOffset+bodyLength > postingOffset || int(universe) >= len(runtime.dictionary) || int(frameType) >= len(runtime.dictionary) {
			return nil, fmt.Errorf("%w: compact entry %d bounds", ErrInvalid, id)
		}
		body := runtime.data[bodyOffset : bodyOffset+bodyLength]
		frame, err := dataplane.UnmarshalFrame(body)
		if err != nil || dataplane.DigestBytes(body) != digest || frame.UniverseID != runtime.dictionary[universe] || frame.FrameType != runtime.dictionary[frameType] {
			return nil, fmt.Errorf("%w: compact entry %d reconciliation", ErrInvalid, id)
		}
		keys, err := framePostingKeys(frame)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			expectedPostings[key] = append(expectedPostings[key], id)
		}
		expectedBody += bodyLength
		prior = digest
	}
	if expectedBody != postingOffset {
		return nil, fmt.Errorf("%w: compact frame body gap", ErrInvalid)
	}
	return expectedPostings, nil
}

func parseCompactPostings(data []byte, start int, count, frames uint32) ([]compactPosting, uint64, error) {
	const minimumPostingBytes = 14 // key length + key + counts + one varint
	if start < 0 || start > len(data) || count == 0 || uint64(count) > uint64((len(data)-start)/minimumPostingBytes) {
		return nil, 0, fmt.Errorf("%w: compact posting count exceeds section", ErrInvalid)
	}
	offset := start
	result := make([]compactPosting, 0, count)
	var prior []byte
	var memberships uint64
	for index := uint32(0); index < count; index++ {
		if offset+4 > len(data) {
			return nil, 0, fmt.Errorf("%w: compact posting truncation", ErrInvalid)
		}
		keyLength := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if keyLength < 1 || keyLength > maxPostingKey || offset+keyLength+8 > len(data) {
			return nil, 0, fmt.Errorf("%w: compact posting key bounds", ErrInvalid)
		}
		keyOffset := offset
		key := data[offset : offset+keyLength]
		if index > 0 && bytes.Compare(prior, key) >= 0 {
			return nil, 0, fmt.Errorf("%w: compact posting key order", ErrInvalid)
		}
		offset += keyLength
		entryCount := binary.BigEndian.Uint32(data[offset : offset+4])
		payloadLength := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if entryCount == 0 || entryCount > frames || payloadLength < 1 || offset+payloadLength > len(data) {
			return nil, 0, fmt.Errorf("%w: compact posting payload bounds", ErrInvalid)
		}
		posting := compactPosting{keyOffset: keyOffset, keyLength: keyLength, payloadOffset: offset, payloadLength: payloadLength, count: entryCount}
		if _, err := decodePosting(data, posting, frames); err != nil {
			return nil, 0, err
		}
		result = append(result, posting)
		memberships += uint64(entryCount)
		prior = key
		offset += payloadLength
	}
	if offset != len(data) {
		return nil, 0, fmt.Errorf("%w: compact posting trailing bytes", ErrInvalid)
	}
	return result, memberships, nil
}

func decodePosting(data []byte, posting compactPosting, maximum uint32) ([]uint32, error) {
	payload := data[posting.payloadOffset : posting.payloadOffset+posting.payloadLength]
	result := make([]uint32, 0, posting.count)
	var prior uint32
	for index := uint32(0); index < posting.count; index++ {
		delta, length := binary.Uvarint(payload)
		if length <= 0 || delta > uint64(maximum) || index > 0 && delta == 0 {
			return nil, fmt.Errorf("%w: compact posting delta", ErrInvalid)
		}
		payload = payload[length:]
		value := uint64(prior) + delta
		if value >= uint64(maximum) || index > 0 && uint32(value) <= prior {
			return nil, fmt.Errorf("%w: compact posting frame ID", ErrInvalid)
		}
		prior = uint32(value)
		result = append(result, prior)
	}
	if len(payload) != 0 {
		return nil, fmt.Errorf("%w: compact posting payload trailing bytes", ErrInvalid)
	}
	return result, nil
}

func (runtime *CompactRuntime) reconcileCompact(expected map[string][]uint32, memberships uint64) error {
	var expectedMemberships uint64
	if len(expected) != len(runtime.postings) {
		return fmt.Errorf("%w: compact posting count mismatch", ErrInvalid)
	}
	for _, posting := range runtime.postings {
		key := string(runtime.data[posting.keyOffset : posting.keyOffset+posting.keyLength])
		expectedIDs, exists := expected[key]
		if !exists {
			return fmt.Errorf("%w: compact unexpected posting", ErrInvalid)
		}
		ids, err := decodePosting(runtime.data, posting, runtime.frameCount)
		if err != nil || !sameIDs(ids, expectedIDs) {
			return fmt.Errorf("%w: compact posting reconciliation", ErrInvalid)
		}
		expectedMemberships += uint64(len(expectedIDs))
	}
	if expectedMemberships != memberships {
		return fmt.Errorf("%w: compact posting membership mismatch", ErrInvalid)
	}
	return nil
}

func (runtime *CompactRuntime) Query(query Query) ([]dataplane.Digest, error) {
	if runtime == nil || runtime.data == nil {
		return nil, fmt.Errorf("%w: compact runtime is closed", ErrInvalid)
	}
	keys, err := queryKeys(query)
	if err != nil {
		return nil, err
	}
	sets := make([][]uint32, 0, len(keys))
	for _, key := range keys {
		posting, ok := runtime.findPosting(key)
		if !ok {
			return []dataplane.Digest{}, nil
		}
		ids, err := decodePosting(runtime.data, posting, runtime.frameCount)
		if err != nil {
			return nil, err
		}
		sets = append(sets, ids)
	}
	ids := intersect(sets)
	result := make([]dataplane.Digest, 0, len(ids))
	for _, id := range ids {
		_, digest, err := runtime.frameBody(id)
		if err != nil {
			return nil, err
		}
		result = append(result, digest)
		if len(result) >= int(query.Limit) {
			break
		}
	}
	return result, nil
}

// VisitFrames walks every admitted frame in canonical digest order. It is an
// integrity-verification primitive, not an unbounded public query: callers
// receive a copy of each canonical frame body and cannot mutate the mapped
// segment. The walk stops at the first visitor error.
func (runtime *CompactRuntime) VisitFrames(visitor func(dataplane.Digest, []byte) error) error {
	if runtime == nil || runtime.data == nil || visitor == nil {
		return fmt.Errorf("%w: compact frame visitor", ErrInvalid)
	}
	for id := uint32(0); id < runtime.frameCount; id++ {
		body, digest, err := runtime.frameBody(id)
		if err != nil {
			return err
		}
		if err := visitor(digest, append([]byte(nil), body...)); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *CompactRuntime) Trace(digest dataplane.Digest) ([]byte, error) {
	if runtime == nil || runtime.data == nil {
		return nil, fmt.Errorf("%w: compact runtime is closed", ErrInvalid)
	}
	index := sort.Search(int(runtime.frameCount), func(index int) bool {
		offset := runtime.entryOffset + index*compactEntry
		return bytes.Compare(runtime.data[offset:offset+32], digest[:]) >= 0
	})
	if index >= int(runtime.frameCount) {
		return nil, fmt.Errorf("%w: frame not found", ErrInvalid)
	}
	body, found, err := runtime.frameBody(uint32(index))
	if err != nil || found != digest {
		return nil, fmt.Errorf("%w: frame not found", ErrInvalid)
	}
	return append([]byte(nil), body...), nil
}

func (runtime *CompactRuntime) frameBody(id uint32) ([]byte, dataplane.Digest, error) {
	var digest dataplane.Digest
	if runtime == nil || runtime.data == nil || id >= runtime.frameCount {
		return nil, digest, fmt.Errorf("%w: compact frame ID", ErrInvalid)
	}
	offset := runtime.entryOffset + int(id)*compactEntry
	entry := runtime.data[offset : offset+compactEntry]
	copy(digest[:], entry[:32])
	bodyOffset := int(binary.BigEndian.Uint64(entry[32:40]))
	bodyLength := int(binary.BigEndian.Uint32(entry[40:44]))
	return runtime.data[bodyOffset : bodyOffset+bodyLength], digest, nil
}

func (runtime *CompactRuntime) findPosting(key string) (compactPosting, bool) {
	needle := []byte(key)
	index := sort.Search(len(runtime.postings), func(index int) bool {
		posting := runtime.postings[index]
		return bytes.Compare(runtime.data[posting.keyOffset:posting.keyOffset+posting.keyLength], needle) >= 0
	})
	if index >= len(runtime.postings) {
		return compactPosting{}, false
	}
	posting := runtime.postings[index]
	return posting, bytes.Equal(runtime.data[posting.keyOffset:posting.keyOffset+posting.keyLength], needle)
}

func containsID(values []uint32, target uint32) bool {
	index := sort.Search(len(values), func(index int) bool { return values[index] >= target })
	return index < len(values) && values[index] == target
}

func putU32(buffer *bytes.Buffer, value uint32) {
	var data [4]byte
	binary.BigEndian.PutUint32(data[:], value)
	buffer.Write(data[:])
}

func putU64(buffer *bytes.Buffer, value uint64) {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], value)
	buffer.Write(data[:])
}
