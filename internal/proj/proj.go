package proj

import (
	"emergion-sovereign-runtime/internal/core"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
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

type row struct{ ID, State, Summary, Risk string }

func rows(st core.State) []row {
	var out []row
	add := func(m map[string]core.EmergION) {
		for _, e := range m {
			out = append(out, row{e.IDN, e.STA, e.MEM.Summary, e.VAL.Risk})
		}
	}
	add(st.AtGOV)
	add(st.Approved)
	add(st.Accepted)
	add(st.Held)
	add(st.Rejected)
	add(st.Returned)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
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
	t := template.Must(template.New("f").Parse(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>EmergION FIELD</title><style>body{font:15px system-ui;background:#090b12;color:#f3f4f6;margin:0;padding:2rem}h1{font-size:1.5rem}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:.7rem;border-bottom:1px solid #2b3140}code{color:#a78bfa}.G{color:#f59e0b}.F{color:#22c55e}.X{color:#ef4444}</style></head><body><h1>Living FIELD projection</h1><p>Events: {{.Events}} · Tip: <code>{{.TipHash}}</code></p><table><thead><tr><th>EmergION</th><th>State</th><th>Risk</th><th>Meaning</th></tr></thead><tbody>{{range .Rows}}<tr><td><code>{{.ID}}</code></td><td class="{{.State}}">{{.State}}</td><td>{{.Risk}}</td><td>{{.Summary}}</td></tr>{{end}}</tbody></table></body></html>`))
	return t.Execute(f, map[string]any{"Events": st.Events, "TipHash": st.TipHash, "Rows": rows(st)})
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
