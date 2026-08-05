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
	st, err := Rebuild([]core.Event{{Type: "C", EmergION: &em}, {Type: "D", Decision: &d}, {Type: "R", REG: &r}})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Accepted) != 1 || len(st.AtGOV) != 0 {
		t.Fatalf("bad state %#v", st)
	}
}
