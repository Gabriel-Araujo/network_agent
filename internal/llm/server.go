package llm

import (
	"log"

	llm "github.com/Gabriel-Araujo/network_agent/internal/llm/agent"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func Connect(config Config) (*llm.Agent, error) {
	log.Printf("Trying connection with model '%s' through '%s' (%s)\n", config.ModelName, config.ProviderName, config.Url)

	// Instancia o cliente (retorna openai.Client)
	client := openai.NewClient(
		option.WithBaseURL(config.Url),
		option.WithAPIKey(config.ApiKey),
	)

	log.Println("Connected and verified.")
	return &llm.Agent{
		Client:         client,
		ModelName:      config.ModelName,
		AvailableTools: loadSkillsAndTools(),
		SystemPrompt:   loadAgent(),
	}, nil
}
