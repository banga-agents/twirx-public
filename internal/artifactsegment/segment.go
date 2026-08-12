// Package artifactsegment stores independently canonical CBOR documents in a
// deterministic, content-addressed, read-only segment. The segment is a
// publication container only; each entry retains its own SHA-256 identity.
package artifactsegment

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

const (
	Format           = "tw.artifact-segment/0.1"
	Magic            = "TWAS0001"
	HeaderBytes      = 40
	EntryBytes       = 48
	MaxEntries       = 65536
	MaxSegmentBytes  = 256 << 20
	MaxArtifactBytes = 4 << 20
)

const (
	KindPacket  uint32 = 1
	KindMapping uint32 = 2
)

var ErrInvalid = errors.New("artifactsegment: invalid segment")

type Entry struct {
	Digest dataplane.Digest
	CBOR   []byte
}

type Segment struct {
	data      []byte
	kind      uint32
	count     uint32
	indexBase int
	bodyBase  int
}

func Build(kind uint32, source []Entry) ([]byte, dataplane.Digest, error) {
	if !validKind(kind) || len(source) == 0 || len(source) > MaxEntries {
		return nil, dataplane.Digest{}, fmt.Errorf("%w: kind or entry count", ErrInvalid)
	}
	entries := make([]Entry, len(source))
	for index, entry := range source {
		if len(entry.CBOR) == 0 || len(entry.CBOR) > MaxArtifactBytes || dataplane.DigestBytes(entry.CBOR) != entry.Digest {
			return nil, dataplane.Digest{}, fmt.Errorf("%w: entry %d identity", ErrInvalid, index)
		}
		entries[index] = Entry{Digest: entry.Digest, CBOR: append([]byte(nil), entry.CBOR...)}
	}
	sort.Slice(entries, func(i, j int) bool { return bytes.Compare(entries[i].Digest[:], entries[j].Digest[:]) < 0 })
	for index := 1; index < len(entries); index++ {
		if entries[index-1].Digest == entries[index].Digest {
			return nil, dataplane.Digest{}, fmt.Errorf("%w: duplicate entry digest", ErrInvalid)
		}
	}
	bodyBase := HeaderBytes + EntryBytes*len(entries)
	total := bodyBase
	for _, entry := range entries {
		if total > MaxSegmentBytes-len(entry.CBOR) {
			return nil, dataplane.Digest{}, fmt.Errorf("%w: segment byte bound", ErrInvalid)
		}
		total += len(entry.CBOR)
	}
	result := make([]byte, total)
	copy(result[:8], Magic)
	binary.BigEndian.PutUint32(result[8:12], kind)
	binary.BigEndian.PutUint32(result[12:16], uint32(len(entries)))
	binary.BigEndian.PutUint64(result[16:24], HeaderBytes)
	binary.BigEndian.PutUint64(result[24:32], uint64(bodyBase))
	binary.BigEndian.PutUint64(result[32:40], uint64(total))
	offset := bodyBase
	for index, entry := range entries {
		position := HeaderBytes + index*EntryBytes
		copy(result[position:position+32], entry.Digest[:])
		binary.BigEndian.PutUint64(result[position+32:position+40], uint64(offset))
		binary.BigEndian.PutUint32(result[position+40:position+44], uint32(len(entry.CBOR)))
		copy(result[offset:], entry.CBOR)
		offset += len(entry.CBOR)
	}
	return result, dataplane.DigestBytes(result), nil
}

func Open(data []byte, expected dataplane.Digest) (*Segment, error) {
	if len(data) < HeaderBytes || len(data) > MaxSegmentBytes || dataplane.DigestBytes(data) != expected || string(data[:8]) != Magic {
		return nil, fmt.Errorf("%w: segment identity", ErrInvalid)
	}
	kind := binary.BigEndian.Uint32(data[8:12])
	count := binary.BigEndian.Uint32(data[12:16])
	indexBase := binary.BigEndian.Uint64(data[16:24])
	bodyBase := binary.BigEndian.Uint64(data[24:32])
	end := binary.BigEndian.Uint64(data[32:40])
	if !validKind(kind) || count == 0 || count > MaxEntries || indexBase != HeaderBytes || bodyBase != HeaderBytes+uint64(count)*EntryBytes || end != uint64(len(data)) || bodyBase >= end {
		return nil, fmt.Errorf("%w: header", ErrInvalid)
	}
	segment := &Segment{data: data, kind: kind, count: count, indexBase: int(indexBase), bodyBase: int(bodyBase)}
	if err := segment.verifyEntries(); err != nil {
		return nil, err
	}
	return segment, nil
}

func (segment *Segment) Kind() uint32  { return segment.kind }
func (segment *Segment) Count() uint32 { return segment.count }

func (segment *Segment) Entry(index uint32) (Entry, error) {
	if segment == nil || index >= segment.count {
		return Entry{}, fmt.Errorf("%w: entry index", ErrInvalid)
	}
	position := segment.indexBase + int(index)*EntryBytes
	var digest dataplane.Digest
	copy(digest[:], segment.data[position:position+32])
	offset := int(binary.BigEndian.Uint64(segment.data[position+32 : position+40]))
	length := int(binary.BigEndian.Uint32(segment.data[position+40 : position+44]))
	return Entry{Digest: digest, CBOR: segment.data[offset : offset+length]}, nil
}

func (segment *Segment) Find(digest dataplane.Digest) ([]byte, bool) {
	if segment == nil {
		return nil, false
	}
	index := sort.Search(int(segment.count), func(index int) bool {
		position := segment.indexBase + index*EntryBytes
		return bytes.Compare(segment.data[position:position+32], digest[:]) >= 0
	})
	if index == int(segment.count) {
		return nil, false
	}
	entry, err := segment.Entry(uint32(index))
	if err != nil || entry.Digest != digest {
		return nil, false
	}
	return entry.CBOR, true
}

func (segment *Segment) verifyEntries() error {
	expectedOffset := segment.bodyBase
	var prior dataplane.Digest
	for index := uint32(0); index < segment.count; index++ {
		position := segment.indexBase + int(index)*EntryBytes
		var digest dataplane.Digest
		copy(digest[:], segment.data[position:position+32])
		offset := binary.BigEndian.Uint64(segment.data[position+32 : position+40])
		length := binary.BigEndian.Uint32(segment.data[position+40 : position+44])
		reserved := binary.BigEndian.Uint32(segment.data[position+44 : position+48])
		if index > 0 && bytes.Compare(prior[:], digest[:]) >= 0 || offset != uint64(expectedOffset) || length == 0 || length > MaxArtifactBytes || reserved != 0 || offset+uint64(length) > uint64(len(segment.data)) {
			return fmt.Errorf("%w: entry %d bounds", ErrInvalid, index)
		}
		body := segment.data[offset : offset+uint64(length)]
		if dataplane.DigestBytes(body) != digest {
			return fmt.Errorf("%w: entry %d digest", ErrInvalid, index)
		}
		expectedOffset += int(length)
		prior = digest
	}
	if expectedOffset != len(segment.data) {
		return fmt.Errorf("%w: trailing bytes", ErrInvalid)
	}
	return nil
}

func validKind(kind uint32) bool { return kind == KindPacket || kind == KindMapping }
