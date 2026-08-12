package e4vectors

import (
	"testing"

	"github.com/typed-web-commons/typed-web/internal/dataplane"
)

func TestCorpusMatchesGoVerifier(t *testing.T) {
	vectors, err := Corpus()
	if err != nil {
		t.Fatal(err)
	}
	accepted := 0
	for _, vector := range vectors {
		err := dataplane.ValidateDocument(vector.Kind, vector.Data)
		if vector.Valid {
			accepted++
			if err != nil {
				t.Errorf("%s rejected: %v", vector.Name, err)
			}
		} else if err == nil {
			t.Errorf("%s accepted", vector.Name)
		}
	}
	if len(vectors) != 33 || accepted != 7 {
		t.Fatalf("unexpected corpus counts: total=%d accepted=%d", len(vectors), accepted)
	}
}
