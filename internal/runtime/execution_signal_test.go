package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"emergion-sovereign-runtime/internal/adapters"
	"emergion-sovereign-runtime/internal/core"
	livefield "emergion-sovereign-runtime/internal/field"
	"emergion-sovereign-runtime/internal/store"
)

func TestExecutionResultReentersEmergIONPipeline(t *testing.T) {
	root := t.TempDir()

	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store: s,
	}

	request := adapters.ExecutionRequest{
		EmergIONID: "E-PARENT",
		Adapter:    "EMAIL",
		Action:     "SEND",
		Authority:  "SEND_GATED",
	}

	result := adapters.ExecutionResult{
		Adapter:   "EMAIL",
		Action:    "SEND",
		Succeeded: true,
		Output:    "bounded execution proof",
	}

	em, duplicate, err := r.CaptureExecutionResult(
		context.Background(),
		request,
		result,
	)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("execution signal unexpectedly duplicate")
	}

	if em.STA != core.StateAtGOV {
		t.Fatalf("execution signal state = %s", em.STA)
	}
	if !em.VAL.Recoil || !em.VAL.WVC {
		t.Fatalf("execution signal not verified: %#v", em.VAL)
	}
	if em.MEM.Provenance == "" {
		t.Fatal("execution signal provenance missing")
	}

	events, err := s.Events()
	if err != nil {
		t.Fatal(err)
	}

	st, err := livefield.Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := st.AtGOV[em.IDN]
	if !ok {
		t.Fatalf("execution signal %s missing from GOV", em.IDN)
	}

	if got.REL["source_kind"] != "EXECUTION_RESULT" {
		t.Fatalf(
			"source kind = %q",
			got.REL["source_kind"],
		)
	}

	if got.REL["parent_emergion"] != "E-PARENT" {
		t.Fatalf(
			"parent = %q",
			got.REL["parent_emergion"],
		)
	}
}

func TestFailedExecutionAlsoBecomesSignal(t *testing.T) {
	root := t.TempDir()

	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store: s,
	}

	request := adapters.ExecutionRequest{
		EmergIONID: "E-PARENT",
		Adapter:    "EMAIL",
		Action:     "SEND",
		Authority:  "SEND_GATED",
	}

	result := adapters.ExecutionResult{
		Adapter:   "EMAIL",
		Action:    "SEND",
		Succeeded: false,
		Error:     "simulated execution failure",
	}

	em, duplicate, err := r.CaptureExecutionResult(
		context.Background(),
		request,
		result,
	)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("failed execution signal unexpectedly duplicate")
	}

	if em.STA != core.StateAtGOV {
		t.Fatalf("failed execution signal state = %s", em.STA)
	}

	found := false
	for _, fact := range em.VAL.Facts {
		if fact == "execution_failed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("execution_failed fact missing: %#v", em.VAL.Facts)
	}

	if em.VAL.Risk != "M" {
		t.Fatalf("failed execution risk = %q", em.VAL.Risk)
	}
}

func TestExecutionResultMustMatchRequest(t *testing.T) {
	root := t.TempDir()

	s, err := store.Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}

	r := Runtime{
		Store: s,
	}

	_, _, err = r.CaptureExecutionResult(
		context.Background(),
		adapters.ExecutionRequest{
			EmergIONID: "E-PARENT",
			Adapter:    "EMAIL",
			Action:     "SEND",
		},
		adapters.ExecutionResult{
			Adapter:   "WEB",
			Action:    "DEPLOY",
			Succeeded: true,
		},
	)

	if err == nil {
		t.Fatal("mismatched execution result unexpectedly admitted")
	}
}
