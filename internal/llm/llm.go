package llm

import (
	_ "embed"
	"log"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3/responses"
)

//go:embed agent/prompts/system-prompt.md
var systemPrompt string

func loadAgent() string {
	return systemPrompt
}

func LoadTestAgent() *llm.Agent {
	_agent, err := Connect(envConfig())

	if err != nil {
		log.Panic("Failed to load test agent: ", err)
	}

	return _agent
}

func LoadEmbbedAgent() *llm.Agent {
	_agent, err := Connect(Config{
		ModelName:    "fastiraz/text-embedding-qwen3-embedding-8b",
		Url:          "http://localhost:1234/v1",
		ApiKey:       "none",
		ProviderName: "LM Studio",
	})

	if err != nil {
		log.Panic("Failed to load test agent: ", err)
	}

	return _agent
}

func loadSkillsAndTools() []responses.ToolUnionParam {
	return tools.GetSkillAndTools()
}
