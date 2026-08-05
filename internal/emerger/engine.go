package emerger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/reason"
)

type Evidence struct {
	Hash       string
	Bytes      int64
	Stored     int64
	Codec      string
	Provenance string
}

type Engine struct{ Reasoner reason.Reasoner }

func SourceHash(content []byte) string {
	s := sha256.Sum256(content)
	return hex.EncodeToString(s[:])
}

func (e Engine) Emerge(ctx context.Context, in reason.Input, ev Evidence) (core.EmergION, error) {
	if e.Reasoner == nil {
		return core.EmergION{}, fmt.Errorf("reasoner required")
	}
	if len(in.Content) == 0 {
		return core.EmergION{}, fmt.Errorf("empty source")
	}
	if ev.Hash == "" {
		ev.Hash = SourceHash(in.Content)
	}
	if ev.Bytes == 0 {
		ev.Bytes = int64(len(in.Content))
	}
	result, err := e.Reasoner.Analyze(ctx, in)
	if err != nil {
		return core.EmergION{}, err
	}
	if strings.TrimSpace(result.Summary) == "" {
		return core.EmergION{}, fmt.Errorf("RECOIL failed: empty summary")
	}
	id := "E-" + strings.ToUpper(ev.Hash[:16])
	em := core.EmergION{
		IDN: id,
		STA: core.StateAtGOV,
		MEM: core.Memory{SourceHash: ev.Hash, Codec: ev.Codec, Bytes: ev.Bytes, Stored: ev.Stored, Summary: result.Summary, Provenance: ev.Provenance},
		REL: result.Relationships,
		CAP: result.Capabilities,
		VAL: core.Validation{Facts: result.Facts, Gaps: result.Gaps, Risk: result.Risk, Recoil: true, WVC: true, Reasoner: e.Reasoner.Name(), ReasonerVer: e.Reasoner.Version(ctx)},
		EVO: core.Evolution{Version: 1},
	}
	return em, nil
}
