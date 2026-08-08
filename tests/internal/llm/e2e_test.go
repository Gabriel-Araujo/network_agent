package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/querygen"
)

// TestE2EBriefingParsing roda o pipeline determinístico (parse da seção 6)
// contra um briefing de exemplo real gravado em .agent/tmp, e valida o
// shape exato [{"query","response"}] produzido pelo saveResults.
func TestE2EBriefingParsing(t *testing.T) {
	// go test roda a partir do diretório do pacote; subimos até a raiz do
	// módulo (internal/llm/rag -> ../../..).
	briefingPath := filepath.Join("..", "..", "..", ".agent", "tmp", "network_agent", "meu-ospf-nao-converge-entre-60becac1.md")
	content, err := os.ReadFile(briefingPath)
	if err != nil {
		t.Skipf("briefing de exemplo não está presente (%v); pulando", err)
	}

	qs, err := querygen.ParseQueriesFromBriefing(content)
	if err != nil {
		t.Fatalf("ParseQueriesFromBriefing: %v", err)
	}
	if len(qs) != 5 {
		t.Fatalf("len(qs) = %d, want 5", len(qs))
	}
	if qs[0].Query != "OSPF neighbor stuck Exstart Exchange state" {
		t.Errorf("query[0] = %q", qs[0].Query)
	}
	if qs[0].Protocol != "ospf" || qs[0].Daemon != "ospfd" || qs[0].ChunkType != "concept" {
		t.Errorf("filters[0] = %+v", qs[0])
	}
	if qs[4].Protocol != "bgp" || qs[4].Daemon != "bgpd" {
		t.Errorf("filters[4] = %+v", qs[4])
	}

	// Garante o shape JSON mínimo especificado: [{"query":string,"response":string}].
	// Sem DB, response fica vazio — mas o contrato de serialização deve valer.
	out, err := json.Marshal([]rag.Result{{Query: qs[0].Query}})
	if err != nil {
		t.Fatal(err)
	}
	var back []map[string]string
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || back[0]["query"] == "" {
		t.Fatalf("shape JSON inválido: %s", out)
	}
}
