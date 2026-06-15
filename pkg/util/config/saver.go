package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
	"github.com/pelletier/go-toml/v2"
)

func SaveConfig(config []provider_config.ProviderConfig) ([]provider_config.ProviderConfig, error) {
	providers := ConfigFile{
		Providers: config,
	}

	b, err := toml.Marshal(providers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	exec, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	path := filepath.Dir(exec)
	configPath := filepath.Join(path, ".config.toml")
	if err := os.WriteFile(configPath, b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	return config, nil
}
