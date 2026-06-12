package server

import (
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func Connect(config provider_config.ProviderConfig) openai.Client {
	fmt.Printf("Trying connection with model '%s' through '%s' (%s)\n", config.ModelName, config.ProviderName, config.Url)

	client := openai.NewClient(option.WithBaseURL(config.Url), option.WithAPIKey(config.ApiKey))

	println("Connected")

	return client
}
