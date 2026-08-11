package archiveprofile

import (
	"bytes"
	"testing"
)

func TestExtractTitlePreservesNativeLexicalValue(t *testing.T) {
	body := []byte("<!doctype html><HTML><head data-x='1'><title> &raquo; RFC Editor</title></head><body></body></HTML>")
	statement, err := ExtractTitle(body)
	if err != nil {
		t.Fatal(err)
	}
	if statement.NativeLexical != " &raquo; RFC Editor" || statement.Locator != Locator {
		t.Fatalf("native lexical value changed: %#v", statement)
	}
}

func TestExtractTitleRejectsMalformedAndUnboundedInputs(t *testing.T) {
	for _, body := range [][]byte{
		nil,
		[]byte("<head><title></title></head>"),
		[]byte("<head><title><b>mapped</b></title></head>"),
		[]byte("<body><title>outside</title></body>"),
		[]byte("<head><title>unterminated</head>"),
		bytes.Repeat([]byte("a"), MaxBody+1),
		{0xff},
	} {
		if _, err := ExtractTitle(body); err == nil {
			t.Fatalf("malformed body accepted: %q", body)
		}
	}
}

func TestExtractTitleIgnoresCommentAttributeAndScriptDecoys(t *testing.T) {
	body := []byte(`<html><!-- <head><title>comment</title></head> --><head><meta content="<title>attribute</title>"><script>"<title>script</title>"</script><title>real &amp; native</title></head></html>`)
	statement, err := ExtractTitle(body)
	if err != nil {
		t.Fatal(err)
	}
	if statement.NativeLexical != "real &amp; native" {
		t.Fatalf("decoy became native statement: %#v", statement)
	}
}

func FuzzExtractTitle(f *testing.F) {
	f.Add([]byte("<html><head><title>RFC Editor</title></head></html>"))
	f.Add([]byte("<head><title><b>x</b></title></head>"))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = ExtractTitle(data) })
}
