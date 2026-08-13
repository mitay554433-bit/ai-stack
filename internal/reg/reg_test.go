package reg

import (
	"testing"

	"emergion-sovereign-runtime/internal/core"
)

func TestAcceptRequiresApprovedState(t *testing.T) {
	em := core.EmergION{
		IDN: "E-REG",
		STA: core.StateAtGOV,
	}

	if _, _, err := Accept(em, "EV-D"); err == nil {
		t.Fatal("non-approved EmergION crossed REG")
	}
}

func TestAcceptRequiresApprovingDecisionID(t *testing.T) {
	em := core.EmergION{
		IDN: "E-REG",
		STA: core.StateApproved,
	}

	if _, _, err := Accept(em, ""); err == nil {
		t.Fatal("REG accepted without approving decision ID")
	}
}

func TestAcceptProducesAcceptedState(t *testing.T) {
	em := core.EmergION{
		IDN: "E-REG",
		STA: core.StateApproved,
	}

	accepted, receipt, err := Accept(em, "EV-D")
	if err != nil {
		t.Fatal(err)
	}

	if accepted.STA != core.StateAccepted {
		t.Fatalf("state = %s want %s", accepted.STA, core.StateAccepted)
	}

	if receipt.EmergIONID != em.IDN {
		t.Fatalf("receipt EmergION = %q", receipt.EmergIONID)
	}

	if receipt.DecisionID != "EV-D" {
		t.Fatalf("receipt decision = %q", receipt.DecisionID)
	}
}
