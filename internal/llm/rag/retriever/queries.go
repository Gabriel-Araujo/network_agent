package retriever

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
