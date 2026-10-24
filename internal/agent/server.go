package agent

import (
	_ "embed"
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

//go:embed prompts/AGENT.md
var baseAgent string

func Connect(config types.Config) (*Agent, error) {
	provider := config.Providers[config.Selected]
	model := provider.Models[provider.DefaultModel]
	log.Debugf("Trying connection with Provider '%s' through '%s'\n",
		provider.Name, provider.Url)

	if err := ensureLMStudioContext(provider.Url, provider.ApiKey, model.Id, model.ContextSize); err != nil {
		return nil, fmt.Errorf("configurando o contexto do modelo %q: %w", model.Id, err)
	}

	// Instancia o cliente (retorna openai.Client)
	client := openai.NewClient(
		option.WithBaseURL(provider.Url),
		option.WithAPIKey(provider.ApiKey),
	)

	log.Info("conectado")

	return Init(
		client,
		provider.Url,
		provider.ApiKey,
		model.Id,
		model.ContextSize,
		loadSkillsAndTools(),
		baseAgent,
	), nil
}
