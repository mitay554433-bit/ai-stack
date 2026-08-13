package core

import (
	"strings"
	"testing"
)

func TestEmergIONSymbolic(t *testing.T) {
	em := EmergION{
		IDN: "E-TEST",
		STA: StateAtGOV,
		MEM: Memory{
			SourceHash: "abc123",
			Provenance: "source/test",
		},
		VAL: Validation{
			Gaps:   []string{"BRIDGEGAP:test"},
			Recoil: true,
			WVC:    true,
		},
	}

	got := em.Symbolic()

	required := []string{
		"萌現/1\n",
		"識=E-TEST\n",
		"中心=abc123\n",
		"法=FORWARD_CREATES_BACKWARD_IMPRINT;BACKWARD_CREATES_FORWARD_IMPRINT\n",
		"状態=G\n",
		"源=source/test\n",
		"源根=abc123\n",
		"差=BRIDGEGAP:test\n",
		"退印=RETURN_TO:abc123\n",
		"進印=HUMAN_FINAL_REVIEW\n",
		"次=HUMAN_FINAL_REVIEW\n",
		"RECOIL=true\n",
		"WVC=true\n",
		"結果=PASS\n",
		"終/萌現\n",
	}

	for _, want := range required {
		if !strings.Contains(got, want) {
			t.Fatalf("symbolic EmergION missing %q\n%s", want, got)
		}
	}

	if got != em.Symbolic() {
		t.Fatal("symbolic EmergION is not deterministic")
	}
}

func TestEmergIONSymbolicUsesSupersedesForBackwardImprint(t *testing.T) {
	em := EmergION{
		IDN: "E-NEW",
		STA: StateAtGOV,
		MEM: Memory{
			SourceHash: "newhash",
		},
		VAL: Validation{
			Recoil: true,
			WVC:    true,
		},
		EVO: Evolution{
			Version:    1,
			Supersedes: "E-PRIOR",
			Delta:      []string{"changed behavior"},
		},
	}

	got := em.Symbolic()

	if !strings.Contains(got, "退印=SUPERSEDES:E-PRIOR\n") {
		t.Fatalf("symbolic EmergION did not use governed predecessor:\n%s", got)
	}

	if strings.Contains(got, "退印=RETURN_TO:newhash\n") {
		t.Fatalf("symbolic EmergION used source fallback despite predecessor:\n%s", got)
	}
}
