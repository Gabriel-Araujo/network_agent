package rag

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
)

// Section representa uma seção do documento RST, com o caminho de
// breadcrumbs (Path) até a raiz.
type Section struct {
	Level     int
	Title     string
	Path      []string
	StartLine int // Primeira linha de conteúdo PRÓPRIO da seção
	EndLine   int // Onde começa o próximo header (exclusive)
}

// Chunk é a unidade final de indexação.
type Chunk struct {
	ChunkID       string  `json:"chunk_id"`
	ChunkType     string  `json:"chunk_type"` // "command_reference" | "concept"
	Daemon        string  `json:"daemon"`
	Protocol      string  `json:"protocol"`
	SectionPath   string  `json:"section_path"`
	Command       *string `json:"command"`
	Content       string  `json:"content"`
	ParentContent string  `json:"parent_content"`
	SourceURL     string  `json:"source_url"`
	TokenCount    int     `json:"token_count"`
	Order         int     `json:"order"`
}

// HeaderLine é uma ocorrência de header detectada antes da montagem da
// árvore de seções: (linha_do_titulo, nivel, titulo).
type HeaderLine struct {
	TitleLine int
	Level     int
	Title     string
}

// ClicmdEntry é um comando `.. Clicmd::` com seu corpo (descrição/exemplo).
type ClicmdEntry struct {
	Command string
	Body    []string
}

type StackEntry struct {
	Level int
	Title string
}

// -- Retriever Types -- //

// ErrNoQueries indica que o briefing não contém o campo "ragQueries"
// (ou veio vazio), obrigando o fallback por LLM.
var ErrNoQueries = errors.New("briefing sem queries de RAG (campo 'ragQueries')")

// QuerySuggestion é uma query reescrita para busca, com filtros
// opcionais de metadados para a tabela frr_docs. ChunkType vazio
// significa "sem filtro".
type QuerySuggestion struct {
	Query     string
	Protocol  string
	Daemon    string
	ChunkType string
}

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

func (c Config) WithDefaults() Config {
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

// QueryGenerator gera sugestões de query a partir do texto do briefing,
// usado como fallback quando o briefing não traz a tabela determinística.
type QueryGenerator interface {
	GenerateQueries(ctx context.Context, briefing []byte) ([]QuerySuggestion, error)
}

// Result é a saída estruturada pedida pelo usuário:
// [{"query": string, "response": string}], onde response é o contexto
// recuperado (parent_content + origem), não uma resposta gerada.
type Result struct {
	Query    string `json:"query"`
	Response string `json:"response"`
}

// ScoredChunk é um resultado de uma única fonte (vetorial ou FTS),
// identificado pelo chunk_id, antes da fusão RRF.
type ScoredChunk struct {
	ID    string
	Score float64
}
