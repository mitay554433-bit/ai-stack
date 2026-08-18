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

type prm struct {
	SourceEmergIONID string
	SourceHash       string
	KinRoot          string
	Archonym         string
	Capabilities     []string
	Facts            []string
	Relationships    map[string]string
	Facets           []core.Facet
	BuildNodes       []core.BuildNode
	BuildEdges       []core.BuildEdge
	Monetization     *core.Monetization
	Delta            []string
}

func crystallizePRMs(st core.State) ([]prm, error) {
	out := make([]prm, 0, len(st.Accepted))

	for _, em := range st.Accepted {
		root, err := livefield.AcceptedKinRoot(st.Accepted, st.Returned, em.IDN)
		if err != nil {
			return nil, err
		}

		item := prm{
			SourceEmergIONID: em.IDN,
			SourceHash:       em.MEM.SourceHash,
			KinRoot:          root,
			Capabilities:     append([]string(nil), em.CAP...),
			Facts:            append([]string(nil), em.VAL.Facts...),
			Relationships:    map[string]string{},
			Delta:            append([]string(nil), em.EVO.Delta...),
		}

		for key, value := range em.REL {
			item.Relationships[key] = value
		}

		if em.EVO.Metadata != nil {
			item.Archonym = em.EVO.Metadata.Archonym
			item.Facets = append([]core.Facet(nil), em.EVO.Metadata.Facets...)
			item.BuildNodes = append([]core.BuildNode(nil), em.EVO.Metadata.BuildNodes...)
			item.BuildEdges = append([]core.BuildEdge(nil), em.EVO.Metadata.BuildEdges...)
			if em.EVO.Metadata.Monetization != nil {
				value := *em.EVO.Metadata.Monetization
				item.Monetization = &value
			}
		}

		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].SourceEmergIONID < out[j].SourceEmergIONID
	})

	return out, nil
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
		root, err := livefield.AcceptedKinRoot(st.Accepted, st.Returned, e.IDN)
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

type saabLink struct {
	FromPRM string
	ToPRM   string
	Kind    string
}

type commercialProjection struct {
	SourcePRMID string
	Model       string
	Customer    string
	Value       string
	RevenuePath string
}

type saab struct {
	ID               string
	MemberPRMIDs     []string
	KinRoots         []string
	Capabilities     []string
	Commercial       []commercialProjection
	CompositionLinks []saabLink
	BuildNodes       []core.BuildNode
	BuildEdges       []core.BuildEdge
}

type cpsl struct {
	SAABID  string
	Members []string
	Program string
}

