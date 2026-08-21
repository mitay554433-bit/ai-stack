// Package fieldapi exposes the serverless sovereign runtime to native shells.
// A desktop or mobile application can embed this package without invoking the CLI.
package fieldapi

import (
	"context"
	"fmt"

	"emergion-sovereign-runtime/internal/analytics"
	"emergion-sovereign-runtime/internal/core"
	livefield "emergion-sovereign-runtime/internal/field"
	"emergion-sovereign-runtime/internal/gov"
	"emergion-sovereign-runtime/internal/proj"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/reg"
	fieldruntime "emergion-sovereign-runtime/internal/runtime"
	"emergion-sovereign-runtime/internal/store"
)

type Runtime struct {
	store    *store.Store
	reasoner reason.Reasoner
}

func Open(stateRoot string, reasoner reason.Reasoner) (*Runtime, error) {
	if reasoner == nil {
		return nil, fmt.Errorf("reasoner required")
	}
	s, err := store.Open(stateRoot)
	if err != nil {
		return nil, err
	}
	return &Runtime{store: s, reasoner: reasoner}, nil
}
func (r *Runtime) Capture(ctx context.Context, path string) (core.EmergION, error) {
	em, _, err := (fieldruntime.Runtime{Store: r.store, Reasoner: r.reasoner}).Capture(ctx, path, false)
	return em, err
}
func (r *Runtime) state() (core.State, error) {
	ev, err := r.store.Events()
	if err != nil {
		return core.State{}, err
	}
	return livefield.Rebuild(ev)
}
func (r *Runtime) Status() (analytics.Metrics, error) {
	st, err := r.state()
	if err != nil {
		return analytics.Metrics{}, err
	}
	return analytics.Measure(st), nil
}
func (r *Runtime) Decide(id, decision, reasonText string) error {
	st, err := r.state()
	if err != nil {
		return err
	}
	em, ok := st.AtGOV[id]
	if !ok {
		return fmt.Errorf("candidate not at GOV")
	}
	em, receipt, err := gov.Decide(em, gov.Decision(decision), "HUMAN_FINAL", reasonText)
	if err != nil {
		return err
	}
	decisionID, err := r.store.SaveDecision(receipt)
	if err != nil {
		return err
	}
	if receipt.Decision == string(gov.Approve) {
		_, rr, err := reg.Accept(em, decisionID)
		if err != nil {
			return err
		}
		_, err = r.store.SaveAccepted(rr)
		return err
	}
	return nil
}
func (r *Runtime) Render(dir string) error {
	st, err := r.state()
	if err != nil {
		return err
	}
	_, err = proj.Current(dir, st)
	return err
}

func (r *Runtime) RenderCurrent(dir string) (proj.Receipt, error) {
	st, err := r.state()
	if err != nil {
		return proj.Receipt{}, err
	}
	return proj.Current(dir, st)
}

func (r *Runtime) CirculateSAWs(
	ctx context.Context,
) ([]core.EmergION, error) {
	st, err := r.state()
	if err != nil {
		return nil, err
	}

	sources, err := proj.SAWSources(st)
	if err != nil {
		return nil, err
	}

	runtime := fieldruntime.Runtime{
		Store:    r.store,
		Reasoner: r.reasoner,
	}

	out := make([]core.EmergION, 0, len(sources))

	for _, source := range sources {
		em, duplicate, err := runtime.CaptureBytes(
			ctx,
			source.ID,
			source.Content,
			"saw_projection",
		)
		if err != nil {
			return out, err
		}

		if duplicate {
			continue
		}

		out = append(out, em)
	}

	return out, nil
}
