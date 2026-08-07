package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/openai/openai-go/v3"
)

// DefaultEmbeddingModel é o modelo usado no ingester e no retriever.
// Manter alinhado com VECTOR(4096) da tabela frr_docs.
const DefaultEmbeddingModel = "text-embedding-qwen3-embedding-8b"

// Config centraliza o que o retriever precisa: conexão com o Postgres,
// cliente de embeddings e o fallback de geração de queries.
type Config struct {
	DSN            string
	EmbeddingModel string // vazio -> DefaultEmbeddingModel
	Embedder       openai.Client
	Limit          int // resultados finais por query (default 5)
	FetchFactor    int // top-K por fonte = limit * FetchFactor (default 4)
	QueryGen       QueryGenerator
}

func (c Config) withDefaults() Config {
	if c.EmbeddingModel == "" {
		c.EmbeddingModel = DefaultEmbeddingModel
	}
	if c.Limit <= 0 {
		c.Limit = 5
	}
	if c.FetchFactor <= 0 {
		c.FetchFactor = 4
	}
	return c
}

// Result é a saída estruturada pedida pelo usuário:
// [{"query": string, "response": string}], onde response é o contexto
// recuperado (parent_content + origem), não uma resposta gerada.
type Result struct {
	Query    string `json:"query"`
	Response string `json:"response"`
}

// scoredChunk é um resultado de uma única fonte (vetorial ou FTS),
// identificado pelo chunk_id, antes da fusão RRF.
type scoredChunk struct {
	ID    string
	Score float64
}

// VectorLiteral serializa um vetor para a literal textual aceita pelo
// pgvector (ex.: "[1.50000000,-2.25000000]"), castada como ::vector na
// query. Não exigimos registro de tipo Go p/ pgvector.
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

// rrfK é a constante padrão de Reciprocal Rank Fusion.
const rrfK = 60

// fuseRRF funde duas fontes de ranking (map chunk_id -> rank 0-based)
// usando RRF: score[id] += 1/(k + rank + 1) por fonte. Deduplica por id,
// ordena por score decrescente e limita ao limit.
func fuseRRF(vec, fts map[string]int, k, limit int) []scoredChunk {
	total := make(map[string]float64, len(vec)+len(fts))
	for id, rank := range vec {
		total[id] += 1.0 / float64(k+rank+1)
	}
	for id, rank := range fts {
		total[id] += 1.0 / float64(k+rank+1)
	}

	out := make([]scoredChunk, 0, len(total))
	for id, score := range total {
		out = append(out, scoredChunk{ID: id, Score: score})
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
func Retrieve(ctx context.Context, cfg Config, queries []QuerySuggestion) ([]Result, error) {
	cfg = cfg.withDefaults()

	if len(queries) == 0 {
		return nil, nil
	}

	store, err := openStore(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("abrindo pool pgx: %w", err)
	}

	// Probe de dimensão no 1º uso: garante consistência com a coluna.
	if err := probeEmbeddingDim(ctx, cfg, store); err != nil {
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

	results := make([]Result, 0, len(queries))
	for i, q := range queries {
		vecs, err := searchVector(ctx, store, cfg, q, vectors[i])
		if err != nil {
			return nil, fmt.Errorf("busca vetorial [%s]: %w", q.Query, err)
		}
		fts, err := searchFTS(ctx, store, cfg, q)
		if err != nil {
			return nil, fmt.Errorf("busca full-text [%s]: %w", q.Query, err)
		}

		fused := fuseRRF(vecs, fts, rrfK, cfg.Limit)
		chunks, err := fetchChunks(ctx, store, fused)
		if err != nil {
			return nil, fmt.Errorf("buscando metadados dos chunks [%s]: %w", q.Query, err)
		}

		results = append(results, Result{
			Query:    q.Query,
			Response: buildResponse(chunks),
		})
	}

	return results, nil
}

// buildResponse monta o texto de contexto a partir dos chunks recuperados:
// parent_content + indicação de origem.
func buildResponse(chunks []Chunk) string {
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
