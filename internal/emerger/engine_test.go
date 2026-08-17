package emerger

import (
	"context"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/pivot"
	"emergion-sovereign-runtime/internal/reason"
	"errors"
	"strings"
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

type maximalReasoner struct{}

func (maximalReasoner) Analyze(context.Context, reason.Input) (reason.Result, error) {
	return reason.Result{
		Summary:  "maximal semantic proposal",
		Archonym: "MAXIMAL PROPOSAL",
		Relationships: map[string]string{
			"source_name": "claimed-source",
			"source_kind": "PROGRAM",
		},
		Capabilities: []string{
			"OBS",
			"CMP",
		},
		Facts: []string{
			"semantic proposal present",
		},
		Gaps: []string{
			"bounded_gap",
		},
		Risk:       "H",
		Supersedes: "",
		Facets: []string{
			"EMERGENCE_CAPTURE",
			"PROGRAM_FORGE",
		},
		BuildNodes: []reason.BuildNode{
			{
				ID:     "source",
				System: "SOURCE",
				State:  "observed",
			},
			{
				ID:     "target",
				System: "TARGET",
				State:  "proposed",
			},
		},
		BuildEdges: []reason.BuildEdge{
			{
				From: "source",
				To:   "target",
				Kind: "TRANSITION",
			},
		},
		Monetization: &reason.Monetization{
			Model:       "license",
			Customer:    "bounded customer",
			Value:       "governed capability",
			RevenuePath: "approved delivery",
		},
	}, nil
}

func (maximalReasoner) Name() string {
	return "maximal-test"
}

func (maximalReasoner) Version(context.Context) string {
	return "1"
}

func TestEmergerCannotSelfAuthorizeOrSelfVerify(t *testing.T) {
	source := []byte("emergER confinement proof")
	hash := SourceHash(source)

	em, err := (Engine{
		Reasoner: maximalReasoner{},
	}).Emerge(
		context.Background(),
		reason.Input{
			Name:    "maximal.txt",
			Content: source,
		},
		Evidence{
			Hash:       hash,
			Bytes:      int64(len(source)),
			Stored:     int64(len(source)),
			Codec:      "raw",
			Provenance: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if em.STA != "" {
		t.Fatalf("EmergER self-authorized state = %q", em.STA)
	}

	if em.VAL.Recoil {
		t.Fatal("EmergER self-asserted RECOIL verification")
	}

	if em.VAL.WVC {
		t.Fatal("EmergER self-asserted WVC verification")
	}

	if em.EVO.Version != 1 {
		t.Fatalf("EmergER evolution version = %d want 1", em.EVO.Version)
	}

	if em.MEM.SourceHash != hash {
		t.Fatalf(
			"EmergER changed source identity: got %q want %q",
			em.MEM.SourceHash,
			hash,
		)
	}

	expectedID := "E-" + strings.ToUpper(hash[:16])
	if em.IDN != expectedID {
		t.Fatalf(
			"EmergER identity = %q want %q",
			em.IDN,
			expectedID,
		)
	}

	if em.VAL.Reasoner != "maximal-test" {
		t.Fatalf("reasoner attribution = %q", em.VAL.Reasoner)
	}

	if em.EVO.Metadata == nil {
		t.Fatal("EmergER metadata missing")
	}

	if !em.EVO.Metadata.AIIntegrated {
		t.Fatal("AI-integrated EmergER output not marked AI-integrated")
	}

	if em.EVO.Metadata.PromptSchema != "MXPD/2" {
		t.Fatalf(
			"prompt schema = %q want MXPD/2",
			em.EVO.Metadata.PromptSchema,
		)
	}

	if em.EVO.Metadata.Archonym != "MAXIMAL PROPOSAL" {
		t.Fatalf("Archonym = %q want MAXIMAL PROPOSAL", em.EVO.Metadata.Archonym)
	}
}
