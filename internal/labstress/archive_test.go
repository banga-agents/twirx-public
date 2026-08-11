package labstress

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/labapi"
	"github.com/typed-web-commons/typed-web/internal/labengine"
)

func archiveRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func validArchive(t *testing.T) ([]byte, labengine.ResultView) {
	t.Helper()
	root := archiveRoot(t)
	engine, err := labengine.New(root, filepath.Join(t.TempDir(), "results"))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := labapi.New(labapi.Config{Engine: engine, StaticDir: filepath.Join(root, "lab", "static"), PerIPPerMinute: 600, PerIPBurst: 100, AuditWriter: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	payload := `{"origin_id":"controlled-origin-lab","operation_id":"fixture.getOffer","mode":"replay","input":{"product_id":"demo-1"}}`
	response, err := http.Post(server.URL+"/api/v1/invoke", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var view labengine.ResultView
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&view) != nil {
		t.Fatalf("invoke failed: %d", response.StatusCode)
	}
	bundleResponse, err := http.Get(server.URL + "/api/v1/results/" + view.ResultID + "/bundle")
	if err != nil {
		t.Fatal(err)
	}
	defer bundleResponse.Body.Close()
	bundle, err := io.ReadAll(bundleResponse.Body)
	if err != nil || bundleResponse.StatusCode != http.StatusOK {
		t.Fatalf("bundle failed: %d %v", bundleResponse.StatusCode, err)
	}
	return bundle, view
}

func rewriteArchive(t *testing.T, data []byte, alter func(string, []byte) (string, []byte, bool)) []byte {
	t.Helper()
	input, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range input.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		name, body, keep := alter(file.Name, body)
		if !keep {
			continue
		}
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestVerifyArchiveRejectsMalformedAndSubstitutedBundles(t *testing.T) {
	bundle, view := validArchive(t)
	if err := verifyArchive(bundle, view); err != nil {
		t.Fatal(err)
	}
	tests := map[string][]byte{
		"empty":   nil,
		"not-zip": []byte("not a zip archive"),
		"missing-manifest": rewriteArchive(t, bundle, func(name string, body []byte) (string, []byte, bool) {
			return name, body, name != "manifest.cbor"
		}),
		"substituted-body": rewriteArchive(t, bundle, func(name string, body []byte) (string, []byte, bool) {
			if name == "representation.body" {
				body = append([]byte(nil), body...)
				body[0] ^= 1
			}
			return name, body, true
		}),
		"unsafe-name": rewriteArchive(t, bundle, func(name string, body []byte) (string, []byte, bool) {
			if name == "representation.body" {
				name = "../representation.body"
			}
			return name, body, true
		}),
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if err := verifyArchive(data, view); err == nil {
				t.Fatal("accepted malformed proof archive")
			}
		})
	}
}

func TestLoadWorkloadRejectsAmbiguousAndSymlinkedFiles(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.json")
	content := `{"format":"tw.lab-stress-workload/0.1","scenarios":[{"id":"s","origin_id":"o","operation_id":"op","input":{"x":"y"},"weight":1}]}`
	if err := os.WriteFile(valid, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWorkload(valid); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"duplicate.json": `{"format":"tw.lab-stress-workload/0.1","format":"tw.lab-stress-workload/0.1","scenarios":[]}`,
		"unknown.json":   `{"format":"tw.lab-stress-workload/0.1","unknown":true,"scenarios":[]}`,
		"trailing.json":  content + `{}`,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadWorkload(path); err == nil {
			t.Fatalf("accepted malformed workload %s", name)
		}
	}
	symlink := filepath.Join(dir, "link.json")
	if err := os.Symlink(valid, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWorkload(symlink); err == nil {
		t.Fatal("accepted symlinked workload")
	}
}
