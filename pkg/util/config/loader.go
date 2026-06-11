package config

import (
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
	"github.com/pelletier/go-toml/v2"
)

func Load() provider_config.ProviderConfig {
	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	path := filepath.Dir(exec)

	raw, err := os.ReadFile(path + "/.config.toml")

	if err != nil {
		println("No config file detected. Starting config creator.")
		return CreateConfig()[0]
	}

	var config ConfigFile
	err = toml.Unmarshal(raw, &config)
	if err != nil {
		print("error")
		panic(err)
	}

	return config.Providers[0]
}