func deriveSAABs(st core.State) ([]saab, error) {
	prms, err := crystallizePRMs(st)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]prm, len(prms))
	adjacency := make(map[string]map[string]bool, len(prms))

	for _, item := range prms {
		byID[item.SourceEmergIONID] = item
		adjacency[item.SourceEmergIONID] = map[string]bool{}
	}

	for _, item := range prms {
		target := strings.TrimSpace(
			item.Relationships["COMPOSITION_KIN"],
		)
		if target == "" {
			continue
		}

		if target == item.SourceEmergIONID {
			return nil, fmt.Errorf(
				"SAAB rejected self COMPOSITION_KIN %s",
				target,
			)
		}

		if _, ok := byID[target]; !ok {
			return nil, fmt.Errorf(
				"SAAB composition target not accepted PRM: %s",
				target,
			)
		}

		// Membership connectivity is undirected. The explicit governed
		// composition relation itself remains directed in CompositionLinks.
		adjacency[item.SourceEmergIONID][target] = true
		adjacency[target][item.SourceEmergIONID] = true
	}

	ids := make([]string, 0, len(prms))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	visited := map[string]bool{}
	var out []saab

	for _, start := range ids {
		if visited[start] || len(adjacency[start]) == 0 {
			continue
		}

		queue := []string{start}
		visited[start] = true

		var members []string

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			members = append(members, current)

			neighbors := make(
				[]string,
				0,
				len(adjacency[current]),
			)
			for neighbor := range adjacency[current] {
				neighbors = append(neighbors, neighbor)
			}
			sort.Strings(neighbors)

			for _, neighbor := range neighbors {
				if visited[neighbor] {
					continue
				}
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}

		sort.Strings(members)

		if len(members) < 2 {
			continue
		}

		assembly := saab{
			ID:           "SAAB:" + strings.Join(members, "+"),
			MemberPRMIDs: append([]string(nil), members...),
		}

		kinSet := map[string]bool{}
		capSet := map[string]bool{}
		linkSet := map[string]bool{}

		for _, memberID := range members {
			item := byID[memberID]

			if item.KinRoot != "" {
				kinSet[item.KinRoot] = true
			}

			if item.Monetization != nil {
				assembly.Commercial = append(
					assembly.Commercial,
					commercialProjection{
						SourcePRMID: memberID,
						Model:       item.Monetization.Model,
						Customer:    item.Monetization.Customer,
						Value:       item.Monetization.Value,
						RevenuePath: item.Monetization.RevenuePath,
					},
				)
			}

			for _, capability := range item.Capabilities {
				if capability != "" {
					capSet[capability] = true
				}
			}

			target := strings.TrimSpace(
				item.Relationships["COMPOSITION_KIN"],
			)
			if target != "" {
				if _, inside := byID[target]; inside {
					key := memberID + "\x00" + target
					if !linkSet[key] {
						linkSet[key] = true
						assembly.CompositionLinks = append(
							assembly.CompositionLinks,
							saabLink{
								FromPRM: memberID,
								ToPRM:   target,
								Kind:    "COMPOSITION_KIN",
							},
						)
					}
				}
			}

			for _, node := range item.BuildNodes {
				namespaced := node
				namespaced.ID = memberID + "::" + node.ID

				assembly.BuildNodes = append(
					assembly.BuildNodes,
					namespaced,
				)
			}

			for _, edge := range item.BuildEdges {
				namespaced := edge
				namespaced.From = memberID + "::" + edge.From
				namespaced.To = memberID + "::" + edge.To

				assembly.BuildEdges = append(
					assembly.BuildEdges,
					namespaced,
				)
			}
		}

		for root := range kinSet {
			assembly.KinRoots = append(
				assembly.KinRoots,
				root,
			)
		}
		sort.Strings(assembly.KinRoots)

		for capability := range capSet {
			assembly.Capabilities = append(
				assembly.Capabilities,
				capability,
			)
		}
		sort.Strings(assembly.Capabilities)

		sort.Slice(
			assembly.CompositionLinks,
			func(i, j int) bool {
				if assembly.CompositionLinks[i].FromPRM !=
					assembly.CompositionLinks[j].FromPRM {
					return assembly.CompositionLinks[i].FromPRM <
						assembly.CompositionLinks[j].FromPRM
				}
				return assembly.CompositionLinks[i].ToPRM <
					assembly.CompositionLinks[j].ToPRM
			},
		)

		sort.Slice(
			assembly.BuildNodes,
			func(i, j int) bool {
				return assembly.BuildNodes[i].ID <
					assembly.BuildNodes[j].ID
			},
		)

		sort.Slice(
			assembly.BuildEdges,
			func(i, j int) bool {
				left := assembly.BuildEdges[i]
				right := assembly.BuildEdges[j]

				if left.From != right.From {
					return left.From < right.From
				}
				if left.To != right.To {
					return left.To < right.To
				}
				return left.Kind < right.Kind
			},
		)

		out = append(out, assembly)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})

	return out, nil
}

func compileCPSL(st core.State) ([]cpsl, error) {
	assemblies, err := deriveSAABs(st)
	if err != nil {
		return nil, err
	}

	out := make([]cpsl, 0, len(assemblies))

	for _, assembly := range assemblies {
		var b strings.Builder

		b.WriteString("CPSL/1\n")
		fmt.Fprintf(&b, "A|%q\n", assembly.ID)

		for _, member := range assembly.MemberPRMIDs {
			fmt.Fprintf(&b, "P|%q\n", member)
		}

		for _, root := range assembly.KinRoots {
			fmt.Fprintf(&b, "K|%q\n", root)
		}

		for _, capability := range assembly.Capabilities {
			fmt.Fprintf(&b, "C|%q\n", capability)
		}

		for _, link := range assembly.CompositionLinks {
			fmt.Fprintf(
				&b,
				"L|%q|%q|%q\n",
				link.FromPRM,
				link.ToPRM,
				link.Kind,
			)
		}

		for _, node := range assembly.BuildNodes {
			fmt.Fprintf(
				&b,
				"N|%q|%q|%q\n",
				node.ID,
				node.System,
				node.State,
			)
		}

		for _, edge := range assembly.BuildEdges {
			fmt.Fprintf(
				&b,
				"E|%q|%q|%q\n",
				edge.From,
				edge.To,
				edge.Kind,
			)
		}

		b.WriteString("Z\n")

		out = append(out, cpsl{
			SAABID:  assembly.ID,
			Members: append([]string(nil), assembly.MemberPRMIDs...),
			Program: b.String(),
		})
	}

	return out, nil
}

