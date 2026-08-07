package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	intentanalyser "github.com/Gabriel-Araujo/network_agent/internal/skills/intent-analyser"
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

		out, err := agent.Chat(ctx, userInput)

		if err != nil {
			fmt.Printf("Error calling LLM: %v\n", err)
			continue
		}

		fmt.Printf("Agent: %s\n", out)
	}
}
