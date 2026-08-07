package intentanalyser

import (
	"context"
	"log"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
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
		if got := slugify(c.in); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHash8(t *testing.T) {
	h := hash8("x")
	if len(h) != 8 {
		t.Errorf("[ANALYSER TEST] hash8 len = %d, want 8", len(h))
	}
	if hash8("x") == hash8("y") {
		t.Errorf("[ANALYSER TEST] hash8 colidiu para entradas diferentes")
	}
}

func TestAgent(t *testing.T) {
	agent := llm.LoadTestAgent()

	path, err := Do(context.TODO(), "Gere um teste bgp", agent)
	if err != nil {
		panic(err)
	}
	log.Println("[ANALYSER TEST] Saved on: " + path)
}
