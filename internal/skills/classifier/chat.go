package classifier

import (
	"context"
	"errors"

	llm "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

// completeWithAgent calls the provider through /chat/completions with the
// schema in response_format. LM Studio silently ignores text.format on
// /responses and only enforces the schema on chat; client-side validation is
// still the final guarantee.
func completeWithAgent(agent *llm.Agent) completeFunc {
	return func(ctx context.Context, msgs []message, schema map[string]any) (string, error) {
		resp, err := agent.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       agent.ModelName,
			Messages:    messageParams(msgs),
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
			return "", errors.New("provider returned a response without choices")
		}
		return resp.Choices[0].Message.Content, nil
	}
}

func messageParams(msgs []message) []openai.ChatCompletionMessageParamUnion {
	params := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			params = append(params, openai.SystemMessage(m.Content))
		case "assistant":
			params = append(params, openai.AssistantMessage(m.Content))
		default:
			params = append(params, openai.UserMessage(m.Content))
		}
	}
	return params
}
