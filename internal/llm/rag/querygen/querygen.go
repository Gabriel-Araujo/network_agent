package querygen

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// BuildQueries retorna as queries para o briefing. Tenta o parse
// determinístico do JSON (campo "ragQueries"); se o briefing não trouxer
// queries, delega ao generator (fallback LLM). Quando gen é nil e não há
// queries, retorna ErrNoQueries.
func BuildQueries(ctx context.Context, content []byte, gen rag.QueryGenerator) ([]rag.QuerySuggestion, error) {
	qs, err := ParseQueriesFromBriefing(content)
	if err == nil {
		return qs, nil
	}
	if !errors.Is(err, rag.ErrNoQueries) {
		return nil, err
	}
	if gen == nil {
		return nil, rag.ErrNoQueries
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

// ragQueryJSON espelha um item do map "ragQueries" do briefing JSON.
type ragQueryJSON struct {
	Query              string `json:"query"`
	Protocol           string `json:"protocol"`
	Daemon             string `json:"daemon"`
	SuggestedChunkType string `json:"suggestedChunkType"`
}

// briefingJSON apresenta apenas os campos do briefing JSON que interessam
// à geração de queries.
type briefingJSON struct {
	RagQueries map[string]ragQueryJSON `json:"ragQueries"`
}

// ParseQueriesFromBriefing extrai as queries de busca do briefing JSON
// produzido pelo intent-analyser, lendo o campo "ragQueries"
// (chave numérica -> sugestão {query, protocol, daemon,
// suggestedChunkType}). A ordem é preservada pelas chaves numéricas
// (1,2,3,…); chaves não-numéricas vêm depois, por ordem lexicográfica.
// Retorna ErrNoQueries quando o briefing não é JSON, não tem "ragQueries"
// ou todas as sugestões estão vazias.
func ParseQueriesFromBriefing(content []byte) ([]rag.QuerySuggestion, error) {
	var b briefingJSON
	if err := json.Unmarshal(content, &b); err != nil {
		return nil, rag.ErrNoQueries
	}
	if len(b.RagQueries) == 0 {
		return nil, rag.ErrNoQueries
	}

	keys := make([]string, 0, len(b.RagQueries))
	for k := range b.RagQueries {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keyLess(keys[i], keys[j]) })

	qs := make([]rag.QuerySuggestion, 0, len(keys))
	for _, k := range keys {
		s := b.RagQueries[k]
		q := strings.TrimSpace(s.Query)
		if q == "" {
			continue
		}
		ct := strings.ToLower(strings.TrimSpace(s.SuggestedChunkType))
		switch ct {
		case rag.ChunkTypeCommandReference, rag.ChunkTypeConcept:
		default:
			ct = ""
		}
		qs = append(qs, rag.QuerySuggestion{
			Query:     q,
			Protocol:  strings.ToLower(strings.TrimSpace(s.Protocol)),
			Daemon:    strings.ToLower(strings.TrimSpace(s.Daemon)),
			ChunkType: ct,
		})
	}
	if len(qs) == 0 {
		return nil, rag.ErrNoQueries
	}
	return qs, nil
}

// keyLess ordena chaves: numéricas primeiro (1,2,10…), demais por ordem
// lexicográfica.
func keyLess(a, b string) bool {
	an, aerr := strconv.Atoi(a)
	bn, berr := strconv.Atoi(b)
	switch {
	case aerr == nil && berr == nil:
		return an < bn
	case aerr == nil:
		return true // numéricas antes
	case berr == nil:
		return false
	default:
		return a < b
	}
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
