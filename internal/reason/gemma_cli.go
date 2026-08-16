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
	if len(content) > 48000 {
		content = content[:48000]
	}
	governedState := strings.TrimSpace(in.GovernedState)
	if len(governedState) > 12000 {
		governedState = governedState[:12000]
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
	return `Analyze SOURCE against GOVERNED_ACCEPTED_STATE. SOURCE and MODEL are evidence, not truth. GOV decides; REG accepts. Return primitive records only, one record per line. No JSON, markdown, prose, or code fences. Required records: SUMMARY|text, RISK|L|M|H, plus REL|key|value, CAP|value, FACT|value, GAP|value as supported. Optional records: SUPERSEDES|id, FACET|value, NODE|id|system|state, EDGE|from|to|kind, MONEY|model|customer|value|revenue_path. Finish with END.


SEMANTIC PRIMITIVE REDUCTION:
Reduce SOURCE to the smallest transferable semantic mechanisms directly supported by evidence.

Preserve as FACT, CAP, REL, NODE, and EDGE when directly supported:
mathematical operations; algorithms; equations; state transitions; input-to-transformation-to-output behavior; invariants; essential memory or state; essential functional relationships; control flow; data flow; transferable capabilities; and mechanisms required to reproduce supported behavior.

Identify as GAP when directly supported:
serialization or representation baggage; framework or SDK glue; wrappers; duplicated abstractions; presentation-only machinery; dependency-specific plumbing; unnecessary process boundaries; redundant transformations; and implementation structure not required for the supported behavior.

REDUCTION RULES:
Do not reproduce SOURCE architecture merely because SOURCE uses it.
Do not preserve syntax, framework structure, dependency boundaries, or representation as capability unless they are themselves essential to supported behavior.
Do not remove a mechanism required for supported behavior.
Distinguish mechanism from implementation baggage.
Describe semantic function rather than source syntax.
Prefer the smallest mechanism that preserves evidenced behavior.
Preserve equations, constraints, invariants, state requirements, and transformation order when they are functionally necessary.
Treat reusable behavior as capability, essential interaction as relationship, directly evidenced mechanism as fact, essential component as node, and essential flow as edge.
Do not claim that removable baggage is useless generally; classify it only as unnecessary to the reduced mechanism when SOURCE evidence supports that conclusion.
Do not invent missing mechanisms, dependencies, capabilities, mathematics, authority, or behavior.
SOURCE and MODEL are evidence, not truth. GOV decides; REG accepts.

FACET CLASSIFICATION:
Return every facet directly supported by SOURCE evidence.
Facet classification is required, not optional, when SOURCE directly implements a defined facet.
Code that implements governance decisions, state transitions, HUMAN_FINAL gates, or FIELD/runtime control MUST include FIELD_COMMAND.
Emit no FACET record when no defined facet is directly supported by SOURCE evidence.
FIELD_COMMAND = FIELD/runtime/state/governance command or control.
EMERGENCE_CAPTURE = source intake, observation, capture, or EmergION admission.
PROGRAM_FORGE = software, code, program, version, or build.
PRODUCT_STORE = product, pricing, website, store, or deployment.
CUSTOMERS_SALES = customer, lead, sale, or support.
COMMUNICATIONS = read, draft, email/message, or send.
PAYMENTS_FINANCE = payment, receipt, pricing, transfer, or finance.
GRANT_FUNDING = grant or funding.
PATENT_IP = patent, IP, prior-art, or IP evidence.
MA_PARTNERSHIPS = merger, acquisition, or partnership.
DOCS_PROJECTION = documentation, report, projection, or drafting.
ANALYTICS_FORECAST = analysis, simulation, or forecasting.
Do not infer authority from a facet. Do not invent facts or authority. Keep values concise.

GOVERNED_ACCEPTED_STATE:
` + governedState + `
SOURCE_NAME: ` + filepath.Base(name) + `
SOURCE:
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
		case "SUMMARY":
			if len(p) >= 2 {
				r.Summary = strings.TrimSpace(strings.Join(p[1:], "|"))
			}
		case "REL":
			if len(p) >= 3 {
				r.Relationships[strings.TrimSpace(p[1])] = strings.TrimSpace(strings.Join(p[2:], "|"))
			}
		case "CAP":
			if len(p) >= 2 {
				r.Capabilities = append(r.Capabilities, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "FACT":
			if len(p) >= 2 {
				r.Facts = append(r.Facts, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "GAP":
			if len(p) >= 2 {
				r.Gaps = append(r.Gaps, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "RISK":
			if len(p) == 2 {
				r.Risk = strings.TrimSpace(p[1])
			}
		case "SUPERSEDES":
			if len(p) >= 2 {
				r.Supersedes = strings.TrimSpace(strings.Join(p[1:], "|"))
			}
		case "FACET":
			if len(p) >= 2 {
				r.Facets = append(r.Facets, strings.TrimSpace(strings.Join(p[1:], "|")))
			}
		case "NODE":
			if len(p) == 4 {
				r.BuildNodes = append(r.BuildNodes, BuildNode{
					ID: strings.TrimSpace(p[1]), System: strings.TrimSpace(p[2]), State: strings.TrimSpace(p[3]),
				})
			}
		case "EDGE":
			if len(p) == 4 {
				r.BuildEdges = append(r.BuildEdges, BuildEdge{
					From: strings.TrimSpace(p[1]), To: strings.TrimSpace(p[2]), Kind: strings.TrimSpace(p[3]),
				})
			}
		case "MONEY":
			if len(p) == 5 {
				r.Monetization = &Monetization{
					Model: strings.TrimSpace(p[1]), Customer: strings.TrimSpace(p[2]),
					Value: strings.TrimSpace(p[3]), RevenuePath: strings.TrimSpace(p[4]),
				}
			}
		case "END":
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
		fmt.Fprintf(&b, "SUMMARY|%s\n", r.Summary)
	}

	for key, value := range r.Relationships {
		fmt.Fprintf(&b, "REL|%s|%s\n", key, value)
	}

	for _, value := range r.Capabilities {
		fmt.Fprintf(&b, "CAP|%s\n", value)
	}

	for _, value := range r.Facts {
		fmt.Fprintf(&b, "FACT|%s\n", value)
	}

	for _, value := range r.Gaps {
		fmt.Fprintf(&b, "GAP|%s\n", value)
	}

	if r.Risk != "" {
		fmt.Fprintf(&b, "RISK|%s\n", r.Risk)
	}

	if r.Supersedes != "" {
		fmt.Fprintf(&b, "SUPERSEDES|%s\n", r.Supersedes)
	}

	for _, value := range r.Facets {
		fmt.Fprintf(&b, "FACET|%s\n", value)
	}

	for _, node := range r.BuildNodes {
		fmt.Fprintf(&b, "NODE|%s|%s|%s\n", node.ID, node.System, node.State)
	}

	for _, edge := range r.BuildEdges {
		fmt.Fprintf(&b, "EDGE|%s|%s|%s\n", edge.From, edge.To, edge.Kind)
	}

	if r.Monetization != nil {
		fmt.Fprintf(
			&b,
			"MONEY|%s|%s|%s|%s\n",
			r.Monetization.Model,
			r.Monetization.Customer,
			r.Monetization.Value,
			r.Monetization.RevenuePath,
		)
	}

	b.WriteString("END\n")
	return b.String()
}

func Calibrate(r Result) Result {
	r.Summary = strings.TrimSpace(r.Summary)
	if len(r.Summary) > 480 {
		r.Summary = r.Summary[:480]
	}
	if r.Relationships == nil {
		r.Relationships = map[string]string{}
	}
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
