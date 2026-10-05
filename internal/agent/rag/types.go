package rag

import (
	"context"

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

// Filtros é o conjunto único de metadados por input, aplicado a todas as
// queries da busca (um conjunto por busca inteira, não por query). ""
// significa "sem filtro" e casa com o NULL do SQL (nullableStr). Quem chama
// é o Classificador (Modo RAG); a validação de pertencimento é dele — o
// retriever não revalida.
type Filtros struct {
	Protocol  string
	Daemon    string
	ChunkType string
}

// QuerySuggestion é uma query de busca reescrita a partir do input. Só o
// texto: os filtros vivem em Filtros, um conjunto para a busca inteira.
type QuerySuggestion struct {
	Query string
}

// Config centraliza o que o retriever precisa: conexão com o Postgres,
// cliente de embeddings e o gerador de queries.
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

// QueryGenerator reescreve o input do usuário em queries de busca
// (caminho primário da busca), recebendo os filtros escolhidos para orientar
// a reescrita. Falhou a geração, o chamador degrada para uma query única do
// input cru — com aviso explícito.
type QueryGenerator interface {
	GenerateQueries(ctx context.Context, input string, filtros Filtros) ([]QuerySuggestion, error)
}

// Result é a saída estruturada do retriever. Os metadados protocol, daemon e
// chunk_type identificam os filtros usados para recuperar o contexto.
type Result struct {
	Query     string `json:"query"`
	Protocol  string `json:"protocol"`
	Daemon    string `json:"daemon"`
	ChunkType string `json:"chunk_type"`
	Response  string `json:"response"`
}

// ScoredChunk é um resultado de uma única fonte (vetorial ou FTS),
// identificado pelo chunk_id, antes da fusão RRF.
type ScoredChunk struct {
	ID    string
	Score float64
}
