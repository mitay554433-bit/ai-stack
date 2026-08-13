package pivot

import "fmt"

// Result records one reciprocal execution boundary.
//
// Forward describes what the active side is attempting.
// Reciprocal describes what observes or verifies that attempt.
// Invariant is the condition that must hold before execution may cross
// the boundary.
//
// Result is evidence only. It grants no authority and persists nothing.
type Result struct {
	Name       string
	Forward    string
	Reciprocal string
	Invariant  string
	Passed     bool
	Divergence string
}

// Observe executes the reciprocal observation for an already-produced
// forward result. A divergence fails closed before the caller crosses
// its authoritative boundary.
func Observe(
	name string,
	forward string,
	reciprocal string,
	invariant string,
	check func() error,
) (Result, error) {
	result := Result{
		Name:       name,
		Forward:    forward,
		Reciprocal: reciprocal,
		Invariant:  invariant,
	}

	if check == nil {
		result.Divergence = "reciprocal check missing"
		return result, fmt.Errorf(
			"%s pivot divergence: %s",
			name,
			result.Divergence,
		)
	}

	if err := check(); err != nil {
		result.Divergence = err.Error()
		return result, fmt.Errorf(
			"%s pivot divergence: %w",
			name,
			err,
		)
	}

	result.Passed = true
	return result, nil
}
