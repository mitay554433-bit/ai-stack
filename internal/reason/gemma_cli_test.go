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
