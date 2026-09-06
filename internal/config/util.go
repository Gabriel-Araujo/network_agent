package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
)

func readInput() string {
	reader := bufio.NewReader(os.Stdin)

	text, err := reader.ReadString('\n')
	if err != nil {
		log.Panicf("failed to read from Terminal: %v", err)
	}
	return strings.Trim(text, "\r\n ")
}

func selectProvider() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("You can choose between this providers")

	for _, i := range availableProviders {
		fmt.Printf("[1] %s\n", i)
	}

	for {
		text, _ := reader.ReadString('\n')
		var input, err = strconv.Atoi(strings.Trim(text, "\r\n"))
		if err != nil {
			log.Panicf("entrada inválida ao escolher o provider: %v", err)
		}

		if input <= 0 || input > len(availableProviders) {
			err = exception.InvalidInput
		}
		if err == nil {
			return availableProviders[input-1]
		}
		fmt.Println("Invalid input. Try again.")
	}
}

func getUrl() string {
	fmt.Println("Write URL name")
	input := readInput()

	return input
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

func getModelId() string {
	fmt.Println("Write model Id")
	input := readInput()

	return input
}

func setContextSize() int64 {
	fmt.Println("Set model context size.")
	fmt.Println("0 for default")

	for {
		input, err := strconv.ParseInt(readInput(), 10, 64)
		if err != nil {
			continue
		}
		return input
	}
}
