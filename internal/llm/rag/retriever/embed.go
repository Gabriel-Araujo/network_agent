package retriever

import (
	"context"
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/pkg/db"
	"github.com/openai/openai-go/v3"
)

// embedTexts gera os embeddings das queries em uma única chamada, usando o
// modelo de embedding configurado. O MESMO modelo do ingester deve ser
// usado aqui, ou a similaridade vetorial perde o sentido.
func embedTexts(ctx context.Context, cfg rag.Config, texts []string) ([][]float64, error) {
	client := cfg.Embedder
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: cfg.EmbeddingModel,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	})
	if err != nil {
		return nil, err
	}

	vectors := make([][]float64, len(resp.Data))
	for i, emb := range resp.Data {
		vectors[i] = emb.Embedding
	}
	if len(vectors) != len(texts) {
		return nil, fmt.Errorf("esperava %d embeddings, recebi %d", len(texts), len(vectors))
	}

	return vectors, nil
}

// ProbeEmbeddingDim embeds uma string de probe no 1º uso e confere a
// dimensão (4096) contra a coluna VECTOR(4096). Mismatch -> erro claro.
func ProbeEmbeddingDim(ctx context.Context, cfg rag.Config, store *db.DB) error {
	if store.CheckedDim {
		return nil
	}
	vecs, err := embedTexts(ctx, cfg, []string{"probe"})
	if err != nil {
		return fmt.Errorf("probe de dimensão: %w", err)
	}
	want := 4096
	if got := len(vecs[0]); got != want {
		return fmt.Errorf("dimensão do embedding é %d, esperava %d — alinhe modelo ou ALTER TABLE", got, want)
	}
	store.CheckedDim = true
	return nil
}
