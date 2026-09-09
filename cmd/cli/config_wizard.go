package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
)

// Wizard de configuração do CLI: pergunta provedor, URL, chave e modelo pelo
// terminal. Vive aqui (e não em internal/config) porque é específico desta
// interface — a GUI usa a própria tela de Configurações.

func createConfig() (types.Config, error) {
	provider, err := createProvider()
	if err != nil {
		return types.Config{}, err
	}
	return types.Config{
		Providers: []types.Provider{provider},
	}, nil
}

func createProvider() (types.Provider, error) {
	name, err := selectProvider()
	if err != nil {
		return types.Provider{}, err
	}
	model, err := createModel()
	if err != nil {
		return types.Provider{}, err
	}
	return types.Provider{
		Name:         name,
		Url:          getUrl(),
		ApiKey:       getApiKey(),
		DefaultModel: 0,
		Models:       []types.Model{model},
	}, nil
}

func createModel() (types.Model, error) {
	return types.Model{
		Id:          getModelId(),
		Name:        getModelName(),
		ContextSize: setContextSize(),
	}, nil
}

func readInput() string {
	reader := bufio.NewReader(os.Stdin)

	text, err := reader.ReadString('\n')
	if err != nil {
		log.Errorf("failed to read from Terminal: %v", err)
		return ""
	}
	return strings.Trim(text, "\r\n ")
}

func selectProvider() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	providers := config.AvailableProviders()

	fmt.Println("You can choose between this providers")
	for i, p := range providers {
		fmt.Printf("[%d] %s\n", i+1, p)
	}

	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("falha ao ler a entrada: %w", err)
		}

		input, err := strconv.Atoi(strings.Trim(text, "\r\n "))
		if err != nil {
			fmt.Println("Invalid input. Try again.")
			continue
		}
		if input <= 0 || input > len(providers) {
			fmt.Printf("%v. Try again.\n", exception.InvalidInput)
			continue
		}
		return providers[input-1], nil
	}
}

func getUrl() string {
	fmt.Println("Write URL name")
	return readInput()
}

func getApiKey() string {
	fmt.Println("Set your apiKey")
	return readInput()
}

func getModelName() string {
	fmt.Println("Write model name")
	return readInput()
}

func getModelId() string {
	fmt.Println("Write model Id")
	return readInput()
}

func setContextSize() int64 {
	fmt.Println("Set model context size.")
	fmt.Println("0 for default")

	for {
		input, err := strconv.ParseInt(readInput(), 10, 64)
		if err != nil {
			fmt.Println("Invalid input. Try again.")
			continue
		}
		return input
	}
}
