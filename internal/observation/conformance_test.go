package observation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/typed-web-commons/typed-web/internal/cas"
)

type observationCorpus struct {
	Format     string              `json:"format"`
	BodyFile   string              `json:"body_file"`
	BodySHA256 string              `json:"body_sha256"`
	Vectors    []observationVector `json:"vectors"`
}

type observationVector struct {
	ID        string `json:"id"`
	Expected  string `json:"expected"`
	Invariant string `json:"invariant"`
	CBORHex   string `json:"cbor_hex"`
}

func loadObservationCorpus(t testing.TB) (string, observationCorpus, []byte) {
	t.Helper()
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "conformance", "observation", "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus observationCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(corpus.BodyFile)))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != corpus.BodySHA256 {
		t.Fatalf("conformance body digest does not match manifest")
	}
	return root, corpus, body
}

func TestPublicObservationVectors(t *testing.T) {
	_, corpus, body := loadObservationCorpus(t)
	if corpus.Format != "tw.conformance-observation/0.1" || len(corpus.Vectors) == 0 {
		t.Fatal("invalid observation corpus metadata")
	}
	for _, vector := range corpus.Vectors {
		vector := vector
		t.Run(vector.ID, func(t *testing.T) {
			encoded, err := hex.DecodeString(vector.CBORHex)
			if err != nil {
				t.Fatal(err)
			}
			env, decodeErr := UnmarshalCBOR(encoded)
			if vector.Expected == "reject_envelope" {
				if decodeErr == nil {
					t.Fatalf("vector accepted; invariant=%s", vector.Invariant)
				}
				return
			}
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			store := cas.New(filepath.Join(t.TempDir(), "cas"))
			if _, _, err := store.Put(body); err != nil {
				t.Fatal(err)
			}
			verifyErr := VerifyBody(env, store, MaxBodyBytes)
			switch vector.Expected {
			case "accept":
				if verifyErr != nil {
					t.Fatal(verifyErr)
				}
			case "reject_evidence":
				if verifyErr == nil {
					t.Fatalf("evidence accepted; invariant=%s", vector.Invariant)
				}
			default:
				t.Fatalf("unknown expected state %q", vector.Expected)
			}
		})
	}
}

func TestVerifyBodyRejectsCorruptedCAS(t *testing.T) {
	_, corpus, body := loadObservationCorpus(t)
	encoded, err := hex.DecodeString(corpus.Vectors[0].CBORHex)
	if err != nil {
		t.Fatal(err)
	}
	env, err := UnmarshalCBOR(encoded)
	if err != nil {
		t.Fatal(err)
	}
	store := cas.New(filepath.Join(t.TempDir(), "cas"))
	_, path, err := store.Put(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("corrupted\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBody(env, store, MaxBodyBytes); err == nil {
		t.Fatal("corrupted CAS evidence was accepted")
	}
}
