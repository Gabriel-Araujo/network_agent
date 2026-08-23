package loaders

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
)

func getProvider() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("You can choose between this providers")

	for _, i := range providers {
		fmt.Printf("[1] %s\n", i)
	}

	for {
		text, _ := reader.ReadString('\n')
		var input, err = strconv.Atoi(strings.Trim(text, "\r\n"))
		if err != nil {
			log.Panicf("entrada inválida ao escolher o provider: %v", err)
		}

		if input <= 0 || input > len(providers) {
			err = exception.InvalidInput
		}
		if err == nil {
			return providers[input-1]
		}
		fmt.Println("Invalid input. Try again.")
	}
}

func CreateConfig() ([]ProviderConfig, error) {
	name := getProvider()
	return saveConfig([]ProviderConfig{builder[name]()})
}
