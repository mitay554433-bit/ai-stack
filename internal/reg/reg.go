package reg

import (
	"emergion-sovereign-runtime/internal/core"
	"fmt"
	"time"
)

func Accept(em core.EmergION, decisionID string) (core.EmergION, core.REGReceipt, error) {
	if em.STA != core.StateApproved {
		return em, core.REGReceipt{}, fmt.Errorf("GOV approval required")
	}
	em.STA = core.StateAccepted
	return em, core.REGReceipt{EmergIONID: em.IDN, DecisionID: decisionID, At: time.Now().UTC()}, nil
}
