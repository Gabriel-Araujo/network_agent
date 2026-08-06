package llm

import (
	"context"

	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type Agent struct {
	Client             openai.Client
	ModelName          string
	AvailableTools     []responses.ToolUnionParam
	systemPrompt       string
	previousResponseID string
}

func (a *Agent) Chat(ctx context.Context, userMessage string) (string, error) {
	input := responses.ResponseNewParamsInputUnion{}

	if a.previousResponseID == "" {
		input = responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(a.systemPrompt, responses.EasyInputMessageRoleSystem),
				responses.ResponseInputItemParamOfMessage(userMessage, responses.EasyInputMessageRoleUser),
			},
		}

	} else {
		input = responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		}
	}

	params := responses.ResponseNewParams{
		Model: a.ModelName,
		Input: input,
		Tools: a.AvailableTools,
	}

	if a.previousResponseID != "" {
		params.PreviousResponseID = openai.String(a.previousResponseID)
	}

	for {

		resp, err := a.Client.Responses.New(ctx, params)
		if err != nil {
			return "", err
		}

		a.previousResponseID = resp.ID

		outputs := []responses.ResponseInputItemUnionParam{}

		finished := true

		for _, item := range resp.Output {

			switch item.Type {

			case "function_call":

				finished = false

				result := tools.CallFunction(ctx, item.AsFunctionCall())

				outputs = append(outputs, result)

			case "message":

				// será retornado quando não houver mais tool
			}
		}

		if finished {
			return resp.OutputText(), nil
		}

		params = responses.ResponseNewParams{
			Model:              a.ModelName,
			PreviousResponseID: openai.String(resp.ID),
			Input:              responses.ResponseNewParamsInputUnion{OfInputItemList: outputs},
			Tools:              a.AvailableTools,
		}
	}
}
