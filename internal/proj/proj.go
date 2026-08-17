package proj

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"emergion-sovereign-runtime/internal/core"
	livefield "emergion-sovereign-runtime/internal/field"
)

func JSON(path string, st core.State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

type row struct {
	ID      string
	State   string
	Summary string
	Risk    string
}

func rows(st core.State) []row {
	var out []row
	add := func(m map[string]core.EmergION) {
		for _, e := range m {
			out = append(out, row{
				ID:      e.IDN,
				State:   e.STA,
				Summary: e.MEM.Summary,
				Risk:    e.VAL.Risk,
			})
		}
	}

	add(st.AtGOV)
	add(st.Approved)
	add(st.Accepted)
	add(st.Held)
	add(st.Rejected)
	add(st.Returned)

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})

	return out
}

type convergenceRow struct {
	ID            string
	Archonym      string
	Kin           string
	Summary       string
	Topology      string
	Facets        string
	Capabilities  string
	Relationships string
	BuildNodes    string
	BuildEdges    string
}

func convergenceRows(st core.State) ([]convergenceRow, error) {
	out := make([]convergenceRow, 0, len(st.Accepted))

	children := map[string][]string{}
	for _, e := range st.Accepted {
		predecessor := strings.TrimSpace(e.EVO.Supersedes)
		if predecessor == "" {
			continue
		}
		if _, accepted := st.Accepted[predecessor]; !accepted {
			continue
		}
		children[predecessor] = append(children[predecessor], e.IDN)
	}
	for predecessor := range children {
		sort.Strings(children[predecessor])
	}

	for _, e := range st.Accepted {
		row := convergenceRow{
			ID:           e.IDN,
			Summary:      e.MEM.Summary,
			Capabilities: strings.Join(e.CAP, ", "),
		}

		kin := []string{}
		root, err := livefield.AcceptedKinRoot(st.Accepted, e.IDN)
		if err != nil {
			return nil, err
		}
		kin = append(kin, "root → "+root)
		predecessor := strings.TrimSpace(e.EVO.Supersedes)
		if predecessor != "" {
			if _, accepted := st.Accepted[predecessor]; accepted {
				kin = append(kin, "predecessor → "+predecessor)
			}
		}
		for _, descendant := range children[e.IDN] {
			kin = append(kin, "descendant → "+descendant)
		}
		row.Kin = strings.Join(kin, "; ")

		if e.EVO.Metadata != nil {
			row.Archonym = e.EVO.Metadata.Archonym
			row.Topology = string(e.EVO.Metadata.Topology)

			facets := make([]string, 0, len(e.EVO.Metadata.Facets))
			for _, facet := range e.EVO.Metadata.Facets {
				facets = append(facets, string(facet))
			}
			row.Facets = strings.Join(facets, ", ")

			nodes := make([]string, 0, len(e.EVO.Metadata.BuildNodes))
			for _, node := range e.EVO.Metadata.BuildNodes {
				value := node.ID + " → " + node.System
				if node.State != "" {
					value += " [" + node.State + "]"
				}
				nodes = append(nodes, value)
			}
			row.BuildNodes = strings.Join(nodes, "; ")

			edges := make([]string, 0, len(e.EVO.Metadata.BuildEdges))
			for _, edge := range e.EVO.Metadata.BuildEdges {
				value := edge.From + " → " + edge.To
				if edge.Kind != "" {
					value += " [" + edge.Kind + "]"
				}
				edges = append(edges, value)
			}
			row.BuildEdges = strings.Join(edges, "; ")
		}

		keys := make([]string, 0, len(e.REL))
		for key := range e.REL {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		relationships := make([]string, 0, len(keys))
		for _, key := range keys {
			relationships = append(
				relationships,
				key+" → "+e.REL[key],
			)
		}
		row.Relationships = strings.Join(relationships, "; ")

		out = append(out, row)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})

	return out, nil
}

func HTML(path string, st core.State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	t := template.Must(template.New("f").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width">
<title>EmergION Projection</title>
<style>
body{font:15px system-ui;background:#090b12;color:#f3f4f6;margin:0;padding:2rem}
h1{font-size:1.5rem}
h2{margin-top:2rem;font-size:1.15rem}
table{width:100%;border-collapse:collapse}
th,td{text-align:left;vertical-align:top;padding:.7rem;border-bottom:1px solid #2b3140}
code{color:#a78bfa}
.G{color:#f59e0b}
.F{color:#22c55e}
.X{color:#ef4444}
.small{color:#9ca3af;font-size:.9rem}
</style>
</head>
<body>

<h1>Living Projection</h1>
<p>Events: {{.Events}} · Tip: <code>{{.TipHash}}</code></p>
<p class="small">Projection is derived state only. COSL + governance remain authoritative.</p>

<h2>Lifecycle Projection</h2>
<table>
<thead>
<tr>
<th>EmergION</th>
<th>State</th>
<th>Risk</th>
<th>Meaning</th>
</tr>
</thead>
<tbody>
{{range .Rows}}
<tr>
<td><code>{{.ID}}</code></td>
<td class="{{.State}}">{{.State}}</td>
<td>{{.Risk}}</td>
<td>{{.Summary}}</td>
</tr>
{{end}}
</tbody>
</table>

<h2>SPATIAL CONVERGENCE ZONE</h2>
<p class="small">Accepted governed structures only.</p>

<table>
<thead>
<tr>
<th>EmergION</th>
<th>Meaning</th>
<th>Archonym</th>
<th>Kin</th>
<th>Topology</th>
<th>Facets</th>
<th>Capabilities</th>
<th>Relationships</th>
<th>Build Nodes</th>
<th>Build Edges</th>
</tr>
</thead>
<tbody>
{{range .Convergence}}
<tr>
<td><code>{{.ID}}</code></td>
<td>{{.Summary}}</td>
<td>{{.Archonym}}</td>
<td>{{.Kin}}</td>
<td>{{.Topology}}</td>
<td>{{.Facets}}</td>
<td>{{.Capabilities}}</td>
<td>{{.Relationships}}</td>
<td>{{.BuildNodes}}</td>
<td>{{.BuildEdges}}</td>
</tr>
{{end}}
</tbody>
</table>

</body>
</html>`))

	convergence, err := convergenceRows(st)
	if err != nil {
		return err
	}

	return t.Execute(f, map[string]any{
		"Events":      st.Events,
		"TipHash":     st.TipHash,
		"Rows":        rows(st),
		"Convergence": convergence,
	})
}

func EnsureOutput(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("output path required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return root, nil
}
