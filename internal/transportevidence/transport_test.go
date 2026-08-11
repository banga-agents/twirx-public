package transportevidence

import (
	"testing"

	"github.com/typed-web-commons/typed-web/internal/safefetch"
)

func validEvidence() *Evidence {
	return &Evidence{Version: Version, RequestURL: "https://example.test/start", FinalURL: "https://example.test/final", PolicyID: "test", Redirects: []safefetch.Redirect{{FromURL: "https://example.test/start", Status: 302, ToURL: "https://example.test/final"}}, Headers: []safefetch.Header{{Name: "content-language", Value: "en"}, {Name: "content-type", Value: "application/json"}}}
}

func TestRoundTrip(t *testing.T) {
	encoded, err := validEvidence().MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalCBOR(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Redirects) != 1 || len(decoded.Headers) != 2 {
		t.Fatalf("unexpected %#v", decoded)
	}
}

func TestRejectsSensitiveAndBrokenEvidence(t *testing.T) {
	tests := []func(*Evidence){func(e *Evidence) {
		e.Headers = append(e.Headers, safefetch.Header{Name: "set-cookie", Value: "private"})
	}, func(e *Evidence) { e.Redirects[0].Status = 200 }, func(e *Evidence) { e.Redirects[0].ToURL = "https://evil.test/" }, func(e *Evidence) { e.Headers[0], e.Headers[1] = e.Headers[1], e.Headers[0] }}
	for _, mutate := range tests {
		e := validEvidence()
		mutate(e)
		if _, err := e.MarshalCBOR(); err == nil {
			t.Fatal("invalid evidence accepted")
		}
	}
}

func FuzzUnmarshalTransport(f *testing.F) {
	encoded, err := validEvidence().MarshalCBOR()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(encoded)
	f.Add([]byte{0x9f, 0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		e, decodeErr := UnmarshalCBOR(data)
		if decodeErr == nil {
			reencoded, encodeErr := e.MarshalCBOR()
			if encodeErr != nil || string(reencoded) != string(data) {
				t.Fatal("accepted transport was not canonical")
			}
		}
	})
}
