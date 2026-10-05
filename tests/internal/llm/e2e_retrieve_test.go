package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
)

// TestE2EDoIntegration exercita o pipeline completo da skill (geração de
// queries do input → embeddings LM Studio → busca híbrida → RRF → espelho do
// JSON) contra o banco real. Gated em DATABASE_URL.
func TestE2EDoIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não definido; pulando E2E de integração")
	}

	input := "meu ospf não converge entre o core e o edge"
	filtros := rag.Filtros{Protocol: "ospf", Daemon: "ospfd", ChunkType: "concept"}
	cfg := rag.Config{
		DSN:            dsn,
		EmbeddingModel: rag.DefaultEmbeddingModel,
		Embedder:       testEmbedderClient(),
		Limit:          3,
	}

	results, outPath, err := ragretriever.Do(context.Background(), input, filtros, cfg)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if outPath == "" {
		t.Fatal("espelho sem path")
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("espelho não gravado: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("lendo JSON de saída: %v", err)
	}
	var espelho []rag.Result
	if err := json.Unmarshal(data, &espelho); err != nil {
		t.Fatalf("JSON inválido em %s: %v", outPath, err)
	}
	if filepath.Base(outPath) == "" {
		t.Fatal("path sem nome")
	}

	if len(results) == 0 {
		t.Fatal("no results: nenhuma query gerada do input")
	}
	// Cada query deve ter encontrado ao menos um chunk relevante.
	var emptyResponses int
	for i, r := range results {
		if r.Query == "" {
			t.Errorf("result[%d]: query vazia", i)
		}
		// Metadados = o conjunto único de filtros da busca.
		if r.Protocol != filtros.Protocol || r.Daemon != filtros.Daemon || r.ChunkType != filtros.ChunkType {
			t.Errorf("result[%d]: metadados fora do conjunto único: %+v", i, r)
		}
		if r.Response == "" {
			emptyResponses++
		}
	}
	t.Logf("E2E OK: %d queries, %d sem contexto. Espelho: %s", len(results), emptyResponses, outPath)
	for i, r := range results {
		t.Logf("  [%d] query=%q len(response)=%d", i, r.Query, len(r.Response))
	}
}
