package env

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/retriever"
	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	"github.com/Gabriel-Araujo/network_agent/pkg/db"
)

// LoadEnv lê as variáveis usadas pelo retriever. Para cada chave,
// prioriza a variável de ambiente real; se ausente, tenta o arquivo .env
// na raiz do repositório (mesma convenção de internal/llm/config.go).
func LoadEnv() (datasourceURL, embeddingModel string) {
	datasourceURL = firstNonEmpty(os.Getenv("DATABASE_URL"), envFileValue("DATABASE_URL"))
	embeddingModel = firstNonEmpty(os.Getenv("EMBEDDING_MODEL"), envFileValue("EMBEDDING_MODEL"))
	if embeddingModel == "" {
		embeddingModel = rag.DefaultEmbeddingModel
	}
	return datasourceURL, embeddingModel
}

// envFileValue lê do .env na raiz do repo a chave dada (sem sobrecarregar
// variáveis de ambiente). Retorna string vazia se não houver.
func envFileValue(key string) string {
	envPath, err := paths.RepoFile(".env")
	if err != nil {
		return ""
	}
	return ReadEnvKey(envPath, key)
}

// ReadEnvKey lê uma chave KEY=value de um arquivo .env, ignorando linhas
// vazias e comentários (#). Retorna string vazia se não encontrada.
func ReadEnvKey(envPath, key string) string {
	f, err := os.Open(envPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

// firstNonEmpty retorna a primeira string não-vazia.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// SanityProbe resume uma verificação de sanidade leve contra o banco:
// contagem de chunks, hits de full-text e hits de busca vetorial.
type SanityProbe struct {
	ChunkCount int64
	FTSHits    int
	VectorHits int
}

// Probe executa as verificações de sanidade do schema, sem depender de
// chamadas de embedding reais (usa um vetor zero de 4096 dims apenas para
// exercitar o ORDER BY `embedding <=> $1::vector`).
func Probe(ctx context.Context, dsn string) (*SanityProbe, error) {
	s, err := db.OpenDB(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// Vetor zero 4096-dim: válido para o operador <=>, dispensa embed.
	zero := make([]float64, 4096)

	var p SanityProbe
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM frr_docs;`).Scan(&p.ChunkCount); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM frr_docs WHERE content_tsv @@ plainto_tsquery('english', 'bgp neighbor');`,
	).Scan(&p.FTSHits); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM (SELECT 1 FROM frr_docs ORDER BY embedding <=> $1::vector LIMIT 3) x;`,
		retriever.VectorLiteral(zero),
	).Scan(&p.VectorHits); err != nil {
		return nil, err
	}

	return &p, nil
}
