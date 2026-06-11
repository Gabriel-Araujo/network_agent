package provider_config

import "fmt"

func GetApiKey() string {
	var input string

	println("Set your apiKey")
	_, err := fmt.Scanf("%s", &input)
	if err != nil {
		panic("Panic at reading API key")
	}

	return input
}

func GetModelName() string {
	var input string

	println("Write model name")

	_, err := fmt.Scanf("%s", &input)
	if err != nil {
		panic("Panic at reading API key")
	}

	return input
}

func GetUrl() string {
	var input string

	println("Write URL name")

	_, err := fmt.Scanf("%s", &input)
	if err != nil {
		panic("Panic at reading provider URL")
	}

	return input
}
