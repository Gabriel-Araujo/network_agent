package classifier

import (
	"context"
	"errors"

	llm "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

// completarDoAgent fala com o provedor pelo /chat/completions com o schema em
// response_format. É de propósito chat/completions e não /responses: o LM
// Studio ignora text.format em silêncio no /responses e só força o schema no
// chat (Pesquisa: saída estruturada para classificação por listas). Em todo
// caso, a garantia de saída é a validação no cliente.
func completarDoAgent(agent *llm.Agent) completarFunc {
	return func(ctx context.Context, msgs []mensagem, schema map[string]any) (string, error) {
		resp, err := agent.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       agent.ModelName,
			Messages:    paramsDeMensagens(msgs),
			Temperature: openai.Float(0),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
					JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:   "classification",
						Strict: openai.Bool(true),
						Schema: schema,
					},
				},
			},
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
