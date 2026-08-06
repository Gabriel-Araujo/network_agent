package llm

import (
	"log"
	"os"

	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3/responses"
)

func loadAgent() string {
	systemPrompt, err := os.ReadFile(agent_prompt_path)
	if err != nil {
		log.Panic("Failed to load system prompt: ", err)
	}

	return string(systemPrompt)
}

func loadSkillsAndTools() []responses.ToolUnionParam {
	return tools.GetSkillAndTools()
}
