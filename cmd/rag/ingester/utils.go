package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/pkg/util/config"
	"github.com/jackc/pgx/v5"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func Embed(texts []string, client openai.Client) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	resp, err := client.Embeddings.New(context.Background(), openai.EmbeddingNewParams{
		Model: "text-embedding-qwen3-embedding-8b",
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
		return nil, fmt.Errorf(
			"esperava %d embeddings, recebi %d",
			len(texts),
			len(vectors),
		)
	}

	return vectors, nil
}

func LoadChunks(path string) ([]rag.Chunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var chunks []rag.Chunk

	if err := json.Unmarshal(data, &chunks); err != nil {
		return nil, err
	}

	return chunks, nil
}

func Upsert(ctx context.Context, conn *pgx.Conn, chunks []rag.Chunk, batchSize int) error {

	const query = `
INSERT INTO frr_docs (
    chunk_id,
    chunk_type,
    daemon,
    protocol,
    section_path,
    command,
    content,
    parent_content,
    source_url,
    token_count,
    embedding
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
)
ON CONFLICT (chunk_id)
DO UPDATE SET
    content = EXCLUDED.content,
    parent_content = EXCLUDED.parent_content,
    embedding = EXCLUDED.embedding,
    token_count = EXCLUDED.token_count
`

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	client := getOpenaiClient()

	total := len(chunks)

	for i := 0; i < total; i += batchSize {

		end := i + batchSize
		if end > total {
			end = total
		}

		batch := chunks[i:end]

		texts := make([]string, len(batch))
		for j := range batch {
			texts[j] = batch[j].Content
		}

		vectors, err := Embed(texts, client)
		if err != nil {
			return err
		}

		var pgBatch pgx.Batch

		for j, c := range batch {
			pgBatch.Queue(
				query,
				c.ChunkID,
				c.ChunkType,
				c.Daemon,
				c.Protocol,
				c.SectionPath,
				c.Command,
				c.Content,
				c.ParentContent,
				c.SourceURL,
				c.TokenCount,
				toPgVector(vectors[j]),
			)
		}

		results := tx.SendBatch(ctx, &pgBatch)

		for range batch {
			_, err := results.Exec()
			if err != nil {
				results.Close()
				return err
			}
		}

		results.Close()

		fmt.Printf("  %d/%d chunks processados\n", end, total)
	}

	return tx.Commit(ctx)
}

func getOpenaiClient() openai.Client {
	_config, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}
	return openai.NewClient(
		option.WithBaseURL(_config.Url),
		option.WithAPIKey(_config.ApiKey),
	)

}
func toPgVector(v []float64) string {
	var b strings.Builder

	b.WriteByte('[')

	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(x, 'f', -1, 32))
	}

	b.WriteByte(']')

	return b.String()
}
