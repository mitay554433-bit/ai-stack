package reason

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type GemmaCLI struct {
	Binary    string
	Model     string
	Threads   int
	Context   int
	MaxTokens int
	Timeout   time.Duration
	ExtraArgs []string
}

func GemmaFromEnv() GemmaCLI {
	threads := envInt("GEMMA_THREADS", 4)
	ctx := envInt("GEMMA_CONTEXT", 4096)
	max := envInt("GEMMA_MAX_TOKENS", 256)
	timeout := time.Duration(envInt("GEMMA_TIMEOUT_SECONDS", 180)) * time.Second
	extra := strings.Fields(os.Getenv("GEMMA_EXTRA_ARGS"))
	return GemmaCLI{
		Binary:    discoverBinary(os.Getenv("GEMMA_BIN")),
		Model:     discoverModel(os.Getenv("GEMMA_MODEL")),
		Threads:   threads,
		Context:   ctx,
		MaxTokens: max,
		Timeout:   timeout,
		ExtraArgs: extra,
	}
}

func envInt(name string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(name))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
func discoverBinary(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if p, err := exec.LookPath("llama-cli"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(home, "llama.cpp", "build", "bin", "llama-cli"),
		filepath.Join(home, "bin", "llama-cli"),
		filepath.Join(home, "llama-cli"),
		"/usr/local/bin/llama-cli",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return "llama-cli"
}

type modelCandidate struct {
	path     string
	score    int
	modified time.Time
}

func discoverModel(explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, "models"), filepath.Join(home, "gguf"),
		filepath.Join(home, ".cache", "llama.cpp"), filepath.Join(home, ".local", "share", "models"),
		"models", "/opt/models",
	}
	if extra := os.Getenv("GEMMA_MODEL_DIRS"); extra != "" {
		dirs = append(strings.Split(extra, string(os.PathListSeparator)), dirs...)
	}
	var found []modelCandidate
	for _, root := range dirs {
		root = filepath.Clean(root)
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				rel, _ := filepath.Rel(root, path)
				if rel != "." && strings.Count(rel, string(os.PathSeparator)) >= 2 {
					return filepath.SkipDir
				}
				return nil
			}
			name := strings.ToLower(d.Name())
			if !strings.HasSuffix(name, ".gguf") || !strings.Contains(name, "gemma") {
				return nil
			}
			score := 100
			if strings.Contains(name, "instruct") || strings.Contains(name, "-it-") || strings.Contains(name, "_it_") {
				score += 20
			}
			if strings.Contains(name, "q4_k_m") || strings.Contains(name, "q5_k_m") {
				score += 10
			}
			info, _ := d.Info()
			mod := time.Time{}
			if info != nil {
				mod = info.ModTime()
			}
			found = append(found, modelCandidate{path: path, score: score, modified: mod})
			return nil
		})
	}
	if len(found) == 0 {
		return ""
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].modified.After(found[j].modified)
	})
	return found[0].path
}

func (g GemmaCLI) Name() string { return "gemma-llama-cli" }

func (g GemmaCLI) Version(ctx context.Context) string {
	if g.Binary == "" {
		return "unknown"
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, g.Binary, "--version").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	v := strings.TrimSpace(string(out))
	if len(v) > 120 {
		v = v[:120]
	}
	return v
}

func (g GemmaCLI) Validate() error {
	if g.Model == "" {
		return fmt.Errorf("GEMMA_MODEL is not set")
	}
	if _, err := os.Stat(g.Model); err != nil {
		return fmt.Errorf("Gemma model unavailable: %w", err)
	}
	if _, err := exec.LookPath(g.Binary); err != nil {
		return fmt.Errorf("Gemma runtime unavailable: %w", err)
	}
	return nil
}

func gemmaArgs(g GemmaCLI, prompt string) []string {
	args := []string{
		"-m", g.Model,
		"-p", prompt,
		"-n", strconv.Itoa(g.MaxTokens),
		"-c", strconv.Itoa(g.Context),
		"-t", strconv.Itoa(g.Threads),
		"--temp", "0.1",
	}

	args = append(args, g.ExtraArgs...)

	// Execution-boundary invariants. These intentionally come last.
	args = append(args,
		"--log-disable",
		"--color", "off",
		"--single-turn",
		"--simple-io",
		"--no-display-prompt",
	)

	return args
}

