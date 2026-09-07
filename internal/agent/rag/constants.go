package rag

const (
	ChunkTypeCommandReference = "command_reference"
	ChunkTypeConcept          = "concept"

	queriesHeader = "Rewritten queries for RAG"
)

// RetrievalDir é o diretório onde as saídas de busca são salvas, relativo
// ao diretório de trabalho atual. (.agent já está no .gitignore.)
const RetrievalDir = ".agent/tmp/retrieval"

// DefaultEmbeddingModel é o modelo usado no ingester e no retriever.
// Manter alinhado com VECTOR(4096) da tabela frr_docs.
const DefaultEmbeddingModel = "text-embedding-qwen3-embedding-8b"

// RrfK é a constante padrão de Reciprocal Rank Fusion.
const RrfK = 60
