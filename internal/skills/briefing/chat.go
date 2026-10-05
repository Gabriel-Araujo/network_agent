package briefing

import (
	"context"
	"errors"

	llm "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/openai/openai-go/v3"
)

// completarDoAgent fala com o provedor pelo /chat/completions — o mesmo canal
// do Classificador (Pesquisa: saída estruturada para classificação por
// listas). Sem response_format: a estrutura vem do prompt e a garantia de
// saída é a validação no cliente — o provedor pode ignorar o schema em
// silêncio, e o shape do briefing (mapa de chaves dinâmicas) não é expressível
// em schema estrito.
func completarDoAgent(agent *llm.Agent) completarFunc {
	return func(ctx context.Context, msgs []mensagem) (string, error) {
		resp, err := agent.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       agent.ModelName,
			Messages:    paramsDeMensagens(msgs),
			Temperature: openai.Float(0),
		})
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("provedor devolveu resposta sem escolhas")
		}
		return resp.Choices[0].Message.Content, nil
	}
}

func paramsDeMensagens(msgs []mensagem) []openai.ChatCompletionMessageParamUnion {
	params := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Papel {
		case "system":
			params = append(params, openai.SystemMessage(m.Conteudo))
		case "assistant":
			params = append(params, openai.AssistantMessage(m.Conteudo))
		default:
			params = append(params, openai.UserMessage(m.Conteudo))
		}
	}
	return params
}
