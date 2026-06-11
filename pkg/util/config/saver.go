package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
	"github.com/pelletier/go-toml/v2"
)

func encoder(m map[string]string) []byte {
	var b strings.Builder

	for key, value := range m {
		fmt.Fprintf(&b, "%s=%s\n", key, value)
	}

	return []byte(b.String())
}

func SaveConfig(config []provider_config.ProviderConfig) []provider_config.ProviderConfig {
	providers := ConfigFile{
		Providers: config,
	}
	b, err := toml.Marshal(providers)
	if err != nil {
		panic("aaaa")
	}
	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	path := filepath.Dir(exec)
	err = os.WriteFile(fmt.Sprintf("%s/.config.toml", path), b, 0644)
	return config
}
