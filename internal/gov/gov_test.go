package gov

import (
	"testing"

	"emergion-sovereign-runtime/internal/core"
)

func TestResumeHeldRequiresHumanFinal(t *testing.T) {
	em := core.EmergION{
		IDN: "E-HOLD",
		STA: core.StateHeld,
	}

	if _, _, err := ResumeHeld(em, "MODEL", "resume"); err == nil {
		t.Fatal("non-HUMAN_FINAL authority resumed held EmergION")
	}
}

func TestResumeHeldReturnsToGOV(t *testing.T) {
	em := core.EmergION{
		IDN: "E-HOLD",
		STA: core.StateHeld,
	}

	resumed, receipt, err := ResumeHeld(
		em,
		"HUMAN_FINAL",
		"continue review",
	)
	if err != nil {
		t.Fatal(err)
	}

	if resumed.STA != core.StateAtGOV {
		t.Fatalf("resume state = %s want %s", resumed.STA, core.StateAtGOV)
	}

	if receipt.Decision != string(Resume) {
		t.Fatalf("decision = %q", receipt.Decision)
	}

	if receipt.Authority != "HUMAN_FINAL" {
		t.Fatalf("authority = %q", receipt.Authority)
	}
}
