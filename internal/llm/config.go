package llm

import (
	"bufio"
	"os"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/paths"
)

type Config struct {
	ModelName    string
	Url          string
	ApiKey       string
	ProviderName string
}

// envConfig lê o arquivo .env na raiz do repositório e monta uma Config com os
// valores MODEL_NAME, URL, API_KEY e PROVIDER_NAME.
func envConfig() Config {
	envPath, err := paths.RepoFile(".env")
	if err != nil {
		log.Panicf("falha ao resolver o caminho do .env: %v", err)
	}

	f, err := os.Open(envPath)
	if err != nil {
		log.Panicf("falha ao abrir o .env: %v", err)
	}
	defer f.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}

	return Config{
		ModelName:    values["MODEL_NAME"],
		Url:          values["URL"],
		ApiKey:       values["API_KEY"],
		ProviderName: values["PROVIDER_NAME"],
	}
}
