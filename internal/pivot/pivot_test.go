package pivot

import (
	"errors"
	"testing"
)

func TestObservePassesReciprocalAgreement(t *testing.T) {
	result, err := Observe(
		"COSL_APPEND",
		"ENCODE",
		"DECODE",
		"ONE_EVENT_ONE_PHYSICAL_LINE",
		func() error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("pivot did not pass: %#v", result)
	}
	if result.Divergence != "" {
		t.Fatalf("unexpected divergence: %#v", result)
	}
}

func TestObserveCapturesDivergence(t *testing.T) {
	result, err := Observe(
		"COSL_APPEND",
		"ENCODE",
		"DECODE",
		"ONE_EVENT_ONE_PHYSICAL_LINE",
		func() error {
			return errors.New("encoded across multiple physical lines")
		},
	)

	if err == nil {
		t.Fatal("pivot divergence was allowed to cross boundary")
	}
	if result.Passed {
		t.Fatalf("divergent pivot passed: %#v", result)
	}
	if result.Divergence == "" {
		t.Fatalf("divergence evidence missing: %#v", result)
	}
}

func TestDivergenceEmergesAsNonAuthoritativeEmergION(t *testing.T) {
	_, err := Observe(
		"WVC",
		"SOURCE_IDENTITY_CLAIM",
		"PRESERVED_EVIDENCE_OBSERVATION",
		"SOURCE_HASH_MATCH",
		func() error {
			return errors.New("source hash mismatch")
		},
	)
	if err == nil {
		t.Fatal("expected divergence")
	}

	divergence, ok := err.(*DivergenceError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}

	em := divergence.EmergION

	if em.IDN == "" {
		t.Fatal("divergence EmergION identity missing")
	}

	if em.STA != "" {
		t.Fatalf("divergence gained authority state %q", em.STA)
	}

	if em.VAL.Recoil {
		t.Fatal("divergence EmergION incorrectly passed RECOIL")
	}

	if em.VAL.WVC {
		t.Fatal("divergence EmergION incorrectly passed WVC")
	}

	if len(em.VAL.Gaps) != 1 ||
		em.VAL.Gaps[0] != "BRIDGEGAP:PIVOT_WVC" {
		t.Fatalf("unexpected BRIDGEGAP: %#v", em.VAL.Gaps)
	}

	if em.REL["forward"] != "SOURCE_IDENTITY_CLAIM" {
		t.Fatalf("forward relationship missing: %#v", em.REL)
	}

	if em.REL["reciprocal"] != "PRESERVED_EVIDENCE_OBSERVATION" {
		t.Fatalf("reciprocal relationship missing: %#v", em.REL)
	}
}

func TestDivergenceEmergIONIsDeterministic(t *testing.T) {
	makeDivergence := func() *DivergenceError {
		_, err := Observe(
			"COSL_APPEND",
			"ENCODE",
			"DECODE",
			"ONE_EVENT_ONE_PHYSICAL_LINE",
			func() error {
				return errors.New("physical line divergence")
			},
		)

		divergence, ok := err.(*DivergenceError)
		if !ok {
			t.Fatalf("error type = %T", err)
		}
		return divergence
	}

	first := makeDivergence()
	second := makeDivergence()

	if first.EmergION.IDN != second.EmergION.IDN {
		t.Fatalf(
			"nondeterministic identity: %s != %s",
			first.EmergION.IDN,
			second.EmergION.IDN,
		)
	}

	if first.EmergION.MEM.SourceHash != second.EmergION.MEM.SourceHash {
		t.Fatal("divergence evidence hash changed")
	}
}
