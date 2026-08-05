package emerger

import (
	"context"
	"emergion-sovereign-runtime/internal/reason"
	"testing"
)

func TestEmerge(t *testing.T) {
	b := []byte("hello field")
	h := SourceHash(b)
	em, err := (Engine{Reasoner: reason.Heuristic{}}).Emerge(context.Background(), reason.Input{Name: "x", Content: b}, Evidence{Hash: h, Bytes: int64(len(b)), Codec: "gzip", Stored: 10})
	if err != nil {
		t.Fatal(err)
	}
	if em.STA != "G" || !em.VAL.Recoil || !em.VAL.WVC {
		t.Fatalf("bad: %#v", em)
	}
}
