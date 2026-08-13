package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"emergion-sovereign-runtime/internal/adapters"
	"emergion-sovereign-runtime/internal/core"
	"emergion-sovereign-runtime/internal/emerger"
	livefield "emergion-sovereign-runtime/internal/field"
	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/store"
)

type Runtime struct {
	Store    *store.Store
	Reasoner reason.Reasoner
}

type governedProjection struct {
	ID            string            `json:"id"`
	Summary       string            `json:"summary"`
	Capabilities  []string          `json:"capabilities,omitempty"`
	Relationships map[string]string `json:"relationships,omitempty"`
	Topology      core.Topology     `json:"topology,omitempty"`
}

func (r Runtime) governedStateContext() (string, error) {
	events, err := r.Store.Events()
	if err != nil {
		return "", err
	}
	st, err := livefield.Rebuild(events)
	if err != nil {
		return "", err
	}
	items := make([]governedProjection, 0, len(st.Accepted))
	for _, em := range st.Accepted {
		item := governedProjection{
			ID:            em.IDN,
			Summary:       em.MEM.Summary,
			Capabilities:  em.CAP,
			Relationships: em.REL,
		}
		if em.EVO.Metadata != nil {
			item.Topology = em.EVO.Metadata.Topology
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) == 0 {
		return "", nil
	}
	bounded := make([]governedProjection, 0, len(items))
	for _, item := range items {
		candidate := append(bounded, item)
		b, err := json.Marshal(candidate)
		if err != nil {
			return "", err
		}
		if len(b) > 12000 {
			break
		}
		bounded = candidate
	}

	b, err := json.Marshal(bounded)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func coverage(em *core.EmergION, governedState string) error {
	var bridgegaps []string

	if strings.TrimSpace(em.MEM.SourceHash) == "" {
		bridgegaps = append(bridgegaps, "BRIDGEGAP:source_hash")
	}
	if em.MEM.Bytes <= 0 || em.MEM.Stored <= 0 {
		bridgegaps = append(bridgegaps, "BRIDGEGAP:evidence")
	}

	if strings.TrimSpace(em.MEM.Summary) == "" {
		bridgegaps = append(bridgegaps, "BRIDGEGAP:summary")
	}
	if len(em.VAL.Facts) == 0 {
		bridgegaps = append(bridgegaps, "BRIDGEGAP:facts")
	}
	if len(em.CAP) == 0 {
		bridgegaps = append(bridgegaps, "BRIDGEGAP:capabilities")
	}
	if len(em.REL) == 0 {
		bridgegaps = append(bridgegaps, "BRIDGEGAP:relationships")
	}

	if strings.TrimSpace(governedState) != "" {
		if em.REL["governed_state"] == "" {
			bridgegaps = append(bridgegaps, "BRIDGEGAP:living_state_relationship")
		}
		projected := false
		for _, fact := range em.VAL.Facts {
			if fact == "living_state_projected" {
				projected = true
				break
			}
		}
		if !projected {
			bridgegaps = append(bridgegaps, "BRIDGEGAP:living_state_projection")
		}
	}

	if len(bridgegaps) != 0 {
		em.VAL.Gaps = append(em.VAL.Gaps, bridgegaps...)
		return fmt.Errorf("COVERAGE failed: %s", strings.Join(bridgegaps, ", "))
	}

	return nil
}

func protector(em *core.EmergION) {
	if em.REL == nil {
		em.REL = map[string]string{}
	}

	authority := map[string]bool{}
	catalog := adapters.Catalog(false)

	for _, capability := range em.CAP {
		capability = strings.ToUpper(strings.TrimSpace(capability))
		for _, adapter := range catalog {
			for _, allowed := range adapter.Capabilities {
				if capability == allowed {
					authority[adapter.Authority] = true
				}
			}
		}
	}

	if len(authority) == 0 {
		em.REL["protector"] = "NO_EXTERNAL_AUTHORITY_CLAIMED"
		return
	}

	classes := make([]string, 0, len(authority))
	for class := range authority {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	em.REL["protector"] = strings.Join(classes, ",")

	if authority["SEND_GATED"] || authority["TRANSFER_GATED"] || authority["DEPLOY_GATED"] {
		em.REL["protector_gate"] = "HUMAN_FINAL_BOUND"
	}
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func deriveDelta(previous core.EmergION, analysis reason.Result) []string {
	var delta []string

	if strings.TrimSpace(previous.MEM.Summary) != strings.TrimSpace(analysis.Summary) {
		delta = append(delta, "SUMMARY_CHANGED")
	}

	previousCaps := stringSet(previous.CAP)
	currentCaps := stringSet(analysis.Capabilities)

	var addedCaps, removedCaps []string
	for capability := range currentCaps {
		if !previousCaps[capability] {
			addedCaps = append(addedCaps, capability)
		}
	}
	for capability := range previousCaps {
		if !currentCaps[capability] {
			removedCaps = append(removedCaps, capability)
		}
	}
	sort.Strings(addedCaps)
	sort.Strings(removedCaps)
	for _, capability := range addedCaps {
		delta = append(delta, "CAP_ADDED:"+capability)
	}
	for _, capability := range removedCaps {
		delta = append(delta, "CAP_REMOVED:"+capability)
	}

	previousRelationships := previous.REL
	if previousRelationships == nil {
		previousRelationships = map[string]string{}
	}
	currentRelationships := analysis.Relationships
	if currentRelationships == nil {
		currentRelationships = map[string]string{}
	}

	var relationshipKeys []string
	seenRelationships := map[string]bool{}
	for key := range previousRelationships {
		seenRelationships[key] = true
		relationshipKeys = append(relationshipKeys, key)
	}
	for key := range currentRelationships {
		if !seenRelationships[key] {
			relationshipKeys = append(relationshipKeys, key)
		}
	}
	sort.Strings(relationshipKeys)

	for _, key := range relationshipKeys {
		previousValue, previousOK := previousRelationships[key]
		currentValue, currentOK := currentRelationships[key]
		switch {
		case !previousOK && currentOK:
			delta = append(delta, "REL_ADDED:"+key)
		case previousOK && !currentOK:
			delta = append(delta, "REL_REMOVED:"+key)
		case previousOK && currentOK && previousValue != currentValue:
			delta = append(delta, "REL_CHANGED:"+key)
		}
	}

	previousFacets := map[string]bool{}
	if previous.EVO.Metadata != nil {
		for _, facet := range previous.EVO.Metadata.Facets {
			value := strings.TrimSpace(string(facet))
			if value != "" {
				previousFacets[value] = true
			}
		}
	}
	currentFacets := stringSet(analysis.Facets)

	var addedFacets, removedFacets []string
	for facet := range currentFacets {
		if !previousFacets[facet] {
			addedFacets = append(addedFacets, facet)
		}
	}
	for facet := range previousFacets {
		if !currentFacets[facet] {
			removedFacets = append(removedFacets, facet)
		}
	}
	sort.Strings(addedFacets)
	sort.Strings(removedFacets)
	for _, facet := range addedFacets {
		delta = append(delta, "FACET_ADDED:"+facet)
	}
	for _, facet := range removedFacets {
		delta = append(delta, "FACET_REMOVED:"+facet)
	}

	if len(delta) == 0 {
		return []string{"NO_STRUCTURAL_DELTA"}
	}
	return delta
}

func (r Runtime) validateLineage(analysis *reason.Result) error {
	if strings.TrimSpace(analysis.Supersedes) == "" {
		analysis.Supersedes = ""
		analysis.Delta = nil
		return nil
	}

	events, err := r.Store.Events()
	if err != nil {
		return err
	}
	st, err := livefield.Rebuild(events)
	if err != nil {
		return err
	}

	if previous, ok := st.Accepted[analysis.Supersedes]; !ok {
		return fmt.Errorf("lineage rejected: supersedes %q is not REG-accepted", analysis.Supersedes)
	} else {
		analysis.Delta = deriveDelta(previous, *analysis)
	}

	return nil
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
	governedState, err := r.governedStateContext()
	if err != nil {
		return core.EmergION{}, false, fmt.Errorf("living state projection failed: %w", err)
	}
	analysis, err := r.Reasoner.Analyze(ctx, reason.Input{Name: filepath.Base(path), Content: b, GovernedState: governedState})
	if err != nil {
		return core.EmergION{}, false, err
	}
	analysis = reason.Calibrate(analysis)
	if err := r.validateLineage(&analysis); err != nil {
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
	if err := coverage(&em, governedState); err != nil {
		_, _ = r.Store.PruneOrphans()
		return em, false, err
	}

	protector(&em)

	if strings.TrimSpace(em.MEM.Summary) == "" {
		_, _ = r.Store.PruneOrphans()
		return core.EmergION{}, false, fmt.Errorf("RECOIL failed: empty summary")
	}
	em.VAL.Recoil = true

	preserved, err := r.Store.ReadEvidence(ev.Hash)
	if err != nil {
		_, _ = r.Store.PruneOrphans()
		return core.EmergION{}, false, fmt.Errorf("WVC failed: %w", err)
	}
	if store.Hash(preserved) != em.MEM.SourceHash {
		_, _ = r.Store.PruneOrphans()
		return core.EmergION{}, false, fmt.Errorf("WVC failed: source hash mismatch")
	}
	em.VAL.WVC = true
	em.STA = core.StateAtGOV

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
