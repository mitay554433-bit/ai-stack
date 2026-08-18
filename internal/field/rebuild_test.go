package field

import (
	"emergion-sovereign-runtime/internal/core"
	"testing"
	"time"
)

func TestRebuild(t *testing.T) {
	em := core.EmergION{IDN: "E-1", STA: core.StateAtGOV, EVO: core.Evolution{Version: 1}}
	d := core.DecisionReceipt{EmergIONID: "E-1", Decision: "APPROVE", Authority: "HUMAN_FINAL", At: time.Now()}
	r := core.REGReceipt{EmergIONID: "E-1", DecisionID: "EV-D", At: time.Now()}
	st, err := Rebuild([]core.Event{{Type: "C", ID: "EV-C", EmergION: &em}, {Type: "D", ID: "EV-D", Decision: &d}, {Type: "R", ID: "EV-R", REG: &r}})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Accepted) != 1 || len(st.AtGOV) != 0 {
		t.Fatalf("bad state %#v", st)
	}
}

func TestRebuildRejectsUnlinkedREGReceipt(t *testing.T) {
	em := core.EmergION{IDN: "E-1", STA: core.StateAtGOV, EVO: core.Evolution{Version: 1}}
	d := core.DecisionReceipt{EmergIONID: "E-1", Decision: "APPROVE", Authority: "HUMAN_FINAL", At: time.Now()}
	r := core.REGReceipt{EmergIONID: "E-1", DecisionID: "EV-OTHER", At: time.Now()}
	if _, err := Rebuild([]core.Event{{Type: "C", ID: "EV-C", EmergION: &em}, {Type: "D", ID: "EV-D", Decision: &d}, {Type: "R", ID: "EV-R", REG: &r}}); err == nil {
		t.Fatal("expected unlinked REG receipt to fail")
	}
}

func TestHeldEmergIONCanResumeToGOV(t *testing.T) {
	em := core.EmergION{
		IDN: "E-HOLD-RESUME",
		STA: core.StateAtGOV,
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-HOLD",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "HOLD",
				Authority:  "HUMAN_FINAL",
			},
		},
		{
			Type: "D",
			ID:   "EV-RESUME",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "RESUME",
				Authority:  "HUMAN_FINAL",
			},
		},
	}

	st, err := Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := st.Held[em.IDN]; ok {
		t.Fatal("resumed EmergION remained held")
	}

	resumed, ok := st.AtGOV[em.IDN]
	if !ok {
		t.Fatal("resumed EmergION did not return to GOV")
	}

	if resumed.STA != core.StateAtGOV {
		t.Fatalf("resumed state = %s", resumed.STA)
	}
}

func TestResumeRejectsNonHeldTarget(t *testing.T) {
	em := core.EmergION{
		IDN: "E-NOT-HELD",
		STA: core.StateAtGOV,
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-RESUME",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "RESUME",
				Authority:  "HUMAN_FINAL",
			},
		},
	}

	if _, err := Rebuild(events); err == nil {
		t.Fatal("non-held EmergION was resumed")
	}
}

func TestRejectedEmergIONRemainsTerminal(t *testing.T) {
	em := core.EmergION{
		IDN: "E-REJECTED",
		STA: core.StateAtGOV,
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-REJECT",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "REJECT",
				Authority:  "HUMAN_FINAL",
			},
		},
	}

	st, err := Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := st.Rejected[em.IDN]; !ok {
		t.Fatal("rejected EmergION missing from rejected state")
	}

	if _, ok := st.AtGOV[em.IDN]; ok {
		t.Fatal("rejected EmergION remained at GOV")
	}

	if _, ok := st.Held[em.IDN]; ok {
		t.Fatal("rejected EmergION entered held state")
	}

	if _, ok := st.Returned[em.IDN]; ok {
		t.Fatal("rejected EmergION entered returned state")
	}

	if _, ok := st.Accepted[em.IDN]; ok {
		t.Fatal("rejected EmergION entered accepted state")
	}
}

func TestRejectedEmergIONCannotResume(t *testing.T) {
	em := core.EmergION{
		IDN: "E-REJECTED",
		STA: core.StateAtGOV,
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-REJECT",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "REJECT",
				Authority:  "HUMAN_FINAL",
			},
		},
		{
			Type: "D",
			ID:   "EV-RESUME",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "RESUME",
				Authority:  "HUMAN_FINAL",
			},
		},
	}

	if _, err := Rebuild(events); err == nil {
		t.Fatal("rejected EmergION was resumed")
	}
}

func TestDecisionReplayRequiresHumanFinal(t *testing.T) {
	em := core.EmergION{
		IDN: "E-AUTHORITY",
		STA: core.StateAtGOV,
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-D",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "APPROVE",
				Authority:  "MODEL",
			},
		},
	}

	if _, err := Rebuild(events); err == nil {
		t.Fatal("non-HUMAN_FINAL decision replay was accepted")
	}
}

