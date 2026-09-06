package llm

import "github.com/Gabriel-Araujo/network_agent/internal/types"

var embedModel = types.Model{
	Id:   "fastiraz/text-embedding-qwen3-embedding-8b",
	Name: "Qwen3 Embedding *B",
}

var testModel = types.Model{
	Id:   "google/gemma-4-12b-qat",
	Name: "Gemma 4 12B QAT",
}

var localProvider = types.Provider{
	Name:         "LM Studio",
	Url:          "http://localhost:1234/v1",
	ApiKey:       "none",
	DefaultModel: 0,
	Models:       []types.Model{testModel, embedModel},
}

func getEmbedConfig() types.Config {
	provider := localProvider
	provider.DefaultModel = 1
	return types.Config{Providers: []types.Provider{provider}}
}

func getTestConfig() types.Config {
	provider := localProvider
	provider.DefaultModel = 0
	return types.Config{Providers: []types.Provider{provider}}
}
