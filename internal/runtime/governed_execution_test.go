package runtime

import (
	"context"
	"testing"

	"emergion-sovereign-runtime/internal/adapters"
)

func governedExecutionProofRequest() adapters.ExecutionRequest {
	return adapters.ExecutionRequest{
		EmergIONID:      "E-GOVERNED-EXECUTION-PROOF",
		SourceHash:      "SOURCE-HASH-PROOF",
		AuthorizationID: "Q-PROOF",
		Adapter:         "LOCAL_GEMMA",
		Action:          "ANALYZE",
		Authority:       "CAP_ONLY",
	}
}

func TestGovernedExecutionRejectsBrokenLineageBeforeAdmission(t *testing.T) {
	request := governedExecutionProofRequest()

	result := adapters.BindExecutionResult(
		request,
		adapters.ExecutionResult{
			Succeeded: true,
			Output:    `{"summary":"bounded proof"}`,
		},
	)

	result.SourceHash = "OTHER-SOURCE"

	r := Runtime{}

	_, _, err := r.CaptureGovernedExecutionResult(
		context.Background(),
		request,
		result,
	)
	if err == nil {
		t.Fatal("broken execution lineage unexpectedly admitted")
	}
}

func TestGovernedExecutionRejectsActionPivotBeforeAdmission(t *testing.T) {
	request := governedExecutionProofRequest()

	result := adapters.BindExecutionResult(
		request,
		adapters.ExecutionResult{
			Succeeded: true,
		},
	)

	result.Action = "SEND"

	r := Runtime{}

	_, _, err := r.CaptureGovernedExecutionResult(
		context.Background(),
		request,
		result,
	)
	if err == nil {
		t.Fatal("execution action pivot unexpectedly admitted")
	}
}

func TestGovernedExecutionRejectsAuthorityPivotBeforeAdmission(t *testing.T) {
	request := governedExecutionProofRequest()

	result := adapters.BindExecutionResult(
		request,
		adapters.ExecutionResult{
			Succeeded: false,
			Error:     "bounded execution defect",
		},
	)

	result.Authority = "HUMAN_FINAL"

	r := Runtime{}

	_, _, err := r.CaptureGovernedExecutionResult(
		context.Background(),
		request,
		result,
	)
	if err == nil {
		t.Fatal("execution authority pivot unexpectedly admitted")
	}
}
