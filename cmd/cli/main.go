package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	agentapi "github.com/Gabriel-Araujo/network_agent/internal/llm/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/env"
	"github.com/Gabriel-Araujo/network_agent/internal/loaders"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	intentanalyser "github.com/Gabriel-Araujo/network_agent/internal/skills/intent-analyser"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
)

var log = logger.Named("MAIN")

func main() {
	// O cli roda como sessão: ganha arquivo de log por default. O if preserva
	// o override — sem ele, o main sobrescreveria LOG_FILE.
	cfg := logger.ConfigFromEnv()
	if cfg.File == "" {
		cfg.File = "logs"
	}
	logger.Init(cfg)

	llmConfig, err := loaders.Load()
	if err != nil {
		log.Fatalf("erro carregando configuração: %v", err)
	}

	// Mapeamento explícito para evitar erros de tipos de pacotes diferentes
	agent, err := llm.Connect(llm.Config{
		ModelName:    llmConfig.ModelName,
		Url:          llmConfig.Url,
		ApiKey:       llmConfig.ApiKey,
		ProviderName: llmConfig.ProviderName,
	})

	if err != nil {
		log.Fatalf("falha ao conectar ao LLM: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Network Agent started. Type your message (or 'exit' to quit):")

	reader := bufio.NewReader(os.Stdin)

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

		intentPath, err := intentanalyser.Do(ctx, userInput, agent)
		if err != nil {
			return
		}

		log.Info("intent salvo", "path", intentPath)

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

		prompt := userInput
		if retrievalJSON := loadRetrievalJSON(retrievalOut); retrievalJSON != "" {
			prompt = userInput + "\n\n<recovered_context>\n" + retrievalJSON + "\n</recovered_context>"
		}

		out, err := agent.Chat(ctx, prompt)

		if err != nil {
			log.Error("falha ao chamar o LLM", "err", err)
			continue
		}

		fmt.Printf("Agent: %s\n", out)
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
	embAgent := llm.LoadEmbbedAgent()
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
