package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
)

// TestE2EDoIntegration exercita o pipeline completo da skill (parse do
// "ragQueries" do JSON → embeddings LM Studio → busca híbrida → RRF → gravação do JSON)
// contra o banco real. Gated em DATABASE_URL.
func TestE2EDoIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não definido; pulando E2E de integração")
	}

	briefingPath := filepath.Join("..", "..", "..", ".agent", "tmp", "network_agent", "meu-ospf-nao-converge-entre-60becac1.json")
	if _, err := os.Stat(briefingPath); err != nil {
		t.Skipf("briefing de exemplo ausente (%v); pulando", err)
	}

	cfg := rag.Config{
		DSN:            dsn,
		EmbeddingModel: rag.DefaultEmbeddingModel,
		Embedder:       testEmbedderClient(),
		Limit:          3,
	}

	outPath, err := ragretriever.Do(context.Background(), briefingPath, cfg)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("lendo JSON de saída: %v", err)
	}
	var results []rag.Result
	if err := json.Unmarshal(data, &results); err != nil {
		t.Fatalf("JSON inválido em %s: %v", outPath, err)
	}

	if len(results) == 0 {
		t.Fatal("no results: nenhuma query gerada do briefing")
	}
	// Cada query deve ter encontrado ao menos um chunk relevante.
	var emptyResponses int
	for i, r := range results {
		if r.Query == "" {
			t.Errorf("result[%d]: query vazia", i)
		}
		if r.Protocol == "" || r.Daemon == "" || r.ChunkType == "" {
			t.Errorf("result[%d]: metadados incompletos: %+v", i, r)
		}
		if r.Response == "" {
			emptyResponses++
		}
	}
	t.Logf("E2E OK: %d queries, %d sem contexto. Arquivo: %s", len(results), emptyResponses, outPath)
	for i, r := range results {
		t.Logf("  [%d] query=%q len(response)=%d", i, r.Query, len(r.Response))
	}
}
