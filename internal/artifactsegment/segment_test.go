package artifactsegment

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestBuildOpenFindAndRejectMutations(t *testing.T) {
	one := []byte{0x81, 0x01}
	two := []byte{0x81, 0x02}
	data, digest, err := Build(KindPacket, []Entry{{Digest: dataplane.DigestBytes(two), CBOR: two}, {Digest: dataplane.DigestBytes(one), CBOR: one}})
	if err != nil {
		t.Fatal(err)
	}
	segment, err := Open(data, digest)
	if err != nil || segment.Count() != 2 || segment.Kind() != KindPacket {
		t.Fatalf("open: segment=%+v err=%v", segment, err)
	}
	if body, ok := segment.Find(dataplane.DigestBytes(two)); !ok || !bytes.Equal(body, two) {
		t.Fatal("entry lookup failed")
	}
	mutations := []func([]byte){
		func(value []byte) { value[0] ^= 1 },
		func(value []byte) { binary.BigEndian.PutUint32(value[8:12], 99) },
		func(value []byte) { binary.BigEndian.PutUint32(value[44:48], 1) },
		func(value []byte) { value[len(value)-1] ^= 1 },
	}
	for index, mutate := range mutations {
		candidate := append([]byte(nil), data...)
		mutate(candidate)
		if _, err := Open(candidate, dataplane.DigestBytes(candidate)); err == nil {
			t.Fatalf("mutation %d was admitted", index)
		}
	}
}

func TestBuildRejectsDuplicateOrMismatchedEntries(t *testing.T) {
	body := []byte{0x81, 0x01}
	digest := dataplane.DigestBytes(body)
	if _, _, err := Build(KindPacket, []Entry{{Digest: digest, CBOR: body}, {Digest: digest, CBOR: body}}); err == nil {
		t.Fatal("duplicate digest admitted")
	}
	if _, _, err := Build(KindPacket, []Entry{{Digest: dataplane.DigestBytes([]byte("wrong")), CBOR: body}}); err == nil {
		t.Fatal("mismatched digest admitted")
	}
}
