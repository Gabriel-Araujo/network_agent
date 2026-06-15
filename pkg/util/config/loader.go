package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
	"github.com/pelletier/go-toml/v2"
)

func Load() (provider_config.ProviderConfig, error) {
	exec, err := os.Executable()
	if err != nil {
		return provider_config.ProviderConfig{}, fmt.Errorf("failed to get executable path: %w", err)
	}

	path := filepath.Dir(exec)
	configPath := filepath.Join(path, ".config.toml")

	raw, err := os.ReadFile(configPath)
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
