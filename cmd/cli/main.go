package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag/env"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
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

	var transcript []history.Message

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

		/*intentPath, err := intentanalyser.Do(ctx, userInput, agent)
		if err != nil {
			log.Error("falha ao analisar a intenção", "err", err)
			continue
		}

		log.Debug("intent salvo", "path", intentPath)

		// Retrieval RAG: gera/extrai queries do briefing, busca no
		// vector híbrido e grava o JSON em.agent/tmp/retrieval.
		retrievalOut, err := runRetrieval(ctx, agent, intentPath)
		if err != nil {
			log.Warn("retrieval falhou, seguindo sem contexto RAG", "err", err)
			retrievalOut = ""
		}
		if retrievalOut != "" {
			log.Info("retrieval salvo", "path", retrievalOut)
		}

		if retrievalJSON := loadRetrievalJSON(retrievalOut); retrievalJSON != "" {
			prompt = userInput + "\n\n<recovered_context>\n" + retrievalJSON + "\n</recovered_context>"
		}*/
		prompt := userInput
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

// runRetrieval monta a Config do retriever (DSN do DATABASE_URL, embedder
// = cliente de embedding próprio via LoadEmbedAgent, fallback LLM "p"
// geração de queries) e executa rag. Do para o briefing gerado.
func runRetrieval(ctx context.Context, agent *agentapi.Agent, intentPath string) (string, error) {
	dsn, _ := env.LoadEnv()
	if dsn == "" {
		return "", fmt.Errorf("DATABASE_URL ausente — adicione ao .env ou exporte no ambiente")
	}

	// Embeddings vêm do LoadEmbedAgent (LM Studio em localhost: 1234),
	// que serve o MESMO modelo usado no ingester (4096 dims).
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

	return ragretriever.Do(ctx, intentPath, cfg)
}

// loadRetrievalJSON lê o JSON de retrieval gerado, se existir.
func loadRetrievalJSON(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
