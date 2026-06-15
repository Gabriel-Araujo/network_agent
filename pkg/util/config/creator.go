package config

import (
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
	"github.com/Gabriel-Araujo/network_agent/pkg/util/provider_config"
)

var PROVIDERS = [...]string{"LM Studio"}

var BUILDER = map[string]func() provider_config.ProviderConfig{
	"LM Studio": provider_config.LMStudioConfig,
}

func showProviderOptions() {
	println("You can choose between this providers")

	for _, i := range PROVIDERS {
		println("[1] %s", i)
	}
}

func getProviderName() string {
	var input int

	_, err := fmt.Scanf("%d", &input)

	for {
		if input <= 0 || input > len(PROVIDERS) {
			err = exception.InvalidInput
		}
		if err == nil {
			break
		}
		println("Invalid input. Try again.")
		_, err = fmt.Scanf("%d", &input)

	}

	return PROVIDERS[input-1]
}

func getProvider() string {
	showProviderOptions()
	return getProviderName()
}

func CreateConfig() ([]provider_config.ProviderConfig, error) {
	name := getProvider()
	return SaveConfig([]provider_config.ProviderConfig{BUILDER[name]()})
}
