package gov

import (
	"fmt"
	"strings"
	"time"

	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/pivot"
)

type Decision string

const (
	Approve Decision = "APPROVE"
	Hold    Decision = "HOLD"
	Reject  Decision = "REJECT"
	Return  Decision = "RETURN"
	Resume  Decision = "RESUME"
)

func Decide(em core.EmergION, d Decision, authority, reason string) (core.EmergION, core.DecisionReceipt, error) {
	_, err := pivot.Observe(
		"GOV_DECISION",
		"REQUESTED_DECISION",
		"GOVERNED_STATE_AND_AUTHORITY",
		"HUMAN_FINAL_AND_STATE_G",
		func() error {
			if authority != "HUMAN_FINAL" {
				return fmt.Errorf("HUMAN_FINAL required")
			}
			if em.STA != core.StateAtGOV {
				return fmt.Errorf("EmergION is not at GOV")
			}
			return nil
		},
	)
	if err != nil {
		return em, core.DecisionReceipt{}, err
	}
	d = Decision(strings.ToUpper(string(d)))
	switch d {
	case Approve:
		em.STA = core.StateApproved
	case Hold:
		em.STA = core.StateHeld
	case Reject:
		em.STA = core.StateRejected
	case Return:
		em.STA = core.StateReturned
	default:
		return em, core.DecisionReceipt{}, fmt.Errorf("invalid decision")
	}
	r := core.DecisionReceipt{EmergIONID: em.IDN, Decision: string(d), Authority: authority, Reason: reason, At: time.Now().UTC()}
	return em, r, nil
}

func ResumeHeld(em core.EmergION, authority, reason string) (core.EmergION, core.DecisionReceipt, error) {
	_, err := pivot.Observe(
		"GOV_RESUME",
		"RESUME_REQUEST",
		"HELD_STATE_AND_AUTHORITY",
		"HUMAN_FINAL_AND_STATE_H",
		func() error {
			if authority != "HUMAN_FINAL" {
				return fmt.Errorf("HUMAN_FINAL required")
			}
			if em.STA != core.StateHeld {
				return fmt.Errorf("EmergION is not held")
			}
			return nil
		},
	)
	if err != nil {
		return em, core.DecisionReceipt{}, err
	}

	em.STA = core.StateAtGOV

	r := core.DecisionReceipt{
		EmergIONID: em.IDN,
		Decision:   string(Resume),
		Authority:  authority,
		Reason:     reason,
		At:         time.Now().UTC(),
	}

	return em, r, nil
}
