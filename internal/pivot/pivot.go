package pivot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"emergion-sovereign-runtime/internal/core"
)

// Result is evidence produced at a reciprocal execution boundary.
//
// It carries no authority and persists nothing by itself.
type Result struct {
	Name       string
	Forward    string
	Reciprocal string
	Invariant  string
	Passed     bool
	Divergence string
}

// DivergenceError is both an error and an emergence boundary.
//
// The EmergION inside it is deliberately non-authoritative:
// it has not passed RECOIL, WVC, GOV, or REG.
type DivergenceError struct {
	Result   Result
	EmergION core.EmergION
}

func (e *DivergenceError) Error() string {
	return fmt.Sprintf(
		"%s pivot divergence: %s",
		e.Result.Name,
		e.Result.Divergence,
	)
}

func (r Result) Evidence() string {
	return strings.Join([]string{
		"name=" + r.Name,
		"forward=" + r.Forward,
		"reciprocal=" + r.Reciprocal,
		"invariant=" + r.Invariant,
		"divergence=" + r.Divergence,
	}, "\n")
}

func bridgegapName(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))

	var b strings.Builder
	lastUnderscore := false

	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if b.Len() != 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	return strings.Trim(b.String(), "_")
}

func divergenceEmergION(result Result) core.EmergION {
	evidence := result.Evidence()

	sum := sha256.Sum256([]byte(evidence))
	sourceHash := hex.EncodeToString(sum[:])

	gap := "BRIDGEGAP:PIVOT"
	if name := bridgegapName(result.Name); name != "" {
		gap = "BRIDGEGAP:PIVOT_" + name
	}

	return core.EmergION{
		IDN: "E-" + strings.ToUpper(sourceHash[:16]),

		// Deliberately not admitted to GOV.
		STA: "",

		MEM: core.Memory{
			SourceHash: sourceHash,
			Codec:      "pivot",
			Bytes:      int64(len(evidence)),
			Stored:     int64(len(evidence)),
			Summary: fmt.Sprintf(
				"pivot divergence %s: %s",
				result.Name,
				result.Divergence,
			),
			Provenance: "reciprocal_pivot",
		},

		REL: map[string]string{
			"pivot":      result.Name,
			"forward":    result.Forward,
			"reciprocal": result.Reciprocal,
			"invariant":  result.Invariant,
		},

		CAP: []string{
			"OBS",
			"CMP",
			"VLD",
		},

		VAL: core.Validation{
			Facts: []string{
				"pivot_divergence_observed",
			},
			Gaps: []string{
				gap,
			},
			Risk:        "M",
			Recoil:      false,
			WVC:         false,
			Reasoner:    "reciprocal_pivot",
			ReasonerVer: "1",
		},

		EVO: core.Evolution{
			Version: 1,
		},
	}
}

// Observe executes the reciprocal observation for an already-produced
// forward result.
//
// Agreement returns a passing Result.
//
// Divergence fails closed and returns a DivergenceError containing the
// structured Result plus a non-authoritative EmergION generated from the
// divergence evidence.
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

		return result, &DivergenceError{
			Result:   result,
			EmergION: divergenceEmergION(result),
		}
	}

	if err := check(); err != nil {
		result.Divergence = err.Error()

		return result, &DivergenceError{
			Result:   result,
			EmergION: divergenceEmergION(result),
		}
	}

	result.Passed = true
	return result, nil
}
