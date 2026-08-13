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
		"進印=REPEAT_ON_NEW_SOURCE\n",
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
