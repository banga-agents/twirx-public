package observation

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func FuzzUnmarshalCBOR(f *testing.F) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "conformance", "observation", "vectors.json"))
	if err != nil {
		f.Fatal(err)
	}
	var corpus observationCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		f.Fatal(err)
	}
	for _, vector := range corpus.Vectors {
		seed, err := hex.DecodeString(vector.CBORHex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		env, err := UnmarshalCBOR(data)
		if err != nil {
			return
		}
		reencoded, err := env.MarshalCBOR()
		if err != nil {
			t.Fatalf("accepted envelope cannot be encoded: %v", err)
		}
		if !bytes.Equal(data, reencoded) {
			t.Fatal("accepted envelope is not in its unique canonical encoding")
		}
	})
}
