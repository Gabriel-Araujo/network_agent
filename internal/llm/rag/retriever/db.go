package retriever

import (
	"context"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/pkg/db"
)

// searchVector devolve map[chunk_id]rank para a busca vetorial de uma
// query, com top-K = limit*FetchFactor.
func searchVector(ctx context.Context, s *db.DB, cfg rag.Config, q rag.QuerySuggestion, vec []float64) (map[string]int, error) {
	limit := cfg.Limit * cfg.FetchFactor
	rows, err := s.Pool.Query(ctx, vectorSearchSQL,
		VectorLiteral(vec),
		nullableStr(q.Daemon),
		nullableStr(q.Protocol),
		nullableStr(q.ChunkType),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int)
	rank := 0
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return nil, err
		}
		out[id] = rank
		rank++
	}
	return out, rows.Err()
}

// searchFTS devolve map[chunk_id]rank para a busca full-text de uma query.
func searchFTS(ctx context.Context, s *db.DB, cfg rag.Config, q rag.QuerySuggestion) (map[string]int, error) {
	limit := cfg.Limit * cfg.FetchFactor
	rows, err := s.Pool.Query(ctx, ftsSearchSQL,
		q.Query,
		nullableStr(q.Daemon),
		nullableStr(q.Protocol),
		nullableStr(q.ChunkType),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int)
	rank := 0
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return nil, err
		}
		out[id] = rank
		rank++
	}
	return out, rows.Err()
}

// fetchChunks busca os metadados dos chunks pelos ids (na ordem do RRF).
func fetchChunks(ctx context.Context, s *db.DB, fused []rag.ScoredChunk) ([]rag.Chunk, error) {
	if len(fused) == 0 {
		return nil, nil
	}
	ids := make([]string, len(fused))
	for i := range fused {
		ids[i] = fused[i].ID
	}

	rows, err := s.Pool.Query(ctx, chunkLookupSQL, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[string]rag.Chunk, len(ids))
	for rows.Next() {
		var c rag.Chunk
		if err := rows.Scan(
			&c.ChunkID, &c.ChunkType, &c.Daemon, &c.Protocol,
			&c.SectionPath, &c.Command, &c.Content,
			&c.ParentContent, &c.SourceURL, &c.TokenCount,
		); err != nil {
			return nil, err
		}
		byID[c.ChunkID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Preserva a ordem do RRF.
	out := make([]rag.Chunk, 0, len(fused))
	for _, f := range fused {
		if c, ok := byID[f.ID]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// nullableStr converte string vazia em *string nil para os filtros SQL.
func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