func TestExactHumanFinalREGReplayProducesAcceptedState(t *testing.T) {
	em := core.EmergION{
		IDN: "E-SCZ-EXACT",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "source",
			Bytes:      1,
			Stored:     1,
			Summary:    "governed accepted structure",
		},
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-D-APPROVE",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "APPROVE",
				Authority:  "HUMAN_FINAL",
			},
		},
		{
			Type: "R",
			ID:   "EV-R",
			REG: &core.REGReceipt{
				EmergIONID: em.IDN,
				DecisionID: "EV-D-APPROVE",
			},
		},
	}

	st, err := Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	accepted, ok := st.Accepted[em.IDN]
	if !ok {
		t.Fatal("exact HUMAN_FINAL + REG replay did not enter Accepted")
	}

	if accepted.STA != core.StateAccepted {
		t.Fatalf(
			"accepted state = %q want %q",
			accepted.STA,
			core.StateAccepted,
		)
	}

	if _, ok := st.Approved[em.IDN]; ok {
		t.Fatal("REG-accepted EmergION remained in Approved")
	}
}

func TestREGReplayRejectsWrongApprovingDecisionLink(t *testing.T) {
	em := core.EmergION{
		IDN: "E-SCZ-BAD-LINK",
		STA: core.StateAtGOV,
		EVO: core.Evolution{Version: 1},
	}

	events := []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-D-APPROVE",
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "APPROVE",
				Authority:  "HUMAN_FINAL",
			},
		},
		{
			Type: "R",
			ID:   "EV-R",
			REG: &core.REGReceipt{
				EmergIONID: em.IDN,
				DecisionID: "EV-D-WRONG",
			},
		},
	}

	if _, err := Rebuild(events); err == nil {
		t.Fatal("REG replay accepted wrong approving decision link")
	}
}

func TestAcceptedKinRootSingleMember(t *testing.T) {
	accepted := map[string]core.EmergION{
		"E-A": {
			IDN: "E-A",
			STA: core.StateAccepted,
		},
	}

	root, err := AcceptedKinRoot(accepted, nil, "E-A")
	if err != nil {
		t.Fatal(err)
	}
	if root != "E-A" {
		t.Fatalf("root = %q want E-A", root)
	}
}

func TestAcceptedKinRootTraversesAcceptedAncestry(t *testing.T) {
	accepted := map[string]core.EmergION{
		"E-A": {
			IDN: "E-A",
			STA: core.StateAccepted,
		},
		"E-B": {
			IDN: "E-B",
			STA: core.StateAccepted,
			EVO: core.Evolution{
				Supersedes: "E-A",
			},
		},
		"E-C": {
			IDN: "E-C",
			STA: core.StateAccepted,
			EVO: core.Evolution{
				Supersedes: "E-B",
			},
		},
	}

	root, err := AcceptedKinRoot(accepted, nil, "E-C")
	if err != nil {
		t.Fatal(err)
	}
	if root != "E-A" {
		t.Fatalf("root = %q want E-A", root)
	}
}

func TestAcceptedKinRootRejectsMissingAcceptedPredecessor(t *testing.T) {
	accepted := map[string]core.EmergION{
		"E-B": {
			IDN: "E-B",
			STA: core.StateAccepted,
			EVO: core.Evolution{
				Supersedes: "E-A",
			},
		},
	}

	if _, err := AcceptedKinRoot(accepted, nil, "E-B"); err == nil {
		t.Fatal("missing accepted predecessor unexpectedly produced Kin root")
	}
}

func TestAcceptedKinRootRejectsCycle(t *testing.T) {
	accepted := map[string]core.EmergION{
		"E-A": {
			IDN: "E-A",
			STA: core.StateAccepted,
			EVO: core.Evolution{
				Supersedes: "E-B",
			},
		},
		"E-B": {
			IDN: "E-B",
			STA: core.StateAccepted,
			EVO: core.Evolution{
				Supersedes: "E-A",
			},
		},
	}

	if _, err := AcceptedKinRoot(accepted, nil, "E-A"); err == nil {
		t.Fatal("cyclic accepted Kin lineage unexpectedly produced root")
	}
}

func TestAcceptedKinRootStopsAtGovernedReturnedPredecessor(t *testing.T) {
	accepted := map[string]core.EmergION{
		"E-SUCCESSOR": {
			IDN: "E-SUCCESSOR",
			STA: core.StateAccepted,
			EVO: core.Evolution{
				Supersedes: "E-RETURNED",
			},
		},
	}

	returned := map[string]core.EmergION{
		"E-RETURNED": {
			IDN: "E-RETURNED",
			STA: core.StateReturned,
		},
	}

	root, err := AcceptedKinRoot(
		accepted,
		returned,
		"E-SUCCESSOR",
	)
	if err != nil {
		t.Fatal(err)
	}

	if root != "E-SUCCESSOR" {
		t.Fatalf("root = %q want E-SUCCESSOR", root)
	}
}
