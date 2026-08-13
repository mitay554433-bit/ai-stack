package reg

import (
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/pivot"
	"fmt"
	"time"
)

func Accept(em core.EmergION, decisionID string) (core.EmergION, core.REGReceipt, error) {
	_, err := pivot.Observe(
		"REG_ACCEPTANCE",
		"ACCEPTANCE_REQUEST",
		"GOV_APPROVAL_EVIDENCE",
		"STATE_A_AND_DECISION_ID_PRESENT",
		func() error {
			if em.STA != core.StateApproved {
				return fmt.Errorf("GOV approval required")
			}
			if decisionID == "" {
				return fmt.Errorf("approving decision ID required")
			}
			return nil
		},
	)
	if err != nil {
		return em, core.REGReceipt{}, err
	}
	em.STA = core.StateAccepted
	return em, core.REGReceipt{EmergIONID: em.IDN, DecisionID: decisionID, At: time.Now().UTC()}, nil
}
