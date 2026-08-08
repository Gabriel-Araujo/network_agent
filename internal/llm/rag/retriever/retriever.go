package retriever

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/querygen"
	"github.com/Gabriel-Araujo/network_agent/pkg/db"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

// VectorLiteral serializa um vetor para a literal textual aceita pelo
// pgvector (ex.: "[1,50000000,-2,25000000]"), castada como::vector na
// query. Não exigimos registro de tipo Go "p" pgvector.
func VectorLiteral(v []float64) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%.8f", x))
	}
	b.WriteByte(']')
	return b.String()
}

// FuseRRF funde duas fontes de ranking (map chunk_id → rank 0-based)
// usando RRF: score[id] += 1/(k + rank + 1) por fonte. Deduplica por id,
// ordena por score decrescente e limita ao limit.
func FuseRRF(vec, fts map[string]int, k, limit int) []rag.ScoredChunk {
	total := make(map[string]float64, len(vec)+len(fts))
	for id, rank := range vec {
		total[id] += 1.0 / float64(k+rank+1)
	}
	for id, rank := range fts {
		total[id] += 1.0 / float64(k+rank+1)
	}

	out := make([]rag.ScoredChunk, 0, len(total))
	for id, score := range total {
		out = append(out, rag.ScoredChunk{ID: id, Score: score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ID < out[j].ID
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Retrieve executa a busca híbrida para cada query sugerida e devolve um
// Result por query. Se a query não retornar nada, ainda assim o Result
// aparece com Response vazio (preservando o shape [{"query","response"}]).
func Retrieve(ctx context.Context, cfg rag.Config, queries []rag.QuerySuggestion) ([]rag.Result, error) {
	cfg = cfg.WithDefaults()

	if len(queries) == 0 {
		return nil, nil
	}

	store, err := db.OpenDB(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("abrindo pool pgx: %w", err)
	}

	// Probe de dimensão no 1º uso: garante consistência com a coluna.
	if err := ProbeEmbeddingDim(ctx, cfg, store); err != nil {
		return nil, err
	}

	texts := make([]string, len(queries))
	for i, q := range queries {
		texts[i] = q.Query
	}

	vectors, err := embedTexts(ctx, cfg, texts)
	if err != nil {
		return nil, fmt.Errorf("embedding de queries: %w", err)
	}

	results := make([]rag.Result, 0, len(queries))
	for i, q := range queries {
		vecs, err := searchVector(ctx, store, cfg, q, vectors[i])
		if err != nil {
			return nil, fmt.Errorf("busca vetorial [%s]: %w", q.Query, err)
		}
		fts, err := searchFTS(ctx, store, cfg, q)
		if err != nil {
			return nil, fmt.Errorf("busca full-text [%s]: %w", q.Query, err)
		}

		fused := FuseRRF(vecs, fts, rag.RrfK, cfg.Limit)
		chunks, err := fetchChunks(ctx, store, fused)
		if err != nil {
			return nil, fmt.Errorf("buscando metadados dos chunks [%s]: %w", q.Query, err)
		}

		results = append(results, rag.Result{
			Query:    q.Query,
			Response: buildResponse(chunks),
		})
	}

	return results, nil
}

// buildResponse monta o texto de contexto a partir dos chunks recuperados:
// parent_content + indicação de origem.
func buildResponse(chunks []rag.Chunk) string {
	if len(chunks) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range chunks {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		b.WriteString(c.ParentContent)
		if c.SourceURL != "" {
			b.WriteString("\n\nSource: " + c.SourceURL)
		}
		if c.SectionPath != "" {
			b.WriteString(" (" + c.SectionPath + ")")
		}
	}
	return b.String()
}

// Do é o ponto de entrada do retriever: lê o arquivo de briefing gerado
// pelo intent-analyser, gera/extrai as queries (parse determinístico da
// seção 6 com fallback LLM), executa a busca híbrida e grava o JSON
// [{"query","response"}] em.agent/tmp/retrieval/<arquivo-sem-ext>.json,
// devolvendo o caminho absoluto.
func Do(ctx context.Context, briefingPath string, cfg rag.Config) (string, error) {
	content, err := os.ReadFile(briefingPath)
	if err != nil {
		return "", fmt.Errorf("lendo briefing %s: %w", briefingPath, err)
	}

	gen := cfg.QueryGen
	queries, err := querygen.BuildQueries(ctx, content, gen)
	if err != nil {
		return "", fmt.Errorf("gerando queries do briefing: %w", err)
	}

	results, err := Retrieve(ctx, cfg, queries)
	if err != nil {
		return "", fmt.Errorf("buscando no pgvector: %w", err)
	}

	outPath, err := SaveResults(briefingPath, results)
	if err != nil {
		return "", err
	}
	return outPath, nil
}

// SaveResults grava o resultado JSON em retrievalDir, seguindo a mesma
// convenção de nome do briefing (<arquivo-sem-ext>.json) e retornando o
// caminho do arquivo criado.
func SaveResults(briefingPath string, results []rag.Result) (string, error) {
	base := filepath.Base(briefingPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	name := base + ".json"

	dir, err := util.SafePath(".", rag.RetrievalDir)
	if err != nil {
		return "", err
	}

	jsonPath := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serializando resultados: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return "", fmt.Errorf("gravando %s: %w", jsonPath, err)
	}
	return jsonPath, nil
}
