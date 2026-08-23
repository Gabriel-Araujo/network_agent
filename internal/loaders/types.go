package loaders

import (
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("CONFIG")

var fileLocation = filepath.Join(os.TempDir(), "network_agent")

type ProviderConfig struct {
	ProviderName string
	ApiKey       string
	Url          string
	ModelName    string
}

var openAiConfig = func() ProviderConfig {
	return ProviderConfig{
		"OpenAI compatibly",
		getApiKey(),
		getUrl(),
		getModelName(),
	}
}

var providers = [...]string{"OpenAI compatibly"}

var builder = map[string]func() ProviderConfig{
	"OpenAI compatibly": openAiConfig,
}

type configFile struct {
	Providers []ProviderConfig `toml:"providers"`
}
