package runtime

import (
	"context"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/store"
	"os"
	"path/filepath"
	"testing"
)

func TestOnceClearsDropzone(t *testing.T) {
	root := t.TempDir()
	s, _ := store.Open(filepath.Join(root, "state"))
	dz := filepath.Join(root, "drop")
	os.MkdirAll(dz, 0700)
	os.WriteFile(filepath.Join(dz, "a.txt"), []byte("hello"), 0600)
	r := Runtime{Store: s, Reasoner: reason.Heuristic{}}
	ids, err := r.Once(context.Background(), dz)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatal(ids)
	}
	ents, _ := os.ReadDir(dz)
	if len(ents) != 0 {
		t.Fatalf("dropzone not clear")
	}
}
