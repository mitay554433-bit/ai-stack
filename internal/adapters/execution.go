package adapters

import (
	"fmt"

	"emergion-sovereign-runtime/internal/core"
)

type ExecutionRequest struct {
	EmergIONID      string `json:"emergion"`
	AuthorizationID string `json:"authorization,omitempty"`
	Adapter         string `json:"adapter"`
	Action          string `json:"action"`
	Authority       string `json:"authority"`
}

type ExecutionResult struct {
	Adapter   string `json:"adapter"`
	Action    string `json:"action"`
	Succeeded bool   `json:"succeeded"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Executor interface {
	Execute(ExecutionRequest) (ExecutionResult, error)
}

func PrepareExecution(
	st core.State,
	emergionID string,
	adapter string,
	action string,
	localGemma bool,
) (ExecutionRequest, error) {
	em, ok := st.Accepted[emergionID]
	if !ok {
		return ExecutionRequest{}, fmt.Errorf(
			"execution target is not REG-accepted: %s",
			emergionID,
		)
	}

	var facets []string
	if em.EVO.Metadata != nil {
		for _, facet := range em.EVO.Metadata.Facets {
			facets = append(facets, string(facet))
		}
	}

	var candidate *ActionCandidate
	for _, item := range DeriveActionCandidates(
		facets,
		em.CAP,
		localGemma,
	) {
		if item.Adapter == adapter && item.Action == action {
			value := item
			candidate = &value
			break
		}
	}

	if candidate == nil {
		return ExecutionRequest{}, fmt.Errorf(
			"action %s:%s is not derivable from accepted EmergION %s",
			adapter,
			action,
			emergionID,
		)
	}

	var authorization *core.ActionAuthorizationReceipt
	for i := len(st.ActionAuthorizations) - 1; i >= 0; i-- {
		item := st.ActionAuthorizations[i]

		if item.EmergIONID == emergionID &&
			item.Adapter == adapter &&
			item.Action == action &&
			item.Authorized {
			value := item
			authorization = &value
			break
		}
	}

	if candidate.HumanFinalRequired {
		if authorization == nil {
			return ExecutionRequest{}, fmt.Errorf(
				"action %s:%s requires authorization",
				adapter,
				action,
			)
		}

		if authorization.Authority != "HUMAN_FINAL" {
			return ExecutionRequest{}, fmt.Errorf(
				"action %s:%s requires HUMAN_FINAL",
				adapter,
				action,
			)
		}
	}

	return ExecutionRequest{
		EmergIONID: emergionID,
		Adapter:    adapter,
		Action:     action,
		Authority:  candidate.Authority,
	}, nil
}
