package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"emergion-sovereign-runtime/internal/adapters"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/reason"
)

type executionSignalEnvelope struct {
	Schema          string `json:"schema"`
	SourceKind      string `json:"source_kind"`
	ParentEmergION  string `json:"parent_emergion"`
	SourceHash      string `json:"source_hash"`
	AuthorizationID string `json:"authorization,omitempty"`
	Authority       string `json:"authority"`
	Adapter         string `json:"adapter"`
	Action          string `json:"action"`
	Succeeded       bool   `json:"succeeded"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
}

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

	envelope := executionSignalEnvelope{
		Schema:          "EXECUTION_SIGNAL_V1",
		SourceKind:      "EXECUTION_RESULT",
		ParentEmergION:  request.EmergIONID,
		SourceHash:      request.SourceHash,
		AuthorizationID: request.AuthorizationID,
		Authority:       request.Authority,
		Adapter:         request.Adapter,
		Action:          request.Action,
		Succeeded:       result.Succeeded,
		Output:          result.Output,
		Error:           result.Error,
	}

	content, err := json.Marshal(envelope)
	if err != nil {
		return core.EmergION{}, false, err
	}

	signalDir := filepath.Join(r.Store.Root, "execution-signals")
	if err := os.MkdirAll(signalDir, 0o700); err != nil {
		return core.EmergION{}, false, err
	}

	f, err := os.CreateTemp(signalDir, ".signal-*")
	if err != nil {
		return core.EmergION{}, false, err
	}

	path := f.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(path)
		}
	}()

	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return core.EmergION{}, false, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return core.EmergION{}, false, err
	}
	if err := f.Close(); err != nil {
		return core.EmergION{}, false, err
	}

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

	em, duplicate, err := signalRuntime.Capture(
		ctx,
		path,
		true,
	)

	if err == nil {
		remove = false
	}

	return em, duplicate, err
}
