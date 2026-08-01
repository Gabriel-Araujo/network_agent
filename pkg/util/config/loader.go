package config

import (
	"fmt"
	"os"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
	"github.com/pelletier/go-toml/v2"
)

func Load() (provider_config.ProviderConfig, error) {
	raw, err := os.ReadFile(file_location)
	if err != nil {
		println("No config file detected. Starting config creator.")
		configs, err := CreateConfig()
		if err != nil {
			return provider_config.ProviderConfig{}, err
		}
		return configs[0], nil
	}

	var config ConfigFile
	err = toml.Unmarshal(raw, &config)
	if err != nil {
		return provider_config.ProviderConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if len(config.Providers) == 0 {
		return provider_config.ProviderConfig{}, fmt.Errorf("no providers found in config")
	}

	return config.Providers[0], nil
}
