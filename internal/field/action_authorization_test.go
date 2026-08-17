package field

import (
	"emergion-sovereign-runtime/internal/adapters"
	"strings"
	"testing"
	"time"

	"emergion-sovereign-runtime/internal/core"
)

func actionAuthorizationBaseEvents() []core.Event {
	now := time.Now().UTC()

	em := core.EmergION{
		IDN: "E-ACTION-AUTH-PROOF",
		STA: core.StateAtGOV,
		MEM: core.Memory{
			SourceHash: "action-auth-proof",
			Codec:      "test",
			Bytes:      1,
			Stored:     1,
			Summary:    "governed action authorization proof",
			Provenance: "test",
		},
		REL: map[string]string{
			"source": "test",
		},
		CAP: []string{
			"DRAFT",
			"SEND",
		},
		VAL: core.Validation{
			Facts:  []string{"communication action supported"},
			Risk:   "L",
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Metadata: &core.Metadata{
				Topology:   core.TopologyDodecahedronV1,
				CapturedAt: now,
				Facets: []core.Facet{
					core.FacetCommunications,
				},
			},
		},
	}

	return []core.Event{
		{
			Type:     "C",
			ID:       "EV-C",
			At:       now,
			EmergION: &em,
		},
		{
			Type: "D",
			ID:   "EV-D",
			At:   now,
			Decision: &core.DecisionReceipt{
				EmergIONID: em.IDN,
				Decision:   "APPROVE",
				Authority:  "HUMAN_FINAL",
				Reason:     "approve action authorization proof",
				At:         now,
			},
		},
		{
			Type: "R",
			ID:   "EV-R",
			At:   now,
			REG: &core.REGReceipt{
				EmergIONID: em.IDN,
				DecisionID: "EV-D",
				At:         now,
			},
		},
	}
}

func TestActionAuthorizationRetainedWithoutExecuting(t *testing.T) {
	events := actionAuthorizationBaseEvents()
	now := time.Now().UTC()

	events = append(events, core.Event{
		Type: "Q",
		ID:   "EV-Q",
		At:   now,
		ActionAuthorization: &core.ActionAuthorizationReceipt{
			EmergIONID: "E-ACTION-AUTH-PROOF",
			Adapter:    "EMAIL",
			Action:     "SEND",
			Authority:  "HUMAN_FINAL",
			Authorized: true,
			Reason:     "authorize bounded send intent",
			At:         now,
		},
	})

	st, err := Rebuild(events)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := st.Accepted["E-ACTION-AUTH-PROOF"]; !ok {
		t.Fatal("Q removed or changed REG-accepted EmergION")
	}

	if len(st.ActionAuthorizations) != 1 {
		t.Fatalf(
			"action authorizations = %d want 1",
			len(st.ActionAuthorizations),
		)
	}

	got := st.ActionAuthorizations[0]
	if got.Adapter != "EMAIL" || got.Action != "SEND" {
		t.Fatalf("unexpected authorization: %#v", got)
	}
	if got.Authority != "HUMAN_FINAL" || !got.Authorized {
		t.Fatalf("invalid authorization retained: %#v", got)
	}
}

func TestActionAuthorizationBeforeREGIsRejected(t *testing.T) {
	events := actionAuthorizationBaseEvents()
	now := time.Now().UTC()

	q := core.Event{
		Type: "Q",
		ID:   "EV-Q",
		At:   now,
		ActionAuthorization: &core.ActionAuthorizationReceipt{
			EmergIONID: "E-ACTION-AUTH-PROOF",
			Adapter:    "EMAIL",
			Action:     "SEND",
			Authority:  "HUMAN_FINAL",
			Authorized: true,
			At:         now,
		},
	}

	events = append(events[:2], q)

	_, err := Rebuild(events)
	if err == nil {
		t.Fatal("Q before REG unexpectedly accepted")
	}
	if !strings.Contains(err.Error(), "not REG-accepted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNonDerivableActionAuthorizationIsRejected(t *testing.T) {
	events := actionAuthorizationBaseEvents()
	now := time.Now().UTC()

	events = append(events, core.Event{
		Type: "Q",
		ID:   "EV-Q",
		At:   now,
		ActionAuthorization: &core.ActionAuthorizationReceipt{
			EmergIONID: "E-ACTION-AUTH-PROOF",
			Adapter:    "PAYMENTS",
			Action:     "TRANSFER",
			Authority:  "HUMAN_FINAL",
			Authorized: true,
			At:         now,
		},
	})

	_, err := Rebuild(events)
	if err == nil {
		t.Fatal("non-derivable action unexpectedly authorized")
	}
	if !strings.Contains(err.Error(), "not derivable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGatedActionRequiresHumanFinal(t *testing.T) {
	events := actionAuthorizationBaseEvents()
	now := time.Now().UTC()

	events = append(events, core.Event{
		Type: "Q",
		ID:   "EV-Q",
		At:   now,
		ActionAuthorization: &core.ActionAuthorizationReceipt{
			EmergIONID: "E-ACTION-AUTH-PROOF",
			Adapter:    "EMAIL",
			Action:     "SEND",
			Authority:  "MODEL",
			Authorized: true,
			At:         now,
		},
	})

	_, err := Rebuild(events)
	if err == nil {
		t.Fatal("SEND without HUMAN_FINAL unexpectedly authorized")
	}
	if !strings.Contains(err.Error(), "requires HUMAN_FINAL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionAuthorizationRemainsConfinedToExactKinMember(t *testing.T) {
	now := time.Now().UTC()

	predecessor := core.EmergION{
		IDN: "E-KIN-Q-PREDECESSOR",
		STA: core.StateAccepted,
		MEM: core.Memory{
			SourceHash: "kin-q-predecessor",
			Codec:      "test",
			Bytes:      1,
			Stored:     1,
			Summary:    "accepted predecessor",
			Provenance: "test",
		},
		CAP: []string{"SEND"},
		VAL: core.Validation{
			Facts:  []string{"bounded"},
			Risk:   "L",
			Recoil: true,
			WVC:    true,
		},
		EVO: core.Evolution{
			Version: 1,
			Metadata: &core.Metadata{
				Topology:   core.TopologyDodecahedronV1,
				CapturedAt: now,
				Facets: []core.Facet{
					core.FacetAnalyticsForecast,
				},
			},
		},
	}

	successor := predecessor
	successor.IDN = "E-KIN-Q-SUCCESSOR"
	successor.MEM.SourceHash = "kin-q-successor"
	successor.MEM.Summary = "accepted successor"
	successor.EVO.Supersedes = predecessor.IDN

	st := core.EmptyState()
	st.Accepted[predecessor.IDN] = predecessor
	st.Accepted[successor.IDN] = successor

	st.ActionAuthorizations = append(
		st.ActionAuthorizations,
		core.ActionAuthorizationReceipt{
			EventID:    "EV-Q-KIN-PROOF",
			EmergIONID: predecessor.IDN,
			Adapter:    "EMAIL",
			Action:     "SEND",
			Authority:  "HUMAN_FINAL",
			Authorized: true,
			At:         now,
		},
	)

	if _, err := adapters.PrepareExecution(
		st,
		predecessor.IDN,
		"EMAIL",
		"SEND",
		false,
	); err != nil {
		t.Fatalf("exact Kin member authorization rejected: %v", err)
	}

	if _, err := adapters.PrepareExecution(
		st,
		successor.IDN,
		"EMAIL",
		"SEND",
		false,
	); err == nil {
		t.Fatal("predecessor Q authorization leaked across sovereign Kin boundary")
	}
}