func (g GemmaCLI) Analyze(ctx context.Context, in Input) (Result, error) {
	if err := g.Validate(); err != nil {
		return Result{}, err
	}
	if len(in.Content) == 0 {
		return Result{}, fmt.Errorf("empty input")
	}
	content := string(in.Content)
	governedState := strings.TrimSpace(in.GovernedState)

	inputTokens := g.Context - g.MaxTokens - 1024
	if inputTokens < 256 {
		return Result{}, fmt.Errorf("Gemma context too small for bounded analysis")
	}

	inputBytes := inputTokens * 3
	governedLimit := inputBytes / 4
	if len(governedState) > governedLimit {
		governedState = governedState[:governedLimit]
	}

	contentLimit := inputBytes - len(governedState)
	if contentLimit < 1 {
		return Result{}, fmt.Errorf("Gemma context exhausted by governed state")
	}
	if len(content) > contentLimit {
		content = content[:contentLimit]
	}
	prompt := buildPrompt(in.Name, content, governedState)
	args := gemmaArgs(g, prompt)
	cctx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, g.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if cctx.Err() != nil {
			return Result{}, fmt.Errorf("Gemma timed out: %w", cctx.Err())
		}
		return Result{}, fmt.Errorf("Gemma failed: %w: %s", err, trim(stderr.String(), 500))
	}
	res, err := parseResult(stdout.String())
	if err == nil {
		return res, nil
	}

	if alt, altErr := parseResult(stderr.String()); altErr == nil {
		return alt, nil
	}

	combined := stdout.String() + "\n" + stderr.String()
	if alt, altErr := parseResult(combined); altErr == nil {
		return alt, nil
	}

	return Result{}, fmt.Errorf(
		"Gemma output invalid: %w; stdout=%s; stderr=%s",
		err,
		trim(stdout.String(), 500),
		trim(stderr.String(), 500),
	)
}

