package reason

import "context"

type Input struct {
	Name    string
	Content []byte
}

type Result struct {
	Summary       string            `json:"summary"`
	Relationships map[string]string `json:"relationships"`
	Capabilities  []string          `json:"capabilities"`
	Facts         []string          `json:"facts"`
	Gaps          []string          `json:"gaps"`
	Risk          string            `json:"risk"`
}

type Reasoner interface {
	Analyze(context.Context, Input) (Result, error)
	Name() string
	Version(context.Context) string
}
