package emerger

import (
	"context"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/pivot"
	"emergion-sovereign-runtime/internal/reason"
	"errors"
	"testing"
)

func TestEmerge(t *testing.T) {
	b := []byte("hello field")
	h := SourceHash(b)
	em, err := (Engine{Reasoner: reason.Heuristic{}}).Emerge(context.Background(), reason.Input{Name: "x", Content: b}, Evidence{Hash: h, Bytes: int64(len(b)), Codec: "gzip", Stored: 10})
	if err != nil {
		t.Fatal(err)
	}
	if em.STA != "" || em.VAL.Recoil || em.VAL.WVC {
		t.Fatalf("bad: %#v", em)
	}
	if em.EVO.Metadata == nil || em.EVO.Metadata.CapturedAt.IsZero() || em.EVO.Metadata.AIIntegrated {
		t.Fatalf("bad metadata: %#v", em.EVO.Metadata)
	}
	if em.EVO.Metadata.Topology != core.TopologyDodecahedronV1 {
		t.Fatalf("bad topology: %q", em.EVO.Metadata.Topology)
	}
}

type emptySummaryReasoner struct{}

func (emptySummaryReasoner) Analyze(context.Context, reason.Input) (reason.Result, error) {
	return reason.Result{}, nil
}

func (emptySummaryReasoner) Name() string {
	return "empty-summary-test"
}

func (emptySummaryReasoner) Version(context.Context) string {
	return "1"
}

func TestEmptySummaryEmergesAsPivotDivergence(t *testing.T) {
	b := []byte("source whose analysis has no summary")

	_, err := (Engine{
		Reasoner: emptySummaryReasoner{},
	}).Emerge(
		context.Background(),
		reason.Input{
			Name:    "empty.txt",
			Content: b,
		},
		Evidence{
			Hash:   SourceHash(b),
			Bytes:  int64(len(b)),
			Stored: int64(len(b)),
			Codec:  "raw",
		},
	)

	if err == nil {
		t.Fatal("expected reciprocal divergence")
	}

	var divergence *pivot.DivergenceError
	if !errors.As(err, &divergence) {
		t.Fatalf("error type = %T", err)
	}

	if divergence.EmergION.STA != "" {
		t.Fatalf(
			"divergence self-authorized state = %q",
			divergence.EmergION.STA,
		)
	}

	if divergence.EmergION.VAL.Recoil ||
		divergence.EmergION.VAL.WVC {
		t.Fatal("raw divergence was incorrectly verified")
	}

	if len(divergence.EmergION.VAL.Gaps) != 1 ||
		divergence.EmergION.VAL.Gaps[0] !=
			"BRIDGEGAP:PIVOT_EMERGER_RECOIL" {
		t.Fatalf(
			"unexpected gap: %#v",
			divergence.EmergION.VAL.Gaps,
		)
	}
}
