package analytics

import (
	"emergion-sovereign-runtime/internal/core"
	"runtime"
	"time"
)

type Metrics struct {
	AtGOV      int       `json:"at_gov"`
	Approved   int       `json:"approved"`
	Accepted   int       `json:"accepted"`
	Held       int       `json:"held"`
	Rejected   int       `json:"rejected"`
	Returned   int       `json:"returned"`
	Events     int       `json:"events"`
	TipHash    string    `json:"tip_hash"`
	CPU        int       `json:"cpu"`
	GoRoutines int       `json:"goroutines"`
	MeasuredAt time.Time `json:"measured_at"`
}

func Measure(st core.State) Metrics {
	return Metrics{AtGOV: len(st.AtGOV), Approved: len(st.Approved), Accepted: len(st.Accepted), Held: len(st.Held), Rejected: len(st.Rejected), Returned: len(st.Returned), Events: st.Events, TipHash: st.TipHash, CPU: runtime.NumCPU(), GoRoutines: runtime.NumGoroutine(), MeasuredAt: time.Now().UTC()}
}
