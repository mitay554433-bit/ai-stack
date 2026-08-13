package gov

import (
	"fmt"
	"strings"
	"time"

	"emergion-sovereign-runtime/internal/core"
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
	if authority != "HUMAN_FINAL" {
		return em, core.DecisionReceipt{}, fmt.Errorf("HUMAN_FINAL required")
	}
	if em.STA != core.StateAtGOV {
		return em, core.DecisionReceipt{}, fmt.Errorf("EmergION is not at GOV")
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
	if authority != "HUMAN_FINAL" {
		return em, core.DecisionReceipt{}, fmt.Errorf("HUMAN_FINAL required")
	}
	if em.STA != core.StateHeld {
		return em, core.DecisionReceipt{}, fmt.Errorf("EmergION is not held")
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
