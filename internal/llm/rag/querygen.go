package rag

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// QueryGenerator gera sugestões de query a partir do texto do briefing,
// usado como fallback quando o briefing não traz a tabela determinística.
type QueryGenerator interface {
	GenerateQueries(ctx context.Context, briefing []byte) ([]QuerySuggestion, error)
}

// BuildQueries retorna as queries para o briefing. Se o briefing contiver
// a seção "Rewritten queries for RAG", usa o parse determinístico; caso
// contrário, delega ao generator (fallback LLM). Quando gen é nil e o
// briefing não tem a tabela, retorna ErrNoQueriesTable.
func BuildQueries(ctx context.Context, content []byte, gen QueryGenerator) ([]QuerySuggestion, error) {
	qs, err := ParseQueriesFromBriefing(content)
	if err == nil {
		return qs, nil
	}
	if !errors.Is(err, ErrNoQueriesTable) {
		return nil, err
	}
	if gen == nil {
		return nil, ErrNoQueriesTable
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
func (g *LLMQueryGenerator) GenerateQueries(ctx context.Context, briefing []byte) ([]QuerySuggestion, error) {
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
	return parseGeneratedQueries(resp.OutputText())
}

var fenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

// parseGeneratedQueries extrai o array JSON de queries da saída do modelo,
// tolerando texto ao redor e code fences.
func parseGeneratedQueries(text string) ([]QuerySuggestion, error) {
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

func parseQueriesJSON(s string) ([]QuerySuggestion, bool) {
	var raw []genQueryJSON
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, false
	}
	qs := make([]QuerySuggestion, 0, len(raw))
	for _, r := range raw {
		q := strings.TrimSpace(r.Query)
		if q == "" {
			continue
		}
		ct := strings.ToLower(strings.TrimSpace(r.ChunkType))
		switch ct {
		case chunkTypeCommandReference, chunkTypeConcept:
		default:
			ct = ""
		}
		qs = append(qs, QuerySuggestion{
			Query:     q,
			Protocol:  strings.ToLower(strings.TrimSpace(r.Protocol)),
			Daemon:    strings.ToLower(strings.TrimSpace(r.Daemon)),
			ChunkType: ct,
		})
	}
	return qs, true
}
