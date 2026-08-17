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

func TestSpatialConvergenceZoneProjectsGovernedArchonym(t *testing.T) {
	em := core.EmergION{
		IDN: "E-ARCHONYM",
		STA: core.StateAccepted,
		MEM: core.Memory{
			Summary: "accepted semantic identity",
		},
		EVO: core.Evolution{
			Metadata: &core.Metadata{
				Archonym: "VERITEX CORE",
			},
		},
	}

	st := core.EmptyState()
	st.Accepted[em.IDN] = em

	rows, err := convergenceRows(st)
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 {
		t.Fatalf("rows = %d want 1", len(rows))
	}

	if rows[0].Archonym != "VERITEX CORE" {
		t.Fatalf(
			"projected Archonym = %q want VERITEX CORE",
			rows[0].Archonym,
		)
	}
}

func TestPRMCrystallizesAcceptedGovernedPrimitive(t *testing.T) {
	predecessor := core.EmergION{
		IDN: "E-PRM-ROOT",
		STA: core.StateAccepted,
		MEM: core.Memory{
			SourceHash: "source-root",
			Summary:    "accepted root",
		},
	}

	em := core.EmergION{
		IDN: "E-PRM-CHILD",
		STA: core.StateAccepted,
		MEM: core.Memory{
			SourceHash: "source-child",
			Summary:    "accepted reusable mechanism",
		},
		REL: map[string]string{
			"uses":     "bounded-input",
			"produces": "bounded-output",
		},
		CAP: []string{
			"ANALYZE",
			"SIMULATE",
		},
		VAL: core.Validation{
			Facts: []string{
				"deterministic transform",
				"bounded invariant",
			},
			Gaps: []string{
				"BRIDGEGAP:external_dependency",
			},
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Supersedes: predecessor.IDN,
			Delta: []string{
				"DC:+:SIMULATE",
			},
			Metadata: &core.Metadata{
				Archonym: "VERITEX PRIMITIVE",
				Facets: []core.Facet{
					core.FacetProgramForge,
					core.FacetAnalyticsForecast,
				},
				BuildNodes: []core.BuildNode{
					{
						ID:     "input",
						System: "INPUT",
						State:  "BOUND",
					},
					{
						ID:     "transform",
						System: "TRANSFORM",
						State:  "ACTIVE",
					},
				},
				BuildEdges: []core.BuildEdge{
					{
						From: "input",
						To:   "transform",
						Kind: "FLOW",
					},
				},
			},
		},
	}

	st := core.EmptyState()
	st.Accepted[predecessor.IDN] = predecessor
	st.Accepted[em.IDN] = em

	prms, err := crystallizePRMs(st)
	if err != nil {
		t.Fatal(err)
	}

	if len(prms) != 2 {
		t.Fatalf("PRMs = %d want 2", len(prms))
	}

	byID := map[string]prm{}
	for _, item := range prms {
		byID[item.SourceEmergIONID] = item
	}

	got := byID[em.IDN]

	if got.SourceEmergIONID != em.IDN {
		t.Fatalf(
			"source EmergION = %q want %q",
			got.SourceEmergIONID,
			em.IDN,
		)
	}

	if got.SourceHash != em.MEM.SourceHash {
		t.Fatalf(
			"source hash = %q want %q",
			got.SourceHash,
			em.MEM.SourceHash,
		)
	}

	if got.KinRoot != predecessor.IDN {
		t.Fatalf(
			"Kin root = %q want %q",
			got.KinRoot,
			predecessor.IDN,
		)
	}

	if got.Archonym != "VERITEX PRIMITIVE" {
		t.Fatalf("Archonym = %q", got.Archonym)
	}

	if len(got.Capabilities) != 2 ||
		got.Capabilities[0] != "ANALYZE" ||
		got.Capabilities[1] != "SIMULATE" {
		t.Fatalf("capabilities = %#v", got.Capabilities)
	}

	if len(got.Facts) != 2 {
		t.Fatalf("facts = %#v", got.Facts)
	}

	if got.Relationships["uses"] != "bounded-input" ||
		got.Relationships["produces"] != "bounded-output" {
		t.Fatalf("relationships = %#v", got.Relationships)
	}

	if len(got.Facets) != 2 {
		t.Fatalf("facets = %#v", got.Facets)
	}

	if len(got.BuildNodes) != 2 {
		t.Fatalf("build nodes = %#v", got.BuildNodes)
	}

	if len(got.BuildEdges) != 1 {
		t.Fatalf("build edges = %#v", got.BuildEdges)
	}

	if len(got.Delta) != 1 || got.Delta[0] != "DC:+:SIMULATE" {
		t.Fatalf("delta = %#v", got.Delta)
	}

	// PRM deliberately has no Gaps field. BRIDGEGAP remains unresolved
	// validation state and is not crystallized as reusable mechanism.
}

func TestPRMCrystallizationUsesAcceptedStateOnly(t *testing.T) {
	accepted := core.EmergION{
		IDN: "E-PRM-ACCEPTED",
		STA: core.StateAccepted,
		MEM: core.Memory{
			SourceHash: "accepted-source",
		},
	}

	atGOV := core.EmergION{
		IDN: "E-PRM-GOV",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "candidate-source",
		},
	}

	st := core.EmptyState()
	st.Accepted[accepted.IDN] = accepted
	st.AtGOV[atGOV.IDN] = atGOV

	prms, err := crystallizePRMs(st)
	if err != nil {
		t.Fatal(err)
	}

	if len(prms) != 1 {
		t.Fatalf("PRMs = %d want 1 accepted-only primitive", len(prms))
	}

	if prms[0].SourceEmergIONID != accepted.IDN {
		t.Fatalf(
			"PRM source = %q want %q",
			prms[0].SourceEmergIONID,
			accepted.IDN,
		)
	}
}

func TestPRMCrystallizationDoesNotMergeSovereignKin(t *testing.T) {
	root := core.EmergION{
		IDN: "E-PRM-KIN-ROOT",
		STA: core.StateAccepted,
		MEM: core.Memory{
			SourceHash: "kin-root-source",
		},
		CAP: []string{"ANALYZE"},
	}

	child := core.EmergION{
		IDN: "E-PRM-KIN-CHILD",
		STA: core.StateAccepted,
		MEM: core.Memory{
			SourceHash: "kin-child-source",
		},
		CAP: []string{"SIMULATE"},
		EVO: core.Evolution{
			Supersedes: root.IDN,
		},
	}

	st := core.EmptyState()
	st.Accepted[root.IDN] = root
	st.Accepted[child.IDN] = child

	prms, err := crystallizePRMs(st)
	if err != nil {
		t.Fatal(err)
	}

	if len(prms) != 2 {
		t.Fatalf(
			"sovereign Kin merged during PRM crystallization: %d PRMs",
			len(prms),
		)
	}

	if prms[0].SourceEmergIONID == prms[1].SourceEmergIONID {
		t.Fatal("distinct sovereign Kin produced duplicate PRM identity")
	}

	if st.Accepted[root.IDN].CAP[0] != "ANALYZE" {
		t.Fatal("PRM crystallization mutated root EmergION")
	}

	if st.Accepted[child.IDN].CAP[0] != "SIMULATE" {
		t.Fatal("PRM crystallization mutated child EmergION")
	}
}
