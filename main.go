package main

import (
	"context"
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	"github.com/Gabriel-Araujo/network_agent/internal/server"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/Gabriel-Araujo/network_agent/pkg/util/config"
	"github.com/openai/openai-go/v3"
)

func main() {
	llmConfig := config.Load()
	llmServer := llm.New(server.Connect(llmConfig))
	llmServer.Messages = append(llmServer.Messages, openai.UserMessage("Edit .config.toml file and put the comment 'hi'."))

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

		if toolCallChunk != nil {
			response := tools.Call_function(*toolCallChunk, true)
			fmt.Print(response.OfTool.Content.OfString.Value)

			llmServer.Messages = append(llmServer.Messages, acc.Choices[0].Message.ToParam())
			llmServer.Messages = append(llmServer.Messages, response)
		} else {
			break
		}
	}
}
