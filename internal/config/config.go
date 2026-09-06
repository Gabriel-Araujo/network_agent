package config

import (
	"encoding/json/v2"
	"os"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
)

var log = logger.Named("CONFIG")

func CreateConfig() types.Config {
	return types.Config{
		Providers: []types.Provider{CreateProvider()},
	}
}

func CreateProvider() types.Provider {
	return types.Provider{
		Name:         selectProvider(),
		Url:          getUrl(),
		ApiKey:       getApiKey(),
		DefaultModel: 0,
		Models:       []types.Model{CreateModel()},
	}
}

func CreateModel() types.Model {
	return types.Model{
		Id:          getModelId(),
		Name:        getModelName(),
		ContextSize: setContextSize(),
	}
}

func Load() types.Config {
	raw, err := os.ReadFile(fileLocation)
	if err != nil {
		log.Info("No config file detected. Starting config creator.")
		return CreateConfig()
	}

	var config types.Config
	err = json.Unmarshal(raw, &config)
	if err != nil {
		log.Info("Failed to load existing config. Starting config creator.")
		return CreateConfig()
	}

	return config
}

func save(config types.Config) {
	raw, err := json.Marshal(config)
	if err != nil {
		log.Errorf("failed to marshal config: %s\n", err.Error())
	}
	if err := os.WriteFile(fileLocation, raw, 0644); err != nil {
		log.Errorf("failed to write config file: %s\n", err.Error())
	}
}
