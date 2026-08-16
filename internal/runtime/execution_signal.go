package runtime

import (
	"context"
	"fmt"
	"strings"

	"emergion-sovereign-runtime/internal/adapters"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/reason"
)

func (r Runtime) CaptureExecutionResult(
	ctx context.Context,
	request adapters.ExecutionRequest,
	result adapters.ExecutionResult,
) (core.EmergION, bool, error) {
	if r.Store == nil {
		return core.EmergION{}, false, fmt.Errorf("runtime store not configured")
	}

	if request.EmergIONID == "" ||
		request.Adapter == "" ||
		request.Action == "" {
		return core.EmergION{}, false, fmt.Errorf(
			"incomplete execution request",
		)
	}

	if result.Adapter != request.Adapter ||
		result.Action != request.Action {
		return core.EmergION{}, false, fmt.Errorf(
			"execution result does not match request",
		)
	}

	var content strings.Builder
	writeField := func(key, value string) {
		fmt.Fprintf(&content, "%s=%d:", key, len(value))
		content.WriteString(value)
		content.WriteByte('\n')
	}

	writeField("SCHEMA", "EXECUTION_SIGNAL_V1")
	writeField("SOURCE_KIND", "EXECUTION_RESULT")
	writeField("PARENT_EMERGION", request.EmergIONID)
	writeField("SOURCE_HASH", request.SourceHash)
	writeField("AUTHORIZATION", request.AuthorizationID)
	writeField("AUTHORITY", request.Authority)
	writeField("ADAPTER", request.Adapter)
	writeField("ACTION", request.Action)
	writeField("SUCCEEDED", fmt.Sprintf("%t", result.Succeeded))
	writeField("OUTPUT", result.Output)
	writeField("ERROR", result.Error)

	facts := []string{
		"execution_result_observed",
	}

	risk := "L"
	if result.Succeeded {
		facts = append(facts, "execution_succeeded")
	} else {
		facts = append(facts, "execution_failed")
		risk = "M"
	}

	signalRuntime := r
	signalRuntime.Reasoner = fixedReasoner{
		name:    "execution-signal",
		version: "v1",
		result: reason.Result{
			Summary: "bounded execution result observation",
			Relationships: map[string]string{
				"source_kind":     "EXECUTION_RESULT",
				"parent_emergion": request.EmergIONID,
				"adapter":         request.Adapter,
				"action":          request.Action,
			},
			Capabilities: []string{
				"OBS",
				"CMP",
			},
			Facts: facts,
			Risk:  risk,
		},
	}

	return signalRuntime.captureBytes(
		ctx,
		"execution-result",
		[]byte(content.String()),
		"execution_signal",
	)
}
