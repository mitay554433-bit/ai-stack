package runtime

import (
	"context"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/gov"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/reg"
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

type lineageReasoner struct {
	result reason.Result
}

func (l lineageReasoner) Analyze(context.Context, reason.Input) (reason.Result, error) {
	return l.result, nil
}

func (lineageReasoner) Name() string {
	return "lineage-test"
}

func (lineageReasoner) Version(context.Context) string {
	return "1"
}

func TestCaptureRejectsUnacceptedSupersedes(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(root, "candidate.txt")
	if err := os.WriteFile(source, []byte("candidate source"), 0600); err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store: s,
		Reasoner: lineageReasoner{
			result: reason.Result{
				Summary:       "candidate",
				Relationships: map[string]string{"source_name": "candidate.txt"},
				Capabilities:  []string{"OBS"},
				Facts:         []string{"source_preserved"},
				Risk:          "L",
				Supersedes:    "E-NOT-ACCEPTED",
				Delta:         []string{"changed behavior"},
			},
		},
	}

	_, _, err = r.Capture(context.Background(), source, false)
	if err == nil {
		t.Fatal("expected unaccepted supersedes lineage to be rejected")
	}

	events, eventErr := s.Events()
	if eventErr != nil {
		t.Fatal(eventErr)
	}
	if len(events) != 0 {
		t.Fatalf("rejected lineage wrote events: %d", len(events))
	}
}

func TestCaptureAcceptsREGAcceptedSupersedes(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	predecessor := core.EmergION{
		IDN: "E-ACCEPTED-PREV",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "accepted-source-hash",
			Bytes:      1,
			Stored:     1,
		},
		VAL: core.Validation{
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{Version: 1},
	}

	if _, err := s.SaveCandidate(predecessor); err != nil {
		t.Fatal(err)
	}

	approved, decision, err := gov.Decide(
		predecessor,
		gov.Approve,
		"HUMAN_FINAL",
		"test predecessor approval",
	)
	if err != nil {
		t.Fatal(err)
	}

	decisionID, err := s.SaveDecision(decision)
	if err != nil {
		t.Fatal(err)
	}

	accepted, receipt, err := reg.Accept(approved, decisionID)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.STA != core.StateAccepted {
		t.Fatalf("predecessor was not REG accepted: %s", accepted.STA)
	}

	if _, err := s.SaveAccepted(receipt); err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(root, "successor.txt")
	if err := os.WriteFile(source, []byte("successor source"), 0600); err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store: s,
		Reasoner: lineageReasoner{
			result: reason.Result{
				Summary: "successor",
				Relationships: map[string]string{
					"source_name":    "successor.txt",
					"governed_state": "accepted_context_present",
				},
				Capabilities: []string{"OBS"},
				Facts: []string{
					"source_preserved",
					"living_state_projected",
				},
				Risk:       "L",
				Supersedes: predecessor.IDN,
				Delta:      []string{"changed behavior"},
			},
		},
	}

	em, duplicate, err := r.Capture(context.Background(), source, false)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("successor unexpectedly treated as duplicate")
	}

	if em.EVO.Supersedes != predecessor.IDN {
		t.Fatalf(
			"supersedes lost: got %q want %q",
			em.EVO.Supersedes,
			predecessor.IDN,
		)
	}

	expectedDelta := []string{
		"SUMMARY_CHANGED",
		"CAP_ADDED:OBS",
		"REL_ADDED:governed_state",
		"REL_ADDED:source_name",
	}
	if len(em.EVO.Delta) != len(expectedDelta) {
		t.Fatalf("unexpected deterministic delta: %#v", em.EVO.Delta)
	}
	for i, want := range expectedDelta {
		if em.EVO.Delta[i] != want {
			t.Fatalf("delta[%d] = %q want %q; full=%#v", i, em.EVO.Delta[i], want, em.EVO.Delta)
		}
	}
	for _, item := range em.EVO.Delta {
		if item == "changed behavior" {
			t.Fatalf("model-supplied delta crossed runtime boundary: %#v", em.EVO.Delta)
		}
	}

	if em.STA != core.StateAtGOV {
		t.Fatalf("successor not delivered to GOV: %s", em.STA)
	}
}
