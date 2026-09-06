package llm

import (
	llm "github.com/Gabriel-Araujo/network_agent/internal/llm/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func Connect(config types.Config) (*llm.Agent, error) {
	provider := config.Providers[config.Selected]
	log.Debugf("Trying connection with Provider '%s' through '%s'\n",
		provider.Name, provider.Url)

	// Instancia o cliente (retorna openai.Client)
	client := openai.NewClient(
		option.WithBaseURL(provider.Url),
		option.WithAPIKey(provider.ApiKey),
	)

	log.Info("conectado")

	model := provider.Models[provider.DefaultModel]
	return llm.Init(
		client,
		model.Id,
		model.ContextSize,
		loadSkillsAndTools(),
		loadAgent(),
	), nil
}
