package ragretriever

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag/retriever"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

var log = logger.Named("RAG-RETRIEVER")

// ErrEspelhoNaoGravado sinaliza a falha de gravação do espelho em
// .agent/tmp/retrieval: o valor em memória volta completo e o path vem
// vazio — o espelho nunca derruba o caminho (checável com errors.Is).
var ErrEspelhoNaoGravado = errors.New("espelho do retrieval não gravado")

// Do gera as queries a partir do input textual (caminho primário; degrada com
// aviso explícito para uma query única do input cru), executa a busca híbrida
// com o conjunto único de filtros e grava o espelho
// .agent/tmp/retrieval/<slug>-<hash8>.json. Retorna os resultados em memória,
// o path do espelho e o erro — sentinela ErrEspelhoNaoGravado quando só o
// espelho falhou (resultados completos e path vazio nesse caso).
func Do(ctx context.Context, input string, filtros rag.Filtros, cfg rag.Config) ([]rag.Result, string, error) {
	queries := BuildQueries(ctx, input, filtros, cfg.QueryGen)

	results, err := retriever.Retrieve(ctx, cfg, queries, filtros)
	if err != nil {
		return nil, "", fmt.Errorf("buscando no pgvector: %w", err)
	}

	path, err := SaveResults(input, results)
	if err != nil {
		log.Warn("espelho do retrieval não gravado — resultados devolvidos em memória", "err", err)
		return results, "", fmt.Errorf("%w: %w", ErrEspelhoNaoGravado, err)
	}
	return results, path, nil
}

// SaveResults grava o JSON de []rag.Result em .agent/tmp/retrieval com o nome
// <slug>-<hash8>.json derivado do input — molde do briefing: o mesmo input
// regrava o mesmo espelho (sobrescreve, sem histórico de execuções).
// Retorna o caminho do arquivo criado.
func SaveResults(input string, results []rag.Result) (string, error) {
	name := util.Slugify(input) + "-" + util.Hash8(input) + ".json"

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

// BuildQueries monta as queries da busca: o gerador é o caminho primário
// (input + filtros → queries reescritas); falhou a geração — ou não há
// gerador — a busca degrada com aviso explícito para uma única query do input
// cru. Nunca falha silenciosa; nunca a busca inteira se perde por causa da
// reescrita.
func BuildQueries(ctx context.Context, input string, filtros rag.Filtros, gen rag.QueryGenerator) []rag.QuerySuggestion {
	if gen != nil {
		queries, err := gen.GenerateQueries(ctx, input, filtros)
		switch {
		case err != nil:
			log.Warn("geração de queries falhou — buscando com o input cru", "err", err)
		case len(queries) == 0:
			log.Warn("geração de queries devolveu vazio — buscando com o input cru")
		default:
			return queries
		}
	} else {
		log.Warn("gerador de queries ausente — buscando com o input cru")
	}
	return []rag.QuerySuggestion{{Query: input}}
}

// LLMQueryGenerator implementa a geração de queries (caminho primário) usando
// a API de Responses do openai-go.
type LLMQueryGenerator struct {
	Client    openai.Client
	ModelName string
}

// GenerateQueries pede ao modelo que reescreva o input em queries de busca
// para a tabela frr_docs, recebendo os filtros escolhidos (e o vocabulário
// das Listas) para orientar a reescrita.
func (g *LLMQueryGenerator) GenerateQueries(ctx context.Context, input string, filtros rag.Filtros) ([]rag.QuerySuggestion, error) {
	resp, err := g.Client.Responses.New(ctx, responses.ResponseNewParams{
		Model: g.ModelName,
		// Mesma restrição do intent-analyser: só pode haver system na primeira
		// posição, e o servidor mapeia developer para system.
		Instructions: openai.String(montarPrompt(filtros)),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(input, responses.EasyInputMessageRoleUser),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return ParseGeneratedQueries(resp.OutputText())
}

var fenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

// ParseGeneratedQueries extrai o array JSON de queries (strings) da saída do
// gerador, tolerando texto ao redor e code fences. Strings vazias são
// descartadas.
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

func parseQueriesJSON(content string) ([]rag.QuerySuggestion, bool) {
	var raw []string
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, false
	}

	queries := make([]rag.QuerySuggestion, 0, len(raw))
	for _, item := range raw {
		query := strings.TrimSpace(item)
		if query == "" {
			continue
		}
		queries = append(queries, rag.QuerySuggestion{Query: query})
	}
	return queries, true
}
