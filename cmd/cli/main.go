package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	agentapi "github.com/Gabriel-Araujo/network_agent/internal/llm/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/env"
	intentanalyser "github.com/Gabriel-Araujo/network_agent/internal/skills/intent-analyser"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
	"github.com/Gabriel-Araujo/network_agent/pkg/util/config"
)

func main() {
	llmConfig, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}

	// Mapeamento explícito para evitar erros de tipos de pacotes diferentes
	agent, err := llm.Connect(llm.Config{
		ModelName:    llmConfig.ModelName,
		Url:          llmConfig.Url,
		ApiKey:       llmConfig.ApiKey,
		ProviderName: llmConfig.ProviderName,
	})

	if err != nil {
		log.Fatalf("Failed to connect to LLM: %v\n", err)
	}

	ctx := context.Background()

	fmt.Println("Network Agent started. Type your message (or 'exit' to quit):")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nUser: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
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

		log.Println("MAIN - intent saved at: " + intentPath)

		// Retrieval RAG: gera/extrai queries do briefing, busca no
		// vector híbrido e grava o JSON em.agent/tmp/retrieval.
		retrievalOut, err := runRetrieval(ctx, agent, intentPath)
		if err != nil {
			log.Printf("MAIN - retrieval falhou (seguindo sem contexto RAG): %v\n", err)
			retrievalOut = ""
		}
		if retrievalOut != "" {
			log.Println("MAIN - retrieval saved at: " + retrievalOut)
		}

		prompt := userInput
		if retrievalJSON := loadRetrievalJSON(retrievalOut); retrievalJSON != "" {
			prompt = userInput + "\n\n<recovered_context>\n" + retrievalJSON + "\n</recovered_context>"
		}

		out, err := agent.Chat(ctx, prompt)

		if err != nil {
			fmt.Printf("Error calling LLM: %v\n", err)
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