type saw struct {
	ID           string
	SAABID       string
	MemberPRMIDs []string
	Capabilities []string
	Commercial   []commercialProjection
	BuildNodes   []core.BuildNode
	BuildEdges   []core.BuildEdge
	CPSL         string
}

type libEntry struct {
	SAWID        string
	SAABID       string
	MemberPRMIDs []string
	Capabilities []string
	Commercial   []commercialProjection
}

func extractSAWs(st core.State) ([]saw, error) {
	assemblies, err := deriveSAABs(st)
	if err != nil {
		return nil, err
	}

	programs, err := compileCPSL(st)
	if err != nil {
		return nil, err
	}

	cpslBySAAB := make(map[string]cpsl, len(programs))
	for _, program := range programs {
		if _, exists := cpslBySAAB[program.SAABID]; exists {
			return nil, fmt.Errorf(
				"duplicate CPSL for SAAB %s",
				program.SAABID,
			)
		}
		cpslBySAAB[program.SAABID] = program
	}

	out := make([]saw, 0, len(assemblies))

	for _, assembly := range assemblies {
		program, ok := cpslBySAAB[assembly.ID]
		if !ok {
			return nil, fmt.Errorf(
				"SAW missing CPSL for SAAB %s",
				assembly.ID,
			)
		}

		item := saw{
			ID:           "SAW:" + assembly.ID,
			SAABID:       assembly.ID,
			MemberPRMIDs: append([]string(nil), assembly.MemberPRMIDs...),
			Capabilities: append([]string(nil), assembly.Capabilities...),
			Commercial:   append([]commercialProjection(nil), assembly.Commercial...),
			BuildNodes:   append([]core.BuildNode(nil), assembly.BuildNodes...),
			BuildEdges:   append([]core.BuildEdge(nil), assembly.BuildEdges...),
			CPSL:         program.Program,
		}

		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})

	return out, nil
}

func buildLIB(st core.State) ([]libEntry, error) {
	artifacts, err := extractSAWs(st)
	if err != nil {
		return nil, err
	}

	out := make([]libEntry, 0, len(artifacts))
	seen := map[string]bool{}

	for _, artifact := range artifacts {
		if artifact.ID == "" {
			return nil, fmt.Errorf("LIB rejected empty SAW identity")
		}

		if seen[artifact.ID] {
			return nil, fmt.Errorf(
				"LIB rejected duplicate SAW identity %s",
				artifact.ID,
			)
		}
		seen[artifact.ID] = true

		out = append(out, libEntry{
			SAWID:        artifact.ID,
			SAABID:       artifact.SAABID,
			MemberPRMIDs: append([]string(nil), artifact.MemberPRMIDs...),
			Capabilities: append([]string(nil), artifact.Capabilities...),
			Commercial:   append([]commercialProjection(nil), artifact.Commercial...),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].SAWID < out[j].SAWID
	})

	return out, nil
}

type SAWSource struct {
	ID      string
	Content []byte
}

func SAWSources(st core.State) ([]SAWSource, error) {
	artifacts, err := extractSAWs(st)
	if err != nil {
		return nil, err
	}

	out := make([]SAWSource, 0, len(artifacts))

	for _, artifact := range artifacts {
		var b strings.Builder

		b.WriteString("SAW/1\n")
		fmt.Fprintf(&b, "I|%q\n", artifact.ID)
		fmt.Fprintf(&b, "A|%q\n", artifact.SAABID)

		for _, member := range artifact.MemberPRMIDs {
			fmt.Fprintf(&b, "P|%q\n", member)
		}

		for _, capability := range artifact.Capabilities {
			fmt.Fprintf(&b, "C|%q\n", capability)
		}

		for _, commercial := range artifact.Commercial {
			fmt.Fprintf(
				&b,
				"M|%q|%q|%q|%q|%q\n",
				commercial.SourcePRMID,
				commercial.Model,
				commercial.Customer,
				commercial.Value,
				commercial.RevenuePath,
			)
		}

		fmt.Fprintf(&b, "X|%q\n", artifact.CPSL)
		b.WriteString("Z\n")

		out = append(out, SAWSource{
			ID:      artifact.ID,
			Content: []byte(b.String()),
		})
	}

	return out, nil
}
