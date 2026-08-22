package fieldapi

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/gov"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/reg"
	"emergion-sovereign-runtime/internal/store"
)

type sawCirculationReasoner struct{}

func (sawCirculationReasoner) Analyze(
	_ context.Context,
	in reason.Input,
) (reason.Result, error) {
	return reason.Result{
		Summary: "bounded " + in.Name,
		Relationships: map[string]string{
			"source_name": in.Name,
			"source_kind": "PROGRAM",
		},
		Capabilities: []string{
			"OBS",
			"CMP",
			"ANALYZE",
		},
		Facts: []string{
			"source_preserved",
		},
		Risk: "L",
	}, nil
}

func (sawCirculationReasoner) Name() string {
	return "fieldapi-saw-circulation-test"
}

func (sawCirculationReasoner) Version(context.Context) string {
	return "1"
}

func TestCirculateSAWsUsesGovernedCaptureWithoutSelfAcceptance(t *testing.T) {
	root := t.TempDir()

	rt, err := Open(
		filepath.Join(root, "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	sourceA := core.EmergION{
		IDN: "E-FIELDAPI-SAW-A",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "fieldapi-saw-source-a",
			Bytes:      1,
			Stored:     1,
			Summary:    "accepted SAW source A",
		},
		REL: map[string]string{
			"COMPOSITION_KIN": "E-FIELDAPI-SAW-B",
		},
		CAP: []string{
			"OBS",
			"ANALYZE",
		},
		VAL: core.Validation{
			Facts:  []string{"bounded source A"},
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Version: 1,
		},
	}

	sourceB := core.EmergION{
		IDN: "E-FIELDAPI-SAW-B",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "fieldapi-saw-source-b",
			Bytes:      1,
			Stored:     1,
			Summary:    "accepted SAW source B",
		},
		REL: map[string]string{},
		CAP: []string{
			"CMP",
		},
		VAL: core.Validation{
			Facts:  []string{"bounded source B"},
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Version: 1,
		},
	}

	accept := func(em core.EmergION) {
		if _, err := rt.store.SaveCandidate(em); err != nil {
			t.Fatal(err)
		}

		approved, decision, err := gov.Decide(
			em,
			gov.Approve,
			"HUMAN_FINAL",
			"fieldapi SAW circulation proof",
		)
		if err != nil {
			t.Fatal(err)
		}

		decisionID, err := rt.store.SaveDecision(decision)
		if err != nil {
			t.Fatal(err)
		}

		_, receipt, err := reg.Accept(approved, decisionID)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := rt.store.SaveAccepted(receipt); err != nil {
			t.Fatal(err)
		}
	}

	accept(sourceA)
	accept(sourceB)

	before, err := rt.state()
	if err != nil {
		t.Fatal(err)
	}

	if len(before.Accepted) != 2 {
		t.Fatalf(
			"accepted prerequisite count = %d want 2",
			len(before.Accepted),
		)
	}

	circulated, err := rt.CirculateSAWs(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(circulated) != 1 {
		t.Fatalf(
			"circulated SAWs = %d want 1",
			len(circulated),
		)
	}

	em := circulated[0]

	if em.STA != core.StateAtGOV {
		t.Fatalf(
			"circulated SAW state = %q want %q",
			em.STA,
			core.StateAtGOV,
		)
	}

	if !em.VAL.Recoil || !em.VAL.WVC {
		t.Fatal("circulated SAW did not pass governed RECOIL/WVC")
	}

	if em.MEM.SourceHash == "" {
		t.Fatal("circulated SAW source identity missing")
	}

	if !strings.Contains(em.MEM.Summary, "SAW:") {
		t.Fatalf(
			"circulated SAW summary does not identify SAW source: %q",
			em.MEM.Summary,
		)
	}

	after, err := rt.state()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := after.AtGOV[em.IDN]; !ok {
		t.Fatalf(
			"circulated SAW %s did not enter GOV",
			em.IDN,
		)
	}

	if _, ok := after.Accepted[em.IDN]; ok {
		t.Fatal("CirculateSAWs self-authorized SAW into REG")
	}

	second, err := rt.CirculateSAWs(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(second) != 0 {
		t.Fatalf(
			"duplicate SAW circulation emitted %d candidates",
			len(second),
		)
	}

	finalState, err := rt.state()
	if err != nil {
		t.Fatal(err)
	}

	if len(finalState.Accepted) != 2 {
		t.Fatalf(
			"CirculateSAWs mutated accepted authority: %d",
			len(finalState.Accepted),
		)
	}

	if len(finalState.AtGOV) != 1 {
		t.Fatalf(
			"unexpected GOV candidate count after idempotent circulation: %d",
			len(finalState.AtGOV),
		)
	}

	events, err := rt.store.Events()
	if err != nil {
		t.Fatal(err)
	}

	for _, event := range events {
		if event.EmergION == nil ||
			event.EmergION.IDN != em.IDN {
			continue
		}

		if event.Type != "C" {
			t.Fatalf(
				"circulated SAW unexpectedly wrote authority event %q",
				event.Type,
			)
		}
	}
}

func TestCirculateSAWsDoesNothingWithoutGovernedComposition(t *testing.T) {
	root := t.TempDir()

	rt, err := Open(
		filepath.Join(root, "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	out, err := rt.CirculateSAWs(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 0 {
		t.Fatalf(
			"empty accepted FIELD produced %d SAWs",
			len(out),
		)
	}

	events, err := rt.store.Events()
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 0 {
		t.Fatalf(
			"empty circulation wrote %d events",
			len(events),
		)
	}
}

var _ = store.Hash

func TestStatusJSONUsesExistingRuntimeState(t *testing.T) {
	rt, err := Open(
		filepath.Join(t.TempDir(), "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	wire, err := rt.StatusJSON()
	if err != nil {
		t.Fatal(err)
	}

	if !json.Valid([]byte(wire)) {
		t.Fatalf("StatusJSON returned invalid JSON: %q", wire)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(wire), &got); err != nil {
		t.Fatal(err)
	}

	if _, ok := got["events"]; !ok {
		t.Fatalf("StatusJSON missing canonical events metric: %s", wire)
	}

	if _, ok := got["tip_hash"]; !ok {
		t.Fatalf("StatusJSON missing canonical tip hash: %s", wire)
	}
}

func TestActionsJSONCannotBypassREGAcceptance(t *testing.T) {
	rt, err := Open(
		filepath.Join(t.TempDir(), "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rt.ActionsJSON("E-NOT-REG-ACCEPTED", true); err == nil {
		t.Fatal("ActionsJSON bypassed REG acceptance")
	}
}

func TestDecideBindingCannotBypassGOV(t *testing.T) {
	rt, err := Open(
		filepath.Join(t.TempDir(), "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := rt.DecideBinding(
		"E-NOT-AT-GOV",
		"APPROVE",
		"binding boundary rejection proof",
	); err == nil {
		t.Fatal("DecideBinding bypassed GOV")
	}
}

func TestAuthorizeBindingCannotBypassREGAcceptance(t *testing.T) {
	rt, err := Open(
		filepath.Join(t.TempDir(), "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rt.AuthorizeBinding(
		"E-NOT-REG-ACCEPTED",
		"LOCAL_GEMMA",
		"ANALYZE",
		"binding boundary rejection proof",
		true,
	); err == nil {
		t.Fatal("AuthorizeBinding bypassed REG acceptance")
	}
}

func TestRenderCurrentJSONUsesCanonicalProjectionReceipt(t *testing.T) {
	rt, err := Open(
		filepath.Join(t.TempDir(), "state"),
		sawCirculationReasoner{},
	)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "projection")

	wire, err := rt.RenderCurrentJSON(out)
	if err != nil {
		t.Fatal(err)
	}

	if !json.Valid([]byte(wire)) {
		t.Fatalf("RenderCurrentJSON returned invalid JSON: %q", wire)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(wire), &got); err != nil {
		t.Fatal(err)
	}

	if _, ok := got["tip_hash"]; !ok {
		t.Fatalf("projection receipt missing tip_hash: %s", wire)
	}

	if _, ok := got["field_json_sha256"]; !ok {
		t.Fatalf("projection receipt missing field_json_sha256: %s", wire)
	}

	if _, ok := got["field_html_sha256"]; !ok {
		t.Fatalf("projection receipt missing field_html_sha256: %s", wire)
	}
}
