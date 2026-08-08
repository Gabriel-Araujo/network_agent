package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/retriever"
)

// TestE2EDoIntegration exercita o pipeline completo (parse da seção 6 →
// embeddings LM Studio → busca híbrida → RRF → gravação do JSON) contra
// o banco real. Gated em DATABASE_URL.
func TestE2EDoIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não definido; pulando E2E de integração")
	}

	briefingPath := filepath.Join("..", "..", "..", ".agent", "tmp", "network_agent", "meu-ospf-nao-converge-entre-60becac1.md")
	if _, err := os.Stat(briefingPath); err != nil {
		t.Skipf("briefing de exemplo ausente (%v); pulando", err)
	}

	cfg := rag.Config{
		DSN:            dsn,
		EmbeddingModel: rag.DefaultEmbeddingModel,
		Embedder:       testEmbedderClient(),
		Limit:          3,
	}

	outPath, err := retriever.Do(context.Background(), briefingPath, cfg)
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
		if r.Response == "" {
			emptyResponses++
		}
	}
	t.Logf("E2E OK: %d queries, %d sem contexto. Arquivo: %s", len(results), emptyResponses, outPath)
	for i, r := range results {
		t.Logf("  [%d] query=%q len(response)=%d", i, r.Query, len(r.Response))
	}
}
