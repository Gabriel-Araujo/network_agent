package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	"github.com/Gabriel-Araujo/network_agent/internal/server"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/Gabriel-Araujo/network_agent/pkg/util/config"
	"github.com/openai/openai-go/v3"
)

func main() {
	llmConfig := config.Load()
	llmServer := llm.New(server.Connect(llmConfig))

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

		llmServer.Messages = append(llmServer.Messages, openai.UserMessage(userInput))

		for {
			stream := llmServer.Client.Chat.Completions.NewStreaming(
				context.Background(),
				openai.ChatCompletionNewParams{
					Model:    llmConfig.ModelName,
					Messages: llmServer.Messages,
					Tools:    tools.Tools,
				})

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
				response := tools.CallFunction(*toolCallChunk, true)

				llmServer.Messages = append(llmServer.Messages, acc.Choices[0].Message.ToParam())
				llmServer.Messages = append(llmServer.Messages, response)
			} else {
				llmServer.Messages = append(llmServer.Messages, acc.Choices[0].Message.ToParam())
				break
			}
		}
	}
}
