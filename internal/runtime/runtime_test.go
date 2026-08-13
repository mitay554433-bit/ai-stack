package runtime

import (
	"context"
	"emergion-sovereign-runtime/internal/core"
	livefield "emergion-sovereign-runtime/internal/field"
	"emergion-sovereign-runtime/internal/gov"
	"emergion-sovereign-runtime/internal/pivot"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/reg"
	"emergion-sovereign-runtime/internal/store"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestFullGovernedEmergenceLoop(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	originalPath := filepath.Join(root, "original.txt")
	if err := os.WriteFile(originalPath, []byte("original implementation"), 0600); err != nil {
		t.Fatal(err)
	}

	initialReasoner := lineageReasoner{
		result: reason.Result{
			Summary:       "original implementation",
			Relationships: map[string]string{"source_name": "original.txt"},
			Capabilities:  []string{"OBS"},
			Facts:         []string{"source_preserved"},
			Risk:          "M",
		},
	}

	rt := Runtime{
		Store:    s,
		Reasoner: initialReasoner,
	}

	original, duplicate, err := rt.Capture(
		context.Background(),
		originalPath,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("original source unexpectedly duplicate")
	}
	if original.STA != core.StateAtGOV {
		t.Fatalf("original state = %s", original.STA)
	}

	_, returnReceipt, err := gov.Decide(
		original,
		gov.Return,
		"HUMAN_FINAL",
		"correct and resubmit",
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.SaveDecision(returnReceipt); err != nil {
		t.Fatal(err)
	}

	state, err := livefield.Rebuild(mustEvents(t, s))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Returned[original.IDN]; !ok {
		t.Fatal("original did not enter returned state")
	}

	correctedPath := filepath.Join(root, "corrected.txt")
	if err := os.WriteFile(correctedPath, []byte("corrected implementation"), 0600); err != nil {
		t.Fatal(err)
	}

	rework := Runtime{
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

	successor, duplicate, err := rework.Capture(
		context.Background(),
		correctedPath,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("corrected source unexpectedly duplicate")
	}
	if successor.STA != core.StateAtGOV {
		t.Fatalf("reworked state = %s", successor.STA)
	}
	if successor.EVO.Supersedes != original.IDN {
		t.Fatalf(
			"rework predecessor = %q want %q",
			successor.EVO.Supersedes,
			original.IDN,
		)
	}

	held, holdReceipt, err := gov.Decide(
		successor,
		gov.Hold,
		"HUMAN_FINAL",
		"pause for review",
	)
	if err != nil {
		t.Fatal(err)
	}
	if held.STA != core.StateHeld {
		t.Fatalf("hold state = %s", held.STA)
	}
	if _, err := s.SaveDecision(holdReceipt); err != nil {
		t.Fatal(err)
	}

	state, err = livefield.Rebuild(mustEvents(t, s))
	if err != nil {
		t.Fatal(err)
	}
	heldState, ok := state.Held[successor.IDN]
	if !ok {
		t.Fatal("successor did not enter held state")
	}

	resumed, resumeReceipt, err := gov.ResumeHeld(
		heldState,
		"HUMAN_FINAL",
		"resume review",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.STA != core.StateAtGOV {
		t.Fatalf("resume state = %s", resumed.STA)
	}
	if _, err := s.SaveDecision(resumeReceipt); err != nil {
		t.Fatal(err)
	}

	state, err = livefield.Rebuild(mustEvents(t, s))
	if err != nil {
		t.Fatal(err)
	}
	resumedState, ok := state.AtGOV[successor.IDN]
	if !ok {
		t.Fatal("resumed successor did not return to GOV")
	}

	approved, approveReceipt, err := gov.Decide(
		resumedState,
		gov.Approve,
		"HUMAN_FINAL",
		"approve corrected implementation",
	)
	if err != nil {
		t.Fatal(err)
	}

	decisionID, err := s.SaveDecision(approveReceipt)
	if err != nil {
		t.Fatal(err)
	}

	accepted, regReceipt, err := reg.Accept(approved, decisionID)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.STA != core.StateAccepted {
		t.Fatalf("accepted state = %s", accepted.STA)
	}

	if _, err := s.SaveAccepted(regReceipt); err != nil {
		t.Fatal(err)
	}

	state, err = livefield.Rebuild(mustEvents(t, s))
	if err != nil {
		t.Fatal(err)
	}

	final, ok := state.Accepted[successor.IDN]
	if !ok {
		t.Fatal("successor did not reach REG accepted state")
	}
	if final.STA != core.StateAccepted {
		t.Fatalf("final state = %s", final.STA)
	}
	if final.EVO.Supersedes != original.IDN {
		t.Fatalf(
			"final lineage = %q want %q",
			final.EVO.Supersedes,
			original.IDN,
		)
	}
	if len(final.EVO.Delta) == 0 {
		t.Fatal("final governed delta missing")
	}
}

func mustEvents(t *testing.T, s *store.Store) []core.Event {
	t.Helper()

	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestDeriveDeltaIgnoresRuntimeProtectorRelationships(t *testing.T) {
	previous := core.EmergION{
		MEM: core.Memory{
			Summary: "same summary",
		},
		REL: map[string]string{
			"source_name":    "before.txt",
			"protector":      "NO_EXTERNAL_AUTHORITY_CLAIMED",
			"protector_gate": "HUMAN_FINAL_BOUND",
		},
	}

	analysis := reason.Result{
		Summary: "same summary",
		Relationships: map[string]string{
			"source_name": "after.txt",
		},
	}

	got := deriveDelta(previous, analysis)

	want := []string{
		"REL_CHANGED:source_name",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("delta mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestPivotDivergenceReentersNormalAdmission(t *testing.T) {
	root := t.TempDir()

	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store:    s,
		Reasoner: lineageReasoner{},
	}

	_, cause := pivot.Observe(
		"WVC",
		"SOURCE_IDENTITY_CLAIM",
		"PRESERVED_EVIDENCE_OBSERVATION",
		"SOURCE_HASH_MATCH",
		func() error {
			return fmt.Errorf("source hash mismatch")
		},
	)
	if cause == nil {
		t.Fatal("expected pivot divergence")
	}

	em, duplicate, err := r.recapture(
		context.Background(),
		"",
		cause,
	)
	if err == nil {
		t.Fatal("RECAPTURE must preserve original failure")
	}

	if duplicate {
		t.Fatal("divergence unexpectedly reported duplicate")
	}

	if em.STA != core.StateAtGOV {
		t.Fatalf("divergence state = %q want %q", em.STA, core.StateAtGOV)
	}

	if !em.VAL.Recoil {
		t.Fatal("divergence did not pass RECOIL during normal admission")
	}

	if !em.VAL.WVC {
		t.Fatal("divergence did not pass WVC during normal admission")
	}

	if len(em.VAL.Gaps) != 1 ||
		em.VAL.Gaps[0] != "BRIDGEGAP:PIVOT_WVC" {
		t.Fatalf("unexpected divergence gaps: %#v", em.VAL.Gaps)
	}

	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}

	state, err := livefield.Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	admitted, ok := state.AtGOV[em.IDN]
	if !ok {
		t.Fatal("divergence was not handed to HUMAN_FINAL review")
	}

	if admitted.IDN != em.IDN {
		t.Fatalf("admitted divergence = %q", admitted.IDN)
	}

	if _, ok := state.Accepted[em.IDN]; ok {
		t.Fatal("divergence self-authorized into REG")
	}
}

func TestRecaptureReturnsTypedContinuationResult(t *testing.T) {
	root := t.TempDir()

	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store:    s,
		Reasoner: lineageReasoner{},
	}

	_, cause := pivot.Observe(
		"WVC",
		"SOURCE_IDENTITY_CLAIM",
		"PRESERVED_EVIDENCE_OBSERVATION",
		"SOURCE_HASH_MATCH",
		func() error {
			return fmt.Errorf("source hash mismatch")
		},
	)
	if cause == nil {
		t.Fatal("expected pivot divergence")
	}

	em, _, err := r.recapture(
		context.Background(),
		"",
		cause,
	)
	if err == nil {
		t.Fatal("RECAPTURE must retain original divergence")
	}

	var recaptured *RecaptureError
	if !errors.As(err, &recaptured) {
		t.Fatalf("error type = %T", err)
	}

	if recaptured.EmergION.IDN != em.IDN {
		t.Fatalf(
			"RECAPTURE identity = %q want %q",
			recaptured.EmergION.IDN,
			em.IDN,
		)
	}

	if recaptured.EmergION.STA != core.StateAtGOV {
		t.Fatalf("RECAPTURE state = %q", recaptured.EmergION.STA)
	}

	if !recaptured.EmergION.VAL.Recoil ||
		!recaptured.EmergION.VAL.WVC {
		t.Fatal("RECAPTURE result is not verified")
	}
}
