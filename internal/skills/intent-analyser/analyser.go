package intentanalyser

import (
	"context"
	"os"

	"github.com/Gabriel-Araujo/network_agent/internal/llm"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func Do(input string, agent *llm.Agent) *responses.Response {

	response, err := agent.Client.Responses.New(context.TODO(), responses.ResponseNewParams{
		Model:        agent.ModelName,
		Instructions: openai.String(sPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(skillPrompt, responses.EasyInputMessageRoleDeveloper),
				responses.ResponseInputItemParamOfMessage(input, responses.EasyInputMessageRoleUser),
			}},
		Tools: agent.AvailableTools,
	})

	if err != nil {
		panic(err)
	}

	os.WriteFile("./.agent/tmp/teste1.md", []byte(response.OutputText()), 0644)

	return response
}
