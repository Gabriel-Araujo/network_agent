package agent

import (
	_ "embed"

	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

//go:embed prompts/AGENT.md
var baseAgent string

func Connect(config types.Config) (*Agent, error) {
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
	return Init(
		client,
		model.Id,
		model.ContextSize,
		loadSkillsAndTools(),
		baseAgent,
	), nil
}
