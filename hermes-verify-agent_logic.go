package main

import (
	"fmt"
	"github.com/Gabriel-Araujo/network-agent/internal/llm"
	"github.com/openai/openai-go/v3"
)

func main() {
	// Inicializa um agente com um cliente dummy
	agent := &llm.Agent{
		Client:    &openai.Client{},
		ModelName: "test-model",
		Messages:  []openai.ChatCompletionMessageParamUnion{},
	}

	// Teste 1: Verificar se ToolCall adiciona mensagens corretamente sem dar Panic
	acc := openai.ChatCompletionAccumulator{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "test_content",
				},
			},
		},
	}
	
	dummyChunk := &openai.ChatCompletionChunkChoiceDeltaToolCall{
		Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
			Name:      "test_tool",
			Arguments: `{"arg": "val"}`,
		},
		ID: "tool-123",
	}

	agent.ToolCall(dummyChunk, acc)

	if len(agent.Messages) == 2 {
		fmt.Println("✅ ToolCall: Success (2 messages appended)")
	} else {
		fmt.Printf("❌ ToolCall: Failed (expected 2, got %d)\n", len(agent.Messages))
		panic("Verification failed")
	}

	// Teste 2: Verificar se AskStream inicializa corretamente (não deve dar panic)
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ AskStream: Panic detected: %v\n", r)
			panic(r)
		}
	}()
	agent.AskStream("test question")
	fmt.Println("✅ AskStream: Initialization Success")
}
