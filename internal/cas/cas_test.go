package cas

import (
	"os"
	"testing"
)

func TestPutRead(t *testing.T) {
	store := New(t.TempDir())
	data := []byte("typed web evidence")
	digest, path, err := store.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	got, err := store.Read(digest, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("got %q", got)
	}
}
