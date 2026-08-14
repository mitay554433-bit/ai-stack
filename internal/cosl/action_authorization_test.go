package cosl

import (
	"testing"
	"time"

	"emergion-sovereign-runtime/internal/core"
)

func TestActionAuthorizationCOSLRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Nanosecond)

	ev := core.Event{
		Type: "Q",
		ID:   "EV-Q-COSL",
		At:   now,
		ActionAuthorization: &core.ActionAuthorizationReceipt{
			EmergIONID: "E-ACTION-AUTH-PROOF",
			Adapter:    "EMAIL",
			Action:     "SEND",
			Authority:  "HUMAN_FINAL",
			Authorized: true,
			Reason:     "bounded authorization",
			At:         now,
		},
		PrevHash: "previous",
	}

	line, err := Encode(ev)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := Decode(line)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.ActionAuthorization == nil {
		t.Fatal("action authorization lost during COSL round trip")
	}

	got := decoded.ActionAuthorization
	if got.EmergIONID != "E-ACTION-AUTH-PROOF" ||
		got.Adapter != "EMAIL" ||
		got.Action != "SEND" ||
		got.Authority != "HUMAN_FINAL" ||
		!got.Authorized ||
		got.Reason != "bounded authorization" {
		t.Fatalf("authorization changed during COSL round trip: %#v", got)
	}
}
