package labengine

import (
	"context"
	"path/filepath"
	"testing"
)

func TestViewPreservesNativeAndSemanticValues(t *testing.T) {
	engine, err := New(repositoryRoot(t), filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	invocation, err := engine.Invoke(context.Background(), Request{OriginID: "controlled-origin-lab", OperationID: "fixture.getOffer", Mode: ModeReplay, Input: map[string]string{"product_id": "demo-1"}})
	if err != nil {
		t.Fatal(err)
	}
	view := View(invocation)
	if view.Fields[3].Native.Lexical == nil || *view.Fields[3].Native.Lexical != "usd" || view.Fields[3].Semantic.Lexical == nil || *view.Fields[3].Semantic.Lexical != "USD" {
		t.Fatalf("source statement lost: %#v", view.Fields[3])
	}
	loaded, dir, err := engine.Load(view.ResultID)
	if err != nil || dir == "" || loaded.ResultID != view.ResultID {
		t.Fatalf("load: %#v %s %v", loaded, dir, err)
	}
}
