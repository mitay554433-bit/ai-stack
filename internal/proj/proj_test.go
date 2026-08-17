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

	rows, err := convergenceRows(st)
	if err != nil {
		t.Fatal(err)
	}

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

func TestSpatialConvergenceZoneDerivesAcceptedKinWithoutMerging(t *testing.T) {
	predecessor := core.EmergION{
		IDN: "E-PREDECESSOR",
		STA: core.StateAccepted,
		MEM: core.Memory{
			Summary: "accepted predecessor",
		},
	}

	successor := core.EmergION{
		IDN: "E-SUCCESSOR",
		STA: core.StateAccepted,
		MEM: core.Memory{
			Summary: "accepted successor",
		},
		EVO: core.Evolution{
			Supersedes: predecessor.IDN,
		},
	}

	st := core.EmptyState()
	st.Accepted[predecessor.IDN] = predecessor
	st.Accepted[successor.IDN] = successor

	rows, err := convergenceRows(st)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("Kin projection merged sovereign EmergIONs: rows = %d want 2", len(rows))
	}

	byID := map[string]convergenceRow{}
	for _, row := range rows {
		byID[row.ID] = row
	}

	if byID[predecessor.IDN].Kin != "root → E-PREDECESSOR; descendant → E-SUCCESSOR" {
		t.Fatalf("predecessor Kin = %q", byID[predecessor.IDN].Kin)
	}

	if byID[successor.IDN].Kin != "root → E-PREDECESSOR; predecessor → E-PREDECESSOR" {
		t.Fatalf("successor Kin = %q", byID[successor.IDN].Kin)
	}

	if _, ok := st.Accepted[predecessor.IDN]; !ok {
		t.Fatal("predecessor lost from sovereign accepted state")
	}
	if _, ok := st.Accepted[successor.IDN]; !ok {
		t.Fatal("successor lost from sovereign accepted state")
	}
}

func TestSpatialConvergenceZoneDoesNotInventDanglingKin(t *testing.T) {
	em := core.EmergION{
		IDN: "E-ACCEPTED",
		STA: core.StateAccepted,
		MEM: core.Memory{
			Summary: "accepted structure",
		},
		EVO: core.Evolution{
			Supersedes: "E-NOT-ACCEPTED",
		},
	}

	st := core.EmptyState()
	st.Accepted[em.IDN] = em

	if _, err := convergenceRows(st); err == nil {
		t.Fatal("dangling accepted Kin ancestry unexpectedly projected")
	}
}