func buildPrompt(name, content, governedState string) string {
	return `@L:MXPD/2
@T:REDUCE
AX[S!=T;M!=T;GOV>D;REG>A;NI;FC]
OUT[S|x;H|archonym;K|L|M|H;L|k|v;C|x;F|x;G|x;U|id;T|facet;N|id|system|state;E|from|to|kind;M|model|customer|value|revenue_path;Z]
SEM[F=required_mechanism|algorithm|equation|invariant|state_transition;C=transferable_behavior;L=essential_relation;G=missing_mechanism|nonessential_baggage;N=component;E=flow]
KEEP[math,state,order,constraints,input,transform,output]
DROP_AS_G_ONLY_IF_NONESSENTIAL[wrapper,serialization,SDK,framework,presentation,duplication,dependency_plumbing]
FACET[FIELD_COMMAND,EMERGENCE_CAPTURE,PROGRAM_FORGE,PRODUCT_STORE,CUSTOMERS_SALES,COMMUNICATIONS,PAYMENTS_FINANCE,GRANT_FUNDING,PATENT_IP,MA_PARTNERSHIPS,DOCS_PROJECTION,ANALYTICS_FORECAST]
RULE[FIELD_COMMAND=runtime|state|governance_control;NO_INVENT;NO_AUTH_INFER;Z_REQUIRED]

OUTPUT CONTRACT:
- Output primitive lines only.
- Use the vertical pipe character | as the delimiter.
- Never replace | with a colon.
- S| MUST contain a non-empty one-sentence summary.
- H| MAY contain one canonical semantic Archonym; omit it when unsupported.
- K| MUST be exactly K|L, K|M, or K|H.
- F| contains an evidenced fact.
- C| contains a transferable capability.
- G| contains an evidenced gap when present.
- L|key|value contains an evidenced relationship when present.
- Z MUST be the final line.
- No markdown.
- No prose outside primitive lines.

RISK:
K|L = bounded/local/no evidenced harmful authority
K|M = uncertainty, missing evidence, or meaningful operational risk
K|H = external authority, destructive action, transfer, deployment, or serious evidenced risk

MINIMUM VALID SHAPE:
S|non-empty summary
K|L
F|evidenced fact
C|transferable capability
Z

:
` + governedState + `
:` + filepath.Base(name) + `
` + content
}
func parseResult(s string) (Result, error) {
	r := Result{Relationships: map[string]string{}}
	complete := false

	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		p := strings.Split(line, "|")
		switch p[0] {
		case "SUMMARY", "S":
			if len(p) >= 2 {
				r.Summary = strings.TrimSpace(strings.Join(p[1:], "|"))
			}
		case "REL", "L":
			if len(p) >= 3 {
				r.Relationships[strings.TrimSpace(p[1])] = strings.TrimSpace(strings.Join(p[2:], "|"))
			}
		case "CAP", "C":
			if len(p) >= 2 {
				r.Capabilities = append(r.Capabilities, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "FACT", "F":
			if len(p) >= 2 {
				r.Facts = append(r.Facts, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "GAP", "G":
			if len(p) >= 2 {
				r.Gaps = append(r.Gaps, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "RISK", "K":
			if len(p) == 2 {
				r.Risk = strings.TrimSpace(p[1])
			}
		case "SUPERSEDES", "U":
			if len(p) >= 2 {
				r.Supersedes = strings.TrimSpace(strings.Join(p[1:], "|"))
			}
		case "FACET", "T":
			if len(p) >= 2 {
				r.Facets = append(r.Facets, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "NODE", "N":
			if len(p) == 4 {
				r.BuildNodes = append(r.BuildNodes, BuildNode{
					ID: strings.TrimSpace(p[1]), System: strings.TrimSpace(p[2]), State: strings.TrimSpace(p[3]),
				})
			}
		case "EDGE", "E":
			if len(p) == 4 {
				r.BuildEdges = append(r.BuildEdges, BuildEdge{
					From: strings.TrimSpace(p[1]), To: strings.TrimSpace(p[2]), Kind: strings.TrimSpace(p[3]),
				})
			}
		case "ARCHONYM", "H":
			if len(p) == 2 {
				r.Archonym = strings.TrimSpace(p[1])
			}
		case "MONEY", "M":
			if len(p) == 5 {
				r.Monetization = &Monetization{
					Model: strings.TrimSpace(p[1]), Customer: strings.TrimSpace(p[2]),
					Value: strings.TrimSpace(p[3]), RevenuePath: strings.TrimSpace(p[4]),
				}
			}
		case "END", "Z":
			complete = true
		}
	}

	if !complete || r.Summary == "" || r.Risk == "" {
		return Result{}, fmt.Errorf("no complete primitive Result")
	}
	return r, nil
}

func FormatResult(r Result) string {
	var b strings.Builder

	if r.Summary != "" {
		fmt.Fprintf(&b, "S|%s\n", r.Summary)
	}

	if r.Archonym != "" {
		fmt.Fprintf(&b, "H|%s\n", r.Archonym)
	}

	for key, value := range r.Relationships {
		fmt.Fprintf(&b, "L|%s|%s\n", key, value)
	}

	for _, value := range r.Capabilities {
		fmt.Fprintf(&b, "C|%s\n", value)
	}

	for _, value := range r.Facts {
		fmt.Fprintf(&b, "F|%s\n", value)
	}

	for _, value := range r.Gaps {
		fmt.Fprintf(&b, "G|%s\n", value)
	}

	if r.Risk != "" {
		fmt.Fprintf(&b, "K|%s\n", r.Risk)
	}

	if r.Supersedes != "" {
		fmt.Fprintf(&b, "U|%s\n", r.Supersedes)
	}

	for _, value := range r.Facets {
		fmt.Fprintf(&b, "T|%s\n", value)
	}

	for _, node := range r.BuildNodes {
		fmt.Fprintf(&b, "N|%s|%s|%s\n", node.ID, node.System, node.State)
	}

	for _, edge := range r.BuildEdges {
		fmt.Fprintf(&b, "E|%s|%s|%s\n", edge.From, edge.To, edge.Kind)
	}

	if r.Monetization != nil {
		fmt.Fprintf(
			&b,
			"M|%s|%s|%s|%s\n",
			r.Monetization.Model,
			r.Monetization.Customer,
			r.Monetization.Value,
			r.Monetization.RevenuePath,
		)
	}

	b.WriteString("Z\n")
	return b.String()
}

func Calibrate(r Result) Result {
	r.Summary = strings.TrimSpace(r.Summary)
	r.Archonym = cleanText(r.Archonym, 160)
	if strings.EqualFold(r.Archonym, "null") {
		r.Archonym = ""
	}
	if len(r.Summary) > 480 {
		r.Summary = r.Summary[:480]
	}
	r.Relationships = cleanRelationships(r.Relationships)
	if r.Risk != "L" && r.Risk != "M" && r.Risk != "H" {
		r.Risk = "M"
	}
	r.Capabilities = clean(r.Capabilities, 16)
	r.Facts = clean(r.Facts, 24)
	r.Gaps = clean(r.Gaps, 24)
	r.Supersedes = cleanText(r.Supersedes, 80)
	if strings.EqualFold(r.Supersedes, "null") {
		r.Supersedes = ""
	}
	r.Delta = nil
	r.Facets = cleanFacets(r.Facets)
	r.BuildNodes, r.BuildEdges = cleanBuildGraph(r.BuildNodes, r.BuildEdges)
	if r.Monetization != nil {
		r.Monetization.Model = cleanText(r.Monetization.Model, 160)
		r.Monetization.Customer = cleanText(r.Monetization.Customer, 160)
		r.Monetization.Value = cleanText(r.Monetization.Value, 240)
		r.Monetization.RevenuePath = cleanText(r.Monetization.RevenuePath, 240)
		if r.Monetization.Model == "" && r.Monetization.Customer == "" && r.Monetization.Value == "" && r.Monetization.RevenuePath == "" {
			r.Monetization = nil
		}
	}
	return r
}

func cleanRelationships(in map[string]string) map[string]string {
	out := map[string]string{}
	if in == nil {
		return out
	}

	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		cleanKey := strings.TrimSpace(key)
		cleanValue := strings.TrimSpace(in[key])
		if cleanKey == "" || cleanValue == "" {
			continue
		}
		if _, exists := out[cleanKey]; exists {
			continue
		}
		out[cleanKey] = cleanValue
	}

	return out
}

func cleanBuildGraph(nodes []BuildNode, edges []BuildEdge) ([]BuildNode, []BuildEdge) {
	cleanNodes := make([]BuildNode, 0, 24)
	known := map[string]bool{}
	for _, node := range nodes {
		node.ID = cleanText(node.ID, 80)
		node.System = cleanText(node.System, 160)
		node.State = cleanText(node.State, 80)
		if node.ID == "" || node.System == "" || known[node.ID] {
			continue
		}
		known[node.ID] = true
		cleanNodes = append(cleanNodes, node)
		if len(cleanNodes) == 24 {
			break
		}
	}
	cleanEdges := make([]BuildEdge, 0, 48)
	seen := map[string]bool{}
	for _, edge := range edges {
		edge.From = cleanText(edge.From, 80)
		edge.To = cleanText(edge.To, 80)
		edge.Kind = cleanText(edge.Kind, 80)
		key := edge.From + "\x00" + edge.To + "\x00" + edge.Kind
		if !known[edge.From] || !known[edge.To] || seen[key] {
			continue
		}
		seen[key] = true
		cleanEdges = append(cleanEdges, edge)
		if len(cleanEdges) == 48 {
			break
		}
	}
	return cleanNodes, cleanEdges
}

func cleanText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		value = value[:max]
	}
	return value
}

func cleanFacets(in []string) []string {
	allowed := map[string]bool{
		"FIELD_COMMAND": true, "EMERGENCE_CAPTURE": true, "PROGRAM_FORGE": true,
		"PRODUCT_STORE": true, "CUSTOMERS_SALES": true, "COMMUNICATIONS": true,
		"PAYMENTS_FINANCE": true, "GRANT_FUNDING": true, "PATENT_IP": true,
		"MA_PARTNERSHIPS": true, "DOCS_PROJECTION": true, "ANALYTICS_FORECAST": true,
	}
	out := make([]string, 0, 12)
	seen := map[string]bool{}
	for _, facet := range in {
		facet = strings.TrimSpace(facet)
		if allowed[facet] && !seen[facet] {
			seen[facet] = true
			out = append(out, facet)
		}
	}
	return out
}
func clean(in []string, max int) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		if len(v) > 96 {
			v = v[:96]
		}
		seen[v] = true
		out = append(out, v)
		if len(out) == max {
			break
		}
	}
	return out
}
func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
