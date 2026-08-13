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
