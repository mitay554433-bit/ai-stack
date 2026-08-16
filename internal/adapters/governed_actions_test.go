package adapters

import (
	"reflect"
	"testing"
	"time"

	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/gov"
	"emergion-sovereign-runtime/internal/reg"
)

func TestREGAcceptedStateDerivesBoundedActionsWithoutExecution(t *testing.T) {
	em := core.EmergION{
		IDN: "E-ACTION-PROOF",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "action-proof-source",
			Codec:      "test",
			Bytes:      1,
			Stored:     1,
			Summary:    "governed communications action proof",
			Provenance: "test",
		},
		REL: map[string]string{
			"source": "governed-test",
		},
		CAP: []string{
			"DRAFT",
			"SEND",
		},
		VAL: core.Validation{
			Facts: []string{
				"communication drafting supported",
				"sending requires HUMAN_FINAL",
			},
			Risk:   "L",
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Metadata: &core.Metadata{
				Topology:   core.TopologyDodecahedronV1,
				CapturedAt: time.Now().UTC(),
				Facets: []core.Facet{
					core.FacetCommunications,
				},
			},
		},
	}

	approved, decision, err := gov.Decide(
		em,
		gov.Approve,
		"HUMAN_FINAL",
		"approve bounded action derivation proof",
	)
	if err != nil {
		t.Fatal(err)
	}

	if approved.STA != core.StateApproved {
		t.Fatalf("approved state = %s", approved.STA)
	}
	if decision.Authority != "HUMAN_FINAL" {
		t.Fatalf("decision authority = %q", decision.Authority)
	}

	accepted, receipt, err := reg.Accept(
		approved,
		"EV-D-ACTION-PROOF",
	)
	if err != nil {
		t.Fatal(err)
	}

	if accepted.STA != core.StateAccepted {
		t.Fatalf("accepted state = %s", accepted.STA)
	}
	if receipt.DecisionID != "EV-D-ACTION-PROOF" {
		t.Fatalf("REG decision linkage = %q", receipt.DecisionID)
	}

	beforeCAP := append([]string(nil), accepted.CAP...)
	beforeFacets := append([]core.Facet(nil), accepted.EVO.Metadata.Facets...)

	facets := make([]string, 0, len(accepted.EVO.Metadata.Facets))
	for _, facet := range accepted.EVO.Metadata.Facets {
		facets = append(facets, string(facet))
	}

	actions := DeriveActionCandidates(
		facets,
		accepted.CAP,
		false,
	)

	if !reflect.DeepEqual(beforeCAP, accepted.CAP) {
		t.Fatal("action derivation mutated accepted capabilities")
	}

	if !reflect.DeepEqual(beforeFacets, accepted.EVO.Metadata.Facets) {
		t.Fatal("action derivation mutated accepted facets")
	}

	var draft *ActionCandidate
	var send *ActionCandidate

	for i := range actions {
		action := &actions[i]

		if action.Adapter == "EMAIL" && action.Action == "DRAFT" {
			draft = action
		}

		if action.Adapter == "EMAIL" && action.Action == "SEND" {
			send = action
		}
	}

	if draft == nil {
		t.Fatal("EMAIL:DRAFT action candidate missing")
	}
	if draft.HumanFinalRequired {
		t.Fatal("EMAIL:DRAFT unexpectedly requires HUMAN_FINAL")
	}

	if send == nil {
		t.Fatal("EMAIL:SEND action candidate missing")
	}
	if !send.HumanFinalRequired {
		t.Fatal("EMAIL:SEND must require HUMAN_FINAL")
	}
}
