package skills

import (
	"context"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/intent-analyser"
)

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"meu ospf não converge entre o core", "meu-ospf-nao-converge-entre"},
		{"OSPF Não  Converge!!! 123", "ospf-nao-converge-123"},
		{"Gi0/1 em área 0", "gi0-1-em-area-0"},
		{"   ", ""},
		{"alô, mundo", "alo-mundo"},
	}
	for _, c := range cases {
		if got := intentanalyser.Slugify(c.in); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHash8(t *testing.T) {
	h := intentanalyser.Hash8("x")
	if len(h) != 8 {
		t.Errorf("[ANALYSER TEST] hash8 len = %d, want 8", len(h))
	}
	if intentanalyser.Hash8("x") == intentanalyser.Hash8("y") {
		t.Errorf("[ANALYSER TEST] hash8 colidiu para entradas diferentes")
	}
}

func TestAgent(t *testing.T) {
	agent := llm.LoadTestAgent()

	path, err := intentanalyser.Do(context.TODO(), "Gere um teste bgp", agent)
	if err != nil {
		panic(err)
	}
	t.Log("saved on: " + path)
}
