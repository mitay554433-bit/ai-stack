package cosl

import (
	"testing"
	"time"

	"emergion-sovereign-runtime/internal/core"
)

func TestRoundTrip(t *testing.T) {
	e := core.Event{Type: "C", ID: "EV-1", At: time.Unix(1, 0).UTC(), EmergION: &core.EmergION{IDN: "E-1", STA: core.StateAtGOV, EVO: core.Evolution{Version: 1}}}
	line, err := Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(line)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != e.ID || got.SelfHash == "" {
		t.Fatalf("bad roundtrip: %#v", got)
	}
}
