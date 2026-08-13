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
