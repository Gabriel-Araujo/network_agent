package agent

import (
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3/responses"
)

var log = logger.Named("MODELS")

func LoadTestAgent() *Agent {
	_agent, err := Connect(getTestConfig())

	if err != nil {
		log.Panicf("falha ao carregar o agente de teste: %v", err)
	}

	return _agent
}

func LoadEmbbedAgent() *Agent {
	_agent, err := Connect(getEmbedConfig())

	if err != nil {
		log.Panicf("falha ao carregar o agente de embedding: %v", err)
	}

	return _agent
}

func loadSkillsAndTools() []responses.ToolUnionParam {
	return tools.GetSkillAndTools()
}
