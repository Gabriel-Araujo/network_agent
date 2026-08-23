package ragretriever

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	ragsearch "github.com/Gabriel-Araujo/network_agent/internal/llm/rag/retriever"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// Do lê o briefing JSON gerado pelo intent-analyser, extrai as queries de
// ragQueries (usando o fallback LLM quando necessário), executa a busca
// híbrida e salva o resultado em retrievalDir. Retorna o caminho do JSON.
func Do(ctx context.Context, briefingPath string, cfg rag.Config) (string, error) {
	content, err := os.ReadFile(briefingPath)
	if err != nil {
		return "", fmt.Errorf("lendo briefing %s: %w", briefingPath, err)
	}

	queries, err := BuildQueries(ctx, content, cfg.QueryGen)
	if err != nil {
		return "", fmt.Errorf("gerando queries do briefing: %w", err)
	}

	results, err := ragsearch.Retrieve(ctx, cfg, queries)
	if err != nil {
		return "", fmt.Errorf("buscando no pgvector: %w", err)
	}

	return SaveResults(briefingPath, results)
}

// SaveResults grava o resultado JSON em retrievalDir, seguindo a mesma
// convenção de nome do briefing (<arquivo-sem-ext>.json), e retorna o
// caminho do arquivo criado.
func SaveResults(briefingPath string, results []rag.Result) (string, error) {
	base := filepath.Base(briefingPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	name := base + ".json"

	dir, err := util.SafePath(".", rag.RetrievalDir)
	if err != nil {
		return "", err
	}

	jsonPath := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serializando resultados: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return "", fmt.Errorf("gravando %s: %w", jsonPath, err)
	}
	return jsonPath, nil
}

// BuildQueries extrai as queries do campo ragQueries do briefing JSON. Se o
// campo não existir ou estiver vazio, delega ao generator como fallback LLM.
// Quando não há generator disponível, retorna rag.ErrNoQueries.
func BuildQueries(ctx context.Context, content []byte, gen rag.QueryGenerator) ([]rag.QuerySuggestion, error) {
	queries, err := ParseQueriesFromBriefing(content)
	if err == nil {
		return queries, nil
	}
	if !errors.Is(err, rag.ErrNoQueries) {
		return nil, err
	}
	if gen == nil {
		return nil, rag.ErrNoQueries
	}
	return gen.GenerateQueries(ctx, content)
}

// LLMQueryGenerator implementa o fallback de geração de queries usando a API
// de Responses do openai-go.
type LLMQueryGenerator struct {
	Client    openai.Client
	ModelName string
}

// GenerateQueries pede ao modelo que reescreva o briefing em queries de busca
// para a tabela frr_docs e interpreta a resposta como JSON.
func (g *LLMQueryGenerator) GenerateQueries(ctx context.Context, briefing []byte) ([]rag.QuerySuggestion, error) {
	resp, err := g.Client.Responses.New(ctx, responses.ResponseNewParams{
		Model: g.ModelName,
		// Mesma restrição do intent-analyser: só pode haver system na primeira
		// posição, e o servidor mapeia developer para system.
		Instructions: openai.String(systemPrompt + "\n\n" + skillPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(string(briefing), responses.EasyInputMessageRoleUser),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return ParseGeneratedQueries(resp.OutputText())
}

type ragQueryJSON struct {
	Query              string `json:"query"`
	Protocol           string `json:"protocol"`
	Daemon             string `json:"daemon"`
	SuggestedChunkType string `json:"suggestedChunkType"`
}

type briefingJSON struct {
	RagQueries map[string]ragQueryJSON `json:"ragQueries"`
}

// ParseQueriesFromBriefing extrai, em ordem numérica, as queries do campo
// ragQueries produzido pelo intent-analyser. Valores de protocol, daemon e
// suggestedChunkType são normalizados para os filtros usados no banco.
func ParseQueriesFromBriefing(content []byte) ([]rag.QuerySuggestion, error) {
	var briefing briefingJSON
	if err := json.Unmarshal(content, &briefing); err != nil {
		return nil, rag.ErrNoQueries
	}
	if len(briefing.RagQueries) == 0 {
		return nil, rag.ErrNoQueries
	}

	keys := make([]string, 0, len(briefing.RagQueries))
	for key := range briefing.RagQueries {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keyLess(keys[i], keys[j]) })

	queries := make([]rag.QuerySuggestion, 0, len(keys))
	for _, key := range keys {
		suggestion := briefing.RagQueries[key]
		query := strings.TrimSpace(suggestion.Query)
		if query == "" {
			continue
		}

		chunkType := strings.ToLower(strings.TrimSpace(suggestion.SuggestedChunkType))
		switch chunkType {
		case rag.ChunkTypeCommandReference, rag.ChunkTypeConcept:
		default:
			chunkType = ""
		}

		queries = append(queries, rag.QuerySuggestion{
			Query:     query,
			Protocol:  strings.ToLower(strings.TrimSpace(suggestion.Protocol)),
			Daemon:    strings.ToLower(strings.TrimSpace(suggestion.Daemon)),
			ChunkType: chunkType,
		})
	}
	if len(queries) == 0 {
		return nil, rag.ErrNoQueries
	}
	return queries, nil
}

func keyLess(a, b string) bool {
	an, aerr := strconv.Atoi(a)
	bn, berr := strconv.Atoi(b)
	switch {
	case aerr == nil && berr == nil:
		return an < bn
	case aerr == nil:
		return true
	case berr == nil:
		return false
	default:
		return a < b
	}
}

var fenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

// ParseGeneratedQueries extrai o array JSON de queries da saída do fallback
// LLM, tolerando texto ao redor e code fences.
func ParseGeneratedQueries(text string) ([]rag.QuerySuggestion, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "[") {
		if queries, ok := parseQueriesJSON(text); ok {
			return queries, nil
		}
	}

	if match := fenceRe.FindStringSubmatch(text); match != nil {
		if queries, ok := parseQueriesJSON(match[1]); ok {
			return queries, nil
		}
	}

	return nil, errors.New("não foi possível extrair queries JSON da resposta do LLM")
}

type generatedQueryJSON struct {
	Query     string `json:"query"`
	Protocol  string `json:"protocol"`
	Daemon    string `json:"daemon"`
	ChunkType string `json:"chunk_type"`
}

func parseQueriesJSON(content string) ([]rag.QuerySuggestion, bool) {
	var raw []generatedQueryJSON
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, false
	}

	queries := make([]rag.QuerySuggestion, 0, len(raw))
	for _, item := range raw {
		query := strings.TrimSpace(item.Query)
		if query == "" {
			continue
		}

		chunkType := strings.ToLower(strings.TrimSpace(item.ChunkType))
		switch chunkType {
		case rag.ChunkTypeCommandReference, rag.ChunkTypeConcept:
		default:
			chunkType = ""
		}

		queries = append(queries, rag.QuerySuggestion{
			Query:     query,
			Protocol:  strings.ToLower(strings.TrimSpace(item.Protocol)),
			Daemon:    strings.ToLower(strings.TrimSpace(item.Daemon)),
			ChunkType: chunkType,
		})
	}
	return queries, true
}
