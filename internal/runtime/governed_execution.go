package runtime

import (
	"context"
	"fmt"

	"emergion-sovereign-runtime/internal/adapters"
	"emergion-sovereign-runtime/internal/core"
)

// CaptureGovernedExecutionResult is the strict execution-result admission seam.
//
// # ACCEPTED → EXECUTION REQUEST → EXECUTION RESULT → SIGNAL → G
//
// RESULT != TRUTH.
// EXECUTOR != AUTHORITY.
// Lineage continuity is required before the result may re-enter the
// governed EmergION circulation.
func (r *Runtime) CaptureGovernedExecutionResult(
	ctx context.Context,
	request adapters.ExecutionRequest,
	result adapters.ExecutionResult,
) (core.EmergION, bool, error) {
	if err := adapters.VerifyExecutionResult(request, result); err != nil {
		return core.EmergION{}, false, fmt.Errorf(
			"execution lineage verification failed: %w",
			err,
		)
	}

	return r.CaptureExecutionResult(ctx, request, result)
}
