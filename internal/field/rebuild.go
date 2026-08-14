package field

import (
	"emergion-sovereign-runtime/internal/adapters"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/pivot"
	"fmt"
)

func Rebuild(events []core.Event) (core.State, error) {
	st := core.EmptyState()
	decisionEvents := map[string]string{}
	for _, ev := range events {
		st.Events++
		st.TipHash = ev.SelfHash
		switch ev.Type {
		case "C":
			if ev.EmergION == nil {
				return st, fmt.Errorf("candidate event missing EmergION")
			}
			if _, exists := st.AtGOV[ev.EmergION.IDN]; exists {
				return st, fmt.Errorf("duplicate candidate %s", ev.EmergION.IDN)
			}
			st.AtGOV[ev.EmergION.IDN] = *ev.EmergION
		case "D":
			if ev.Decision == nil {
				return st, fmt.Errorf("decision event missing receipt")
			}

			_, err := pivot.Observe(
				"GOV_REPLAY",
				"DECISION_RECEIPT",
				"LEDGER_AUTHORITY_OBSERVATION",
				"HUMAN_FINAL_REQUIRED",
				func() error {
					if ev.Decision.Authority != "HUMAN_FINAL" {
						return fmt.Errorf("decision requires HUMAN_FINAL")
					}
					return nil
				},
			)
			if err != nil {
				return st, err
			}

			if ev.Decision.Decision == "RESUME" {
				em, ok := st.Held[ev.Decision.EmergIONID]
				if !ok {
					return st, fmt.Errorf("resume target not held: %s", ev.Decision.EmergIONID)
				}
				delete(st.Held, em.IDN)
				em.STA = core.StateAtGOV
				st.AtGOV[em.IDN] = em
				continue
			}
			em, ok := st.AtGOV[ev.Decision.EmergIONID]
			if !ok {
				return st, fmt.Errorf("decision target not at GOV: %s", ev.Decision.EmergIONID)
			}
			delete(st.AtGOV, em.IDN)
			switch ev.Decision.Decision {
			case "APPROVE":
				em.STA = core.StateApproved
				st.Approved[em.IDN] = em
				decisionEvents[em.IDN] = ev.ID
			case "HOLD":
				em.STA = core.StateHeld
				st.Held[em.IDN] = em
			case "REJECT":
				em.STA = core.StateRejected
				st.Rejected[em.IDN] = em
			case "RETURN":
				em.STA = core.StateReturned
				st.Returned[em.IDN] = em
			default:
				return st, fmt.Errorf("invalid decision %s", ev.Decision.Decision)
			}
		case "R":
			if ev.REG == nil {
				return st, fmt.Errorf("REG event missing receipt")
			}
			em, ok := st.Approved[ev.REG.EmergIONID]
			if !ok {
				return st, fmt.Errorf("REG target not approved: %s", ev.REG.EmergIONID)
			}
			_, err := pivot.Observe(
				"REG_REPLAY",
				"REG_RECEIPT",
				"APPROVING_DECISION_OBSERVATION",
				"EXACT_APPROVING_DECISION_LINK",
				func() error {
					if ev.REG.DecisionID == "" || ev.REG.DecisionID != decisionEvents[em.IDN] {
						return fmt.Errorf(
							"REG receipt does not reference approving decision for %s",
							em.IDN,
						)
					}
					return nil
				},
			)
			if err != nil {
				return st, err
			}
			delete(st.Approved, em.IDN)
			delete(decisionEvents, em.IDN)
			em.STA = core.StateAccepted
			st.Accepted[em.IDN] = em
		case "Q":
			if ev.ActionAuthorization == nil {
				return st, fmt.Errorf("action authorization event missing receipt")
			}

			receipt := *ev.ActionAuthorization
			em, ok := st.Accepted[receipt.EmergIONID]
			if !ok {
				return st, fmt.Errorf("action authorization target not REG-accepted: %s", receipt.EmergIONID)
			}

			var facets []string
			if em.EVO.Metadata != nil {
				for _, facet := range em.EVO.Metadata.Facets {
					facets = append(facets, string(facet))
				}
			}

			candidates := adapters.DeriveActionCandidates(facets, em.CAP, true)
			matched := false
			for _, candidate := range candidates {
				if candidate.Adapter != receipt.Adapter || candidate.Action != receipt.Action {
					continue
				}

				matched = true
				if candidate.HumanFinalRequired && receipt.Authority != "HUMAN_FINAL" {
					return st, fmt.Errorf("action %s:%s requires HUMAN_FINAL", receipt.Adapter, receipt.Action)
				}
				break
			}

			if !matched {
				return st, fmt.Errorf("action %s:%s is not derivable from accepted EmergION %s", receipt.Adapter, receipt.Action, receipt.EmergIONID)
			}
			if !receipt.Authorized {
				return st, fmt.Errorf("action authorization is not authorized")
			}

			st.ActionAuthorizations = append(st.ActionAuthorizations, receipt)
		default:
			return st, fmt.Errorf("unknown event type %s", ev.Type)
		}
	}
	return st, nil
}
