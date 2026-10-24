package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag/env"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
)

var log = logger.Named("MAIN")

func main() {
	cfg := logger.ConfigFromEnv()
	if cfg.File == "" {
		cfg.File = "logs"
	}
	logger.Init(cfg)

	config.SetCreator(createConfig)

	appCfg, err := config.Load()
	if err != nil {
		log.Fatalf("falha ao carregar a configuração (%s): %v", config.Path(), err)
	}

	agent, err := agentapi.Connect(appCfg)

	if err != nil {
		log.Fatalf("falha ao conectar ao LLM: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Network Agent started. Type your message (or 'exit' to quit):")
	reader := bufio.NewReader(os.Stdin)

	listas := rag.Lists()

	var transcript []history.Message

	// Estado entre turnos (fluxo E2E de Religar o fluxo E2E do CLI (08)):
	// a skip rule compara a classificação RAG do turno com a do anterior
	// (nil = sem anterior), e o contexto re-injetado vem do último retrieval
	// real ("" = nunca houve).
	var (
		classificacaoAnterior map[string]string
		ultimoContextoJSON    string
	)

	for {
		fmt.Print("\nUser: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Error("falha ao ler a entrada", "err", err)
			continue
		}
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			break
		}
		if userInput == "" {
			continue
		}

		// Classificação genérica — só exibida na tela: uma linha lista=valor
		// (ou "(vazio)") por lista. Nada mais depende dela.
		if generica, err := classifier.Do(ctx, userInput, listas, false, agent); err != nil {
			log.Warn("classificação genérica falhou — sem exibição", "err", err)
		} else {
			for _, l := range listas {
				valor := generica[l.Name]
				if valor == "" {
					valor = "(vazio)"
				}
				fmt.Printf("%s=%s\n", l.Name, valor)
			}
		}

		// Classificação RAG — o conjunto único de filtros do retrieval. Falha:
		// filtros vazios (idêntico a "nada se encaixa") e a comparação zera —
		// sem estado meio atualizado, o próximo turno busca de novo.
		classificacao, err := classifier.Do(ctx, userInput, listas, true, agent)
		var filtros rag.Filters
		if err != nil {
			log.Warn("classificador RAG falhou — seguindo com filtros vazios", "err", err)
			classificacao = nil
		} else {
			filtros = rag.FiltersFrom(classificacao)
		}

		// Skip rule: o retrieval só roda quando a classificação RAG difere do
		// turno anterior; pulado, o contexto vem do último retrieval real.
		contextoJSON := ""
		if deveBuscar(classificacao, classificacaoAnterior) {
			resultados, espelho, err := runRetrieval(ctx, agent, userInput, filtros)
			if err != nil && !errors.Is(err, ragretriever.ErrMirrorNotSaved) {
				log.Warn("retrieval falhou, seguindo sem contexto RAG", "err", err)
			} else {
				if err != nil {
					log.Warn("espelho do retrieval não gravado — seguindo com o contexto em memória", "err", err)
				} else {
					log.Info("retrieval salvo", "path", espelho)
				}
				contextoJSON = jsonDoContexto(resultados)
				ultimoContextoJSON = contextoJSON
			}
		} else {
			contextoJSON = ultimoContextoJSON
		}
		classificacaoAnterior = classificacao

		prompt := userInput
		if contextoJSON != "" {
			prompt = userInput + "\n\n<recovered_context>\n" + contextoJSON + "\n</recovered_context>"
		}
		resp, err := agent.Chat(ctx, transcript, prompt)

		if err != nil {
			log.Error("falha ao chamar o LLM", "err", err)
			continue
		}

		out := resp.OutputText()
		fmt.Printf("Agent: %s\n", out)

		transcript = append(transcript,
			history.Message{Role: history.RoleUser, Content: prompt, Status: history.StatusDone},
			history.Message{Role: history.RoleAgent, Content: out, Status: history.StatusDone},
		)
	}
}

// deveBuscar aplica a skip rule do fluxo E2E: o retriever só roda quando o
// resultado do Modo RAG difere do turno anterior — igualdade exata do mapa,
// vazio incluído. atual == nil é a falha do classificador e anterior == nil é
// "sem anterior" (primeiro turno ou comparação zerada): nos dois casos,
// sempre busca.
func deveBuscar(atual, anterior map[string]string) bool {
	return atual == nil || anterior == nil || !maps.Equal(atual, anterior)
}

// jsonDoContexto serializa os resultados do retrieval para o bloco
// <recovered_context> do prompt (e para o re-injetar quando a busca é pulada).
func jsonDoContexto(resultados []rag.Result) string {
	if resultados == nil {
		resultados = []rag.Result{}
	}
	data, err := json.MarshalIndent(resultados, "", "  ")
	if err != nil {
		log.Warn("falha ao serializar o contexto recuperado", "err", err)
		return ""
	}
	return string(data)
}

// runRetrieval monta a Config do retriever (DSN do DATABASE_URL, embedder
// = cliente de embedding próprio via LoadEmbedAgent, gerador de queries LLM)
// e executa ragretriever.Do para o input textual, com o conjunto de filtros
// do Classificador. A composição do turno (chamadas ao Classificador, skip
// rule, recovered_context) é de Religar o fluxo E2E do CLI (08).
func runRetrieval(ctx context.Context, agent *agentapi.Agent, input string, filtros rag.Filters) ([]rag.Result, string, error) {
	dsn, _ := env.LoadEnv()
	if dsn == "" {
		return nil, "", fmt.Errorf("DATABASE_URL ausente — adicione ao .env ou exporte no ambiente")
	}

	// Embeddings vêm do LoadEmbbedAgent (LM Studio em localhost: 1234),
	// que serve o MESMO modelo usado no ingester (2560 dims).
	embAgent := agentapi.LoadEmbbedAgent()
	cfg := rag.Config{
		DSN:            dsn,
		EmbeddingModel: embAgent.ModelName,
		Embedder:       embAgent.Client,
		QueryGen: &ragretriever.LLMQueryGenerator{
			Client:    agent.Client,
			ModelName: agent.ModelName,
		},
	}

	return ragretriever.Do(ctx, input, filtros, cfg)
}
