package llm

import (
	"log"
	"os"

	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
)

func loadAgent() openai.ChatCompletionMessageParamUnion {
	systemPrompt, err := os.ReadFile(agent_prompt_path)
	if err != nil {
		log.Panic("Failed to load system prompt: ", err)
	}

	return openai.SystemMessage(string(systemPrompt))
}

func loadSkillsAndTools() []openai.ChatCompletionToolUnionParam {
	return tools.GetSkillAndTools()
}
