package proj

import (
	"testing"

	"emergion-sovereign-runtime/internal/core"
)

func TestSpatialConvergenceZoneContainsAcceptedOnly(t *testing.T) {
	accepted := core.EmergION{
		IDN: "E-ACCEPTED",
		STA: core.StateAccepted,
		MEM: core.Memory{
			Summary: "accepted structure",
		},
	}

	atGOV := core.EmergION{
		IDN: "E-GOV",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			Summary: "not yet accepted",
		},
	}

	rejected := core.EmergION{
		IDN: "E-REJECTED",
		STA: core.StateRejected,
		MEM: core.Memory{
			Summary: "rejected structure",
		},
	}

	st := core.EmptyState()
	st.Accepted[accepted.IDN] = accepted
	st.AtGOV[atGOV.IDN] = atGOV
	st.Rejected[rejected.IDN] = rejected

	rows := convergenceRows(st)

	if len(rows) != 1 {
		t.Fatalf("SPATIAL CONVERGENCE ZONE rows = %d want 1", len(rows))
	}

	if rows[0].ID != accepted.IDN {
		t.Fatalf(
			"SPATIAL CONVERGENCE ZONE contains %q want %q",
			rows[0].ID,
			accepted.IDN,
		)
	}
}
