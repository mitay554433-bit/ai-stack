package field

import (
	"emergion-sovereign-runtime/internal/core"
	"fmt"
)

func Rebuild(events []core.Event) (core.State, error) {
	st := core.EmptyState()
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
			em, ok := st.AtGOV[ev.Decision.EmergIONID]
			if !ok {
				return st, fmt.Errorf("decision target not at GOV: %s", ev.Decision.EmergIONID)
			}
			delete(st.AtGOV, em.IDN)
			switch ev.Decision.Decision {
			case "APPROVE":
				em.STA = core.StateApproved
				st.Approved[em.IDN] = em
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
			delete(st.Approved, em.IDN)
			em.STA = core.StateAccepted
			st.Accepted[em.IDN] = em
		default:
			return st, fmt.Errorf("unknown event type %s", ev.Type)
		}
	}
	return st, nil
}
