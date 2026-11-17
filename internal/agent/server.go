package agent

import (
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Connect monta o Agent da configuração escolhida. É o ponto de entrada:
// classifica e despacha, nunca responde, então não leva system prompt nem
// tools — quem responde é o Subagente que ForSubagent deriva dele. workspace é a pasta do
// usuário, e é parâmetro em vez de campo do types.Config de propósito (ticket
// 05, Q3): o arquivo de configuração é compartilhado com o cmd/cli, e fazer o
// CLI honrar uma pasta escolhida noutro app redirecionaria a escrita para longe
// de onde o usuário está parado.
func Connect(config types.Config, workspace string) (*Agent, error) {
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
		nil,
		"",
		workspace,
	), nil
}
