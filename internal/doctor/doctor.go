package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"emergion-sovereign-runtime/internal/reason"
	"emergion-sovereign-runtime/internal/store"
)

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}
type Report struct {
	Ready  bool      `json:"ready"`
	Checks []Check   `json:"checks"`
	At     time.Time `json:"at"`
}

func Run(ctx context.Context, stateRoot string, gemma reason.GemmaCLI) Report {
	r := Report{Ready: true, At: time.Now().UTC()}
	add := func(name string, ok bool, detail string) {
		r.Checks = append(r.Checks, Check{name, ok, detail})
		if !ok {
			r.Ready = false
		}
	}
	if s, err := store.Open(stateRoot); err != nil {
		add("state", false, err.Error())
	} else {
		probe := filepath.Join(s.Root, ".write-probe")
		if err := os.WriteFile(probe, []byte("ok"), 0600); err != nil {
			add("state", false, err.Error())
		} else {
			os.Remove(probe)
			add("state", true, s.Root)
		}
		if _, err := s.Events(); err != nil {
			add("ledger", false, err.Error())
		} else {
			add("ledger", true, "COSL chain valid")
		}
		if n, err := s.VerifyEvidence(); err != nil {
			add("evidence", false, err.Error())
		} else {
			add("evidence", true, itoa(n)+" object(s) verified")
		}
	}
	if err := gemma.Validate(); err != nil {
		add("gemma", false, err.Error())
	} else {
		add("gemma", true, fmt.Sprintf("%s | bin=%s | model=%s", gemma.Version(ctx), gemma.Binary, gemma.Model))
	}
	return r
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 20)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
