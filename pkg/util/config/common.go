package config

import "github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"

type ConfigFile struct {
	Providers []provider_config.ProviderConfig `toml:"providers"`
}
