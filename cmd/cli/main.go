package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	"github.com/Gabriel-Araujo/network_agent/pkg/util/config"
	"github.com/openai/openai-go/v3"
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

	fmt.Println("Network Agent started. Type your message (or 'exit' to quit):")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nUser: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading input: %v\n", err)
			continue
		}

		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			break
		}
		if userInput == "" {
			continue
		}

		agent.Messages = append(agent.Messages, openai.UserMessage(userInput))

		for {
			stream := agent.Client.Chat.Completions.NewStreaming(
				context.Background(),
				openai.ChatCompletionNewParams{
					Model:    agent.ModelName,
					Messages: agent.Messages,
					Tools:    agent.AvailableTools,
				},
			)

			acc := openai.ChatCompletionAccumulator{}

			var toolCallChunk *openai.ChatCompletionChunkChoiceDeltaToolCall

			fmt.Print("Agent: ")
			for stream.Next() {
				chunk := stream.Current()
				acc.AddChunk(chunk)

				if tool, ok := acc.JustFinishedToolCall(); ok {
					toolCallChunk = &openai.ChatCompletionChunkChoiceDeltaToolCall{
						ID: tool.ID,
						Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
							Name:      tool.Name,
							Arguments: tool.Arguments,
						},
					}
				}
				if len(chunk.Choices) > 0 {
					fmt.Print(chunk.Choices[0].Delta.Content)
				}
			}
			fmt.Println()

			if toolCallChunk != nil {
				agent.ToolCall(toolCallChunk, acc)
				break
			} else {
				agent.Messages = append(agent.Messages, acc.Choices[0].Message.ToParam())
				break
			}
		}
	}
}
