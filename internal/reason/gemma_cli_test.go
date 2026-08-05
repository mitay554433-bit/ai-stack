package reason

import "testing"

func TestParseResult(t *testing.T) {
	r, err := parseResult("noise {\"summary\":\"ok\",\"relationships\":{},\"capabilities\":[\"OBS\"],\"facts\":[\"x\"],\"gaps\":[],\"risk\":\"L\"} tail")
	if err != nil {
		t.Fatal(err)
	}
	if r.Summary != "ok" || r.Risk != "L" {
		t.Fatalf("bad result: %#v", r)
	}
}
