package runtime

import (
	"context"
	"emergion-sovereign-runtime/internal/core"
	livefield "emergion-sovereign-runtime/internal/field"
	"emergion-sovereign-runtime/internal/gov"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/reg"
	"emergion-sovereign-runtime/internal/store"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestGovernedStateContextRemainsValidJSON(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 40; i++ {
		id := fmt.Sprintf("E-PROJECTION-%02d", i)
		sourceHash := fmt.Sprintf("projection-source-%02d", i)

		em := core.EmergION{
			IDN: id,
			STA: core.StateAtGOV,
			MEM: core.Memory{
				SourceHash: sourceHash,
				Bytes:      1,
				Stored:     1,
				Summary:    strings.Repeat("projection-summary-", 40),
			},
			REL: map[string]string{
				"relationship": strings.Repeat("accepted-context-", 20),
			},
			CAP: []string{"OBS", "CMP"},
			VAL: core.Validation{
				Recoil: true,
				WVC:    true,
			},
			EVO: core.Evolution{Version: 1},
		}

		if _, err := s.SaveCandidate(em); err != nil {
			t.Fatal(err)
		}

		approved, decision, err := gov.Decide(
			em,
			gov.Approve,
			"HUMAN_FINAL",
			"projection test",
		)
		if err != nil {
			t.Fatal(err)
		}

		decisionID, err := s.SaveDecision(decision)
		if err != nil {
			t.Fatal(err)
		}

		_, receipt, err := reg.Accept(approved, decisionID)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := s.SaveAccepted(receipt); err != nil {
			t.Fatal(err)
		}
	}

	r := Runtime{Store: s, Reasoner: reason.Heuristic{}}

	projected, err := r.governedStateContext()
	if err != nil {
		t.Fatal(err)
	}

	if len(projected) > 12000 {
		t.Fatalf("projection exceeded bound: %d", len(projected))
	}

	var decoded []governedProjection
	if err := json.Unmarshal([]byte(projected), &decoded); err != nil {
		t.Fatalf("projection is invalid JSON: %v", err)
	}

	if len(decoded) == 0 {
		t.Fatal("bounded projection unexpectedly empty")
	}
}

func TestReworkRejectsNonReturnedPredecessor(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(root, "rework.txt")
	if err := os.WriteFile(source, []byte("corrected source"), 0600); err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store:               s,
		ReturnedPredecessor: "E-NOT-RETURNED",
		Reasoner: lineageReasoner{
			result: reason.Result{
				Summary:       "corrected source",
				Relationships: map[string]string{"source_name": "rework.txt"},
				Capabilities:  []string{"OBS"},
				Facts:         []string{"source_preserved"},
				Risk:          "L",
			},
		},
	}

	_, _, err = r.Capture(context.Background(), source, false)
	if err == nil {
		t.Fatal("expected non-returned predecessor to be rejected")
	}

	events, eventErr := s.Events()
	if eventErr != nil {
		t.Fatal(eventErr)
	}
	if len(events) != 0 {
		t.Fatalf("invalid rework wrote events: %d", len(events))
	}
}

func TestReturnedEmergIONReentersThroughRework(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	original := core.EmergION{
		IDN: "E-RETURNED-ORIGINAL",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "returned-original-hash",
			Bytes:      1,
			Stored:     1,
			Summary:    "original implementation",
		},
		REL: map[string]string{
			"source_name": "original.txt",
		},
		CAP: []string{"OBS"},
		VAL: core.Validation{
			Facts:  []string{"source_preserved"},
			Risk:   "M",
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Version: 1,
		},
	}

	if _, err := s.SaveCandidate(original); err != nil {
		t.Fatal(err)
	}

	_, returnedReceipt, err := gov.Decide(
		original,
		gov.Return,
		"HUMAN_FINAL",
		"correct and resubmit",
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.SaveDecision(returnedReceipt); err != nil {
		t.Fatal(err)
	}

	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}

	state, err := livefield.Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := state.Returned[original.IDN]; !ok {
		t.Fatal("original EmergION did not enter RETURNED state")
	}

	correctedPath := filepath.Join(root, "corrected.txt")
	if err := os.WriteFile(
		correctedPath,
		[]byte("corrected implementation"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store:               s,
		ReturnedPredecessor: original.IDN,
		Reasoner: lineageReasoner{
			result: reason.Result{
				Summary: "corrected implementation",
				Relationships: map[string]string{
					"source_name": "corrected.txt",
				},
				Capabilities: []string{"OBS", "CMP"},
				Facts:        []string{"source_preserved"},
				Risk:         "L",
			},
		},
	}

	successor, duplicate, err := r.Capture(
		context.Background(),
		correctedPath,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("corrected source unexpectedly treated as duplicate")
	}

	if successor.STA != core.StateAtGOV {
		t.Fatalf("reworked EmergION did not return to GOV: %s", successor.STA)
	}

	if successor.EVO.Supersedes != original.IDN {
		t.Fatalf(
			"rework lineage lost: got %q want %q",
			successor.EVO.Supersedes,
			original.IDN,
		)
	}

	if len(successor.EVO.Delta) == 0 {
		t.Fatal("reworked EmergION has no deterministic delta")
	}

	foundSummary := false
	foundCapability := false
	for _, item := range successor.EVO.Delta {
		switch item {
		case "SUMMARY_CHANGED":
			foundSummary = true
		case "CAP_ADDED:CMP":
			foundCapability = true
		}
	}

	if !foundSummary {
		t.Fatalf("summary delta missing: %#v", successor.EVO.Delta)
	}
	if !foundCapability {
		t.Fatalf("capability delta missing: %#v", successor.EVO.Delta)
	}
}
