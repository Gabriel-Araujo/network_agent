package retriever

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/pkg/db"
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
// aparece com Response vazio, preservando seus metadados.
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
			Query:     q.Query,
			Protocol:  q.Protocol,
			Daemon:    q.Daemon,
			ChunkType: q.ChunkType,
			Response:  buildResponse(chunks),
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
