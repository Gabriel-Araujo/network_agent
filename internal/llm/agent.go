package llm

import (
	"context"
	"fmt"
	"log"

	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
)

type Agent struct {
	Client         openai.Client
	ModelName      string
	Messages       []openai.ChatCompletionMessageParamUnion
	AvailableTools []openai.ChatCompletionToolUnionParam
}

// ToolCall processa a execução de ferramentas quando o LLM solicita uma chamada de função.
func (agent *Agent) ToolCall(toolCallChunk *openai.ChatCompletionChunkChoiceDeltaToolCall, acc openai.ChatCompletionAccumulator) {
	if toolCallChunk == nil || toolCallChunk.Function.Name == "" {
		return
	}

	log.Printf("[Tool Call: %s]\n", toolCallChunk.Function.Name)

	// Executa a função da ferramenta
	response := tools.CallFunction(*toolCallChunk, true)

	// Verifica se há escolhas antes de acessar o índice 0 para evitar Panic
	if len(acc.Choices) > 0 {
		agent.Messages = append(agent.Messages, acc.Choices[0].Message.ToParam())
	}

	agent.Messages = append(agent.Messages, response)
}

// AskStream envia uma pergunta ao LLM e processa a resposta em tempo real (streaming).
func (agent *Agent) AskStream(question string) {
	agent.Messages = append(agent.Messages, openai.UserMessage(question))

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

		// Verifica se o modelo terminou uma chamada de ferramenta
		if tool, ok := acc.JustFinishedToolCall(); ok {
			toolCallChunk = &openai.ChatCompletionChunkChoiceDeltaToolCall{
				ID: tool.ID,
				Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
					Name:      tool.Name,
					Arguments: tool.Arguments,
				},
			}
		}

		// Imprime o conteúdo da mensagem em tempo real
		if len(chunk.Choices) > 0 {
			// No v3, o Content é uma string, não um ponteiro.
			// Verificamos se a string não está vazia.
			if chunk.Choices[0].Delta.Content != "" {
				fmt.Print(chunk.Choices[0].Delta.Content)
			}
		}
	}
	fmt.Println()

	// Lógica de pós-processamento para ferramentas
	if toolCallChunk != nil && toolCallChunk.Function.Name != "" {
		agent.ToolCall(toolCallChunk, acc)
	} else if len(acc.Choices) > 0 {
		agent.Messages = append(agent.Messages, acc.Choices[0].Message.ToParam())
	}
}
