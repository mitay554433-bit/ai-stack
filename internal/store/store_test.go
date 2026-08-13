package store

import (
	"emergion-sovereign-runtime/internal/core"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.Preserve([]byte("abc"))
	if err != nil {
		t.Fatal(err)
	}
	em := core.EmergION{IDN: "E-1", STA: core.StateAtGOV, MEM: core.Memory{SourceHash: ev.Hash, Codec: ev.Codec, Bytes: ev.Bytes, Stored: ev.Stored}, VAL: core.Validation{Recoil: true, WVC: true}, EVO: core.Evolution{Version: 1}}
	if _, err = s.SaveCandidate(em); err != nil {
		t.Fatal(err)
	}
	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d", len(events))
	}
	b, err := s.ReadEvidence(ev.Hash)
	if err != nil || string(b) != "abc" {
		t.Fatalf("bad evidence %q %v", b, err)
	}
}

func TestSaveCandidateRejectsInvalidMetadata(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	em := core.EmergION{
		IDN: "E-1", STA: core.StateAtGOV,
		VAL: core.Validation{Recoil: true, WVC: true},
		EVO: core.Evolution{Version: 1, Metadata: &core.Metadata{
			CapturedAt: time.Now().UTC(),
			BuildEdges: []core.BuildEdge{{From: "missing", To: "also-missing"}},
		}},
	}
	if _, err = s.SaveCandidate(em); err == nil {
		t.Fatal("expected invalid metadata to fail")
	}
}

func TestInspectEvidenceReportsPreservedObject(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("inspect this preserved evidence")

	preserved, err := s.Preserve(content)
	if err != nil {
		t.Fatal(err)
	}

	observed, err := s.InspectEvidence(preserved.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if observed.Hash != preserved.Hash {
		t.Fatalf("hash = %q want %q", observed.Hash, preserved.Hash)
	}

	if observed.Bytes != int64(len(content)) {
		t.Fatalf(
			"bytes = %d want %d",
			observed.Bytes,
			len(content),
		)
	}

	if observed.Stored != preserved.Stored {
		t.Fatalf(
			"stored = %d want %d",
			observed.Stored,
			preserved.Stored,
		)
	}

	if observed.Codec != "gzip" {
		t.Fatalf("codec = %q", observed.Codec)
	}
}
