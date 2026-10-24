package rag

import (
	"context"

	"github.com/openai/openai-go/v3"
)

// Section represents an RST document section and its breadcrumb path to root.
type Section struct {
	Level     int
	Title     string
	Path      []string
	StartLine int // First line of the section's own content.
	EndLine   int // Start of the next header (exclusive).
}

// Chunk is the final indexing unit.
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

// HeaderLine is a header occurrence detected before building the section tree:
// (title line, level, title).
type HeaderLine struct {
	TitleLine int
	Level     int
	Title     string
}

// ClicmdEntry is a `.. Clicmd::` command with its body (description/example).
type ClicmdEntry struct {
	Command string
	Body    []string
}

type StackEntry struct {
	Level int
	Title string
}

// -- Retriever Types -- //

// Filters is the single metadata set for an input, applied to every query in a
// search. "" means no filter and maps to SQL NULL (nullableStr). The
// classifier owns membership validation; the retriever does not repeat it.
type Filters struct {
	Protocol  string
	Daemon    string
	ChunkType string
}

// QuerySuggestion is a search query rewritten from the input. Filters live in
// Filters and apply to the entire search.
type QuerySuggestion struct {
	Query string
}

// Config contains the retriever's PostgreSQL connection, embedding client, and
// query generator.
type Config struct {
	DSN            string
	EmbeddingModel string // Empty value uses DefaultEmbeddingModel.
	Embedder       openai.Client
	Limit          int // Final results per query (default 5).
	FetchFactor    int // Top-K per source = limit * FetchFactor (default 4).
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

// QueryGenerator rewrites user input into search queries, using the selected
// filters to guide the rewrite. On failure, the caller falls back to one raw
// input query with an explicit warning.
type QueryGenerator interface {
	GenerateQueries(ctx context.Context, input string, filters Filters) ([]QuerySuggestion, error)
}

// Result is the retriever's structured output. Protocol, daemon, and chunk_type
// identify the filters used to retrieve context.
type Result struct {
	Query     string `json:"query"`
	Protocol  string `json:"protocol"`
	Daemon    string `json:"daemon"`
	ChunkType string `json:"chunk_type"`
	Response  string `json:"response"`
}

// ScoredChunk is a result from one source (vector or FTS), identified by
// chunk_id before RRF fusion.
type ScoredChunk struct {
	ID    string
	Score float64
}
