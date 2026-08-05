package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/emerger"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/store"
)

type Runtime struct {
	Store    *store.Store
	Reasoner reason.Reasoner
}

func (r Runtime) Capture(ctx context.Context, path string, removeOnSuccess bool) (core.EmergION, bool, error) {
	if r.Store == nil || r.Reasoner == nil {
		return core.EmergION{}, false, fmt.Errorf("runtime not configured")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return core.EmergION{}, false, err
	}
	if len(b) == 0 {
		return core.EmergION{}, false, fmt.Errorf("empty source")
	}
	h := store.Hash(b)
	if existing, ok, err := r.Store.FindBySourceHash(h); err != nil {
		return core.EmergION{}, false, err
	} else if ok {
		if removeOnSuccess {
			_ = os.Remove(path)
		}
		return existing, true, nil
	}
	analysis, err := r.Reasoner.Analyze(ctx, reason.Input{Name: filepath.Base(path), Content: b})
	if err != nil {
		return core.EmergION{}, false, err
	}
	ev, err := r.Store.Preserve(b)
	if err != nil {
		return core.EmergION{}, false, err
	}
	// Use a fixed-result reasoner so the source is analyzed exactly once.
	fr := fixedReasoner{name: r.Reasoner.Name(), version: r.Reasoner.Version(ctx), result: analysis}
	em, err := (emerger.Engine{Reasoner: fr}).Emerge(ctx, reason.Input{Name: filepath.Base(path), Content: b}, emerger.Evidence{Hash: ev.Hash, Bytes: ev.Bytes, Stored: ev.Stored, Codec: ev.Codec, Provenance: "local_dropzone"})
	if err != nil {
		_, _ = r.Store.PruneOrphans()
		return core.EmergION{}, false, err
	}
	if _, err = r.Store.SaveCandidate(em); err != nil {
		_, _ = r.Store.PruneOrphans()
		return core.EmergION{}, false, err
	}
	if removeOnSuccess {
		if err = os.Remove(path); err != nil {
			return em, false, fmt.Errorf("captured but could not clear dropzone: %w", err)
		}
	}
	return em, false, nil
}

type fixedReasoner struct {
	name, version string
	result        reason.Result
}

func (f fixedReasoner) Analyze(context.Context, reason.Input) (reason.Result, error) {
	return f.result, nil
}
func (f fixedReasoner) Name() string                   { return f.name }
func (f fixedReasoner) Version(context.Context) string { return f.version }

func (r Runtime) Once(ctx context.Context, dropzone string) ([]string, error) {
	if err := os.MkdirAll(dropzone, 0o700); err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dropzone)
	if err != nil {
		return nil, err
	}
	sort.Slice(ents, func(i, j int) bool { return ents[i].Name() < ents[j].Name() })
	var ids []string
	for _, ent := range ents {
		if ent.IsDir() || strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		em, _, err := r.Capture(ctx, filepath.Join(dropzone, ent.Name()), true)
		if err != nil {
			return ids, fmt.Errorf("%s: %w", ent.Name(), err)
		}
		ids = append(ids, em.IDN)
	}
	return ids, nil
}

func (r Runtime) Run(ctx context.Context, dropzone string, interval time.Duration, onCapture func(string)) error {
	if interval < 250*time.Millisecond {
		interval = 250 * time.Millisecond
	}
	ids, err := r.Once(ctx, dropzone)
	if err != nil {
		return err
	}
	if onCapture != nil {
		for _, id := range ids {
			onCapture(id)
		}
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			ids, err := r.Once(ctx, dropzone)
			if err != nil {
				return err
			}
			if onCapture != nil {
				for _, id := range ids {
					onCapture(id)
				}
			}
		}
	}
}
