package loaders

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func readInput() string {
	reader := bufio.NewReader(os.Stdin)

	text, err := reader.ReadString('\n')
	if err != nil {
		log.Panicf("falha ao ler a entrada: %v", err)
	}
	return strings.Trim(text, "\r\n ")
}

func getApiKey() string {
	fmt.Println("Set your apiKey")
	input := readInput()

	return input
}

func getModelName() string {
	fmt.Println("Write model name")
	input := readInput()

	return input
}

func getUrl() string {
	fmt.Println("Write URL name")
	input := readInput()

	return input
}

func saveConfig(config []ProviderConfig) ([]ProviderConfig, error) {
	providers := configFile{
		Providers: config,
	}

	b, err := toml.Marshal(providers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(fileLocation, b, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	return config, nil
}

func Load() (ProviderConfig, error) {
	raw, err := os.ReadFile(fileLocation)
	if err != nil {
		fmt.Println("No config file detected. Starting config creator.")
		configs, err := CreateConfig()
		if err != nil {
			return ProviderConfig{}, err
		}
		return configs[0], nil
	}

	var config configFile
	err = toml.Unmarshal(raw, &config)
	if err != nil {
		return ProviderConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if len(config.Providers) == 0 {
		return ProviderConfig{}, fmt.Errorf("no providers found in config")
	}

	return config.Providers[0], nil
}
