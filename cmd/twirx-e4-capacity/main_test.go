package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestBuildAndOpen(t *testing.T) {
	segment := filepath.Join(t.TempDir(), "controlled.twux")
	var build bytes.Buffer
	if err := run([]string{"build", "--frames", "100", "--segment", segment}, &build, &build); err != nil {
		t.Fatal(err)
	}
	const prefix = `"segment_digest": "`
	start := bytes.Index(build.Bytes(), []byte(prefix))
	if start < 0 {
		t.Fatal("missing segment digest")
	}
	start += len(prefix)
	end := bytes.IndexByte(build.Bytes()[start:], '"')
	if end < 0 {
		t.Fatal("invalid segment digest")
	}
	digest := string(build.Bytes()[start : start+end])
	var opened bytes.Buffer
	if err := run([]string{"open", "--frames", "100", "--queries", "20", "--segment", segment, "--digest", digest}, &opened, &opened); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(opened.Bytes(), []byte(`"network_requests": 0`)) {
		t.Fatal("capacity report did not state zero network requests")
	}
}
