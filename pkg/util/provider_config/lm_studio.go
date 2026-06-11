package provider_config

type ProviderConfig struct {
	ProviderName string
	ApiKey       string
	Url          string
	ModelName    string
}

var LMStudioConfig = func() ProviderConfig {
	return ProviderConfig{
		"LM Studio",
		GetApiKey(),
		GetUrl(),
		GetModelName(),
	}
}
