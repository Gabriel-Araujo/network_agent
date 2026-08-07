package rag

import (
	"context"
)

// searchVectorTopK é o multiplicador do limite usado para buscar nas
// fontes antes da fusão RRF (o próprio retrieve já aplica cfg.FetchFactor).

// vectorSearchSQL busca os chunks mais próximos via cosine distance
// (embedding <=> $1::vector), usando o índice diskann. Filtros opcionais
// são passados como *string (NULL -> sem filtro).
const vectorSearchSQL = `
SELECT chunk_id,
       1 - (embedding <=> $1::vector) AS score
FROM frr_docs
WHERE ($2::text IS NULL OR daemon = $2)
  AND ($3::text IS NULL OR protocol = $3)
  AND ($4::text IS NULL OR chunk_type = $4)
ORDER BY embedding <=> $1::vector
LIMIT $5;
`

// ftsSearchSQL busca via full-text (content_tsv @@ plainto_tsquery),
// usando o índice GIN; plainto_tsquery sanitiza o input do usuário.
const ftsSearchSQL = `
SELECT chunk_id,
       ts_rank_cd(content_tsv, plainto_tsquery('english', $1)) AS score
FROM frr_docs
WHERE content_tsv @@ plainto_tsquery('english', $1)
  AND ($2::text IS NULL OR daemon = $2)
  AND ($3::text IS NULL OR protocol = $3)
  AND ($4::text IS NULL OR chunk_type = $4)
ORDER BY score DESC
LIMIT $5;
`

// chunkLookupSQL busca os metadados dos chunks vencedores já deduplicados.
const chunkLookupSQL = `
SELECT chunk_id, chunk_type, daemon, protocol, section_path, command,
       content, parent_content, source_url, token_count
FROM frr_docs
WHERE chunk_id = ANY($1::text[]);
`

// searchVector devolve map[chunk_id]rank para a busca vetorial de uma
// query, com top-K = limit*FetchFactor.
func searchVector(ctx context.Context, s *store, cfg Config, q QuerySuggestion, vec []float64) (map[string]int, error) {
	limit := cfg.Limit * cfg.FetchFactor
	rows, err := s.pool.Query(ctx, vectorSearchSQL,
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
func searchFTS(ctx context.Context, s *store, cfg Config, q QuerySuggestion) (map[string]int, error) {
	limit := cfg.Limit * cfg.FetchFactor
	rows, err := s.pool.Query(ctx, ftsSearchSQL,
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
func fetchChunks(ctx context.Context, s *store, fused []scoredChunk) ([]Chunk, error) {
	if len(fused) == 0 {
		return nil, nil
	}
	ids := make([]string, len(fused))
	for i := range fused {
		ids[i] = fused[i].ID
	}

	rows, err := s.pool.Query(ctx, chunkLookupSQL, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[string]Chunk, len(ids))
	for rows.Next() {
		var c Chunk
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
	out := make([]Chunk, 0, len(fused))
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
