package llm

import (
	agent "github.com/Gabriel-Araujo/network_agent/internal"
	"github.com/openai/openai-go/v3"
)

type Llm struct {
	Client   openai.Client
	Messages []openai.ChatCompletionMessageParamUnion
}

func New(client openai.Client) Llm {
	return Llm{
		Client:   client,
		Messages: []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(agent.SYSTEM_PROMPT)},
	}
}
