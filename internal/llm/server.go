package llm

import (
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Config struct {
	ModelName    string
	Url          string
	ApiKey       string
	ProviderName string
}

func Connect(config Config) (*Agent, error) {
	log.Printf("Trying connection with model '%s' through '%s' (%s)\n", config.ModelName, config.ProviderName, config.Url)

	// Instancia o cliente (retorna openai.Client)
	client := openai.NewClient(
		option.WithBaseURL(config.Url),
		option.WithAPIKey(config.ApiKey),
	)

	log.Println("Connected and verified.")
	return &Agent{
		Client:         client,
		ModelName:      config.ModelName,
		Messages:       []openai.ChatCompletionMessageParamUnion{loadAgent()},
		AvailableTools: loadSkillsAndTools(),
	}, nil
}
