package runtime

import (
	"context"
	"emergion-sovereign-runtime/internal/core"
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

func TestProtectorHumanFinalGate(t *testing.T) {
	em := core.EmergION{
		CAP: []string{"SEND", "TRANSFER", "DEPLOY"},
		REL: map[string]string{},
	}

	protector(&em)

	if em.REL["protector_gate"] != "HUMAN_FINAL_BOUND" {
		t.Fatalf("protector did not preserve HUMAN_FINAL boundary: %#v", em.REL)
	}

	if em.REL["protector"] == "" {
		t.Fatalf("protector authority envelope missing: %#v", em.REL)
	}
}
