package querygen

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// BuildQueries retorna as queries para o briefing. Se o briefing contiver
// a seção "Rewritten queries for RAG", usa o parse determinístico; caso
// contrário, delega ao generator (fallback LLM). Quando gen é nil e o
// briefing não tem a tabela, retorna ErrNoQueriesTable.
func BuildQueries(ctx context.Context, content []byte, gen rag.QueryGenerator) ([]rag.QuerySuggestion, error) {
	qs, err := ParseQueriesFromBriefing(content)
	if err == nil {
		return qs, nil
	}
	if !errors.Is(err, rag.ErrNoQueriesTable) {
		return nil, err
	}
	if gen == nil {
		return nil, rag.ErrNoQueriesTable
	}
	return gen.GenerateQueries(ctx, content)
}

// LLMQueryGenerator implementa QueryGenerator usando a API de Responses
// do openai-go, no mesmo padrão da skill intentanalyser.
type LLMQueryGenerator struct {
	Client    openai.Client
	ModelName string
}

// GenerateQueries pede ao modelo que reescreva o briefing em queries de
// busca para a tabela frr_docs, retornando JSON puro.
func (g *LLMQueryGenerator) GenerateQueries(ctx context.Context, briefing []byte) ([]rag.QuerySuggestion, error) {
	resp, err := g.Client.Responses.New(ctx, responses.ResponseNewParams{
		Model:        g.ModelName,
		Instructions: openai.String(queryGenSystemPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(queryGenPrompt, responses.EasyInputMessageRoleDeveloper),
				responses.ResponseInputItemParamOfMessage(string(briefing), responses.EasyInputMessageRoleUser),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return ParseGeneratedQueries(resp.OutputText())
}

var fenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

// ParseGeneratedQueries extrai o array JSON de queries da saída do modelo,
// tolerando texto ao redor e code fences.
func ParseGeneratedQueries(text string) ([]rag.QuerySuggestion, error) {
	text = strings.TrimSpace(text)

	// Caso o modelo devolva apenas o JSON direto.
	if strings.HasPrefix(text, "[") {
		if qs, ok := parseQueriesJSON(text); ok {
			return qs, nil
		}
	}

	// Tenta achar um bloco de code fence json.
	if m := fenceRe.FindStringSubmatch(text); m != nil {
		if qs, ok := parseQueriesJSON(m[1]); ok {
			return qs, nil
		}
	}

	return nil, errors.New("não foi possível extrair queries JSON da resposta do LLM")
}

type genQueryJSON struct {
	Query     string `json:"query"`
	Protocol  string `json:"protocol"`
	Daemon    string `json:"daemon"`
	ChunkType string `json:"chunk_type"`
}

func parseQueriesJSON(s string) ([]rag.QuerySuggestion, bool) {
	var raw []genQueryJSON
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, false
	}
	qs := make([]rag.QuerySuggestion, 0, len(raw))
	for _, r := range raw {
		q := strings.TrimSpace(r.Query)
		if q == "" {
			continue
		}
		ct := strings.ToLower(strings.TrimSpace(r.ChunkType))
		switch ct {
		case rag.ChunkTypeCommandReference, rag.ChunkTypeConcept:
		default:
			ct = ""
		}
		qs = append(qs, rag.QuerySuggestion{
			Query:     q,
			Protocol:  strings.ToLower(strings.TrimSpace(r.Protocol)),
			Daemon:    strings.ToLower(strings.TrimSpace(r.Daemon)),
			ChunkType: ct,
		})
	}
	return qs, true
}
