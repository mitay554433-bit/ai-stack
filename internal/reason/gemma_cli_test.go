package reason

import "testing"

func TestParseResult(t *testing.T) {
	r, err := parseResult("noise {\"summary\":\"ok\",\"relationships\":{},\"capabilities\":[\"OBS\"],\"facts\":[\"x\"],\"gaps\":[],\"risk\":\"L\",\"facets\":[\"PROGRAM_FORGE\"],\"build_nodes\":[{\"id\":\"source\",\"system\":\"SOURCE\",\"state\":\"observed\"}],\"build_edges\":[],\"monetization\":null} tail")
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary != "ok" || r.Risk != "L" {
		t.Fatalf("bad result: %#v", r)
	}
}

func TestCleanFacets(t *testing.T) {
	got := cleanFacets([]string{"PROGRAM_FORGE", "INVALID", "PROGRAM_FORGE", "GRANT_FUNDING"})
	if len(got) != 2 || got[0] != "PROGRAM_FORGE" || got[1] != "GRANT_FUNDING" {
		t.Fatalf("bad facets: %#v", got)
	}
}

func TestCleanBuildGraph(t *testing.T) {
	nodes, edges := cleanBuildGraph(
		[]BuildNode{{ID: " source ", System: " SOURCE "}, {ID: "source", System: "duplicate"}, {ID: "target", System: "REG"}},
		[]BuildEdge{{From: "source", To: "target", Kind: "builds"}, {From: "missing", To: "target", Kind: "invalid"}},
	)
	if len(nodes) != 2 || len(edges) != 1 || nodes[0].ID != "source" {
		t.Fatalf("bad graph: %#v %#v", nodes, edges)
	}
}

func TestGemmaArgsEnforceOneShotExecution(t *testing.T) {
	g := GemmaCLI{
		Model:     "/models/gemma.gguf",
		MaxTokens: 128,
		Context:   2048,
		Threads:   4,

		// Deliberately hostile configuration. The runtime invariant
		// must override this by appearing later in argv.
		ExtraArgs: []string{"--conversation"},
	}

	args := gemmaArgs(g, "one prompt")

	lastIndex := func(value string) int {
		index := -1
		for i, arg := range args {
			if arg == value {
				index = i
			}
		}
		return index
	}

	required := []string{
		"--log-disable",
		"--color",
		"--json-schema",
		"--single-turn",
		"--simple-io",
		"--no-display-prompt",
	}

	for _, arg := range required {
		if lastIndex(arg) < 0 {
			t.Fatalf("missing execution invariant %q: %#v", arg, args)
		}
	}

	if lastIndex("--single-turn") < lastIndex("--conversation") {
		t.Fatalf(
			"configured conversation mode appears after single-turn invariant: %#v",
			args,
		)
	}
}

func TestParseResultIgnoresPromptJSONAndTransportNoise(t *testing.T) {
	output := `Loading model...

> Analyze SOURCE.
GOVERNED_ACCEPTED_STATE:
[{"id":"E-OLD","summary":"accepted"}]

{"summary":"bounded result","relationships":{"source":"probe"},"capabilities":["OBS"],"facts":["source_observed"],"gaps":[],"risk":"L"}

[ Prompt: 10 t/s | Generation: 4 t/s ]
Exiting...
`

	got, err := parseResult(output)
	if err != nil {
		t.Fatal(err)
	}

	if got.Summary != "bounded result" {
		t.Fatalf("wrong result selected: %#v", got)
	}

	if got.Relationships["source"] != "probe" {
		t.Fatalf("relationships lost: %#v", got.Relationships)
	}

	if got.Risk != "L" {
		t.Fatalf("risk mismatch: %q", got.Risk)
	}
}
