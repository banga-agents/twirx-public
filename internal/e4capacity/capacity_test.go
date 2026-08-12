package e4capacity

import (
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
	"github.com/typed-web-commons/typed-web/internal/universesnapshot"
)

func TestControlledFramesAreDeterministicAndQueryable(t *testing.T) {
	first, err := ControlledFrames(100)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ControlledFrames(100)
	if err != nil {
		t.Fatal(err)
	}
	data, digest, err := universesnapshot.BuildCompact(first)
	if err != nil {
		t.Fatal(err)
	}
	repeated, repeatedDigest, err := universesnapshot.BuildCompact(second)
	if err != nil || digest != repeatedDigest || string(data) != string(repeated) {
		t.Fatal("controlled compact release is not deterministic")
	}
	runtime, err := universesnapshot.OpenCompact(data, digest)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Query(universesnapshot.Query{UniverseID: UniverseID, FrameType: FrameType, SlotRole: "world:country", SlotValue: &dataplane.TypedValue{Type: "identifier", Lexical: Country(42)}, Limit: 1})
	if err != nil || len(result) != 1 {
		t.Fatalf("controlled query = %x, %v", result, err)
	}
}

func TestControlledFramesRejectOutOfBounds(t *testing.T) {
	for _, count := range []int{0, universesnapshot.MaxFrames + 1} {
		if _, err := ControlledFrames(count); err == nil {
			t.Fatalf("accepted count %d", count)
		}
	}
}
