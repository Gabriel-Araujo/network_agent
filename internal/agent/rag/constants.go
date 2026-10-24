package rag

const (
	ChunkTypeCommandReference = "command_reference"
	ChunkTypeConcept          = "concept"
)

// RetrievalDir é o diretório onde as saídas de busca são salvas, relativo
// ao diretório de trabalho atual. (.agent não está no .gitignore — os
// espelhos em .agent/tmp são artefatos de execução do agente.)
const RetrievalDir = ".agent/tmp/retrieval"

// EmbeddingDimension é a dimensão dos vetores persistidos em frr_docs.
const EmbeddingDimension = 2560

// DefaultEmbeddingModel é o modelo usado no ingester e no retriever.
// Manter alinhado com VECTOR(2560) da tabela frr_docs.
const DefaultEmbeddingModel = "text-embedding-qwen3-embedding-4b"

// RrfK é a constante padrão de Reciprocal Rank Fusion.
const RrfK = 60
