package types

import "errors"

type Config struct {
	Selected  int8       `json:"selected"`
	Providers []Provider `json:"providers"`

	// Workspace é a pasta que o usuário escolheu no app de desktop: a única
	// raiz de escrita das file tools (ADR 0003). Vazio é "nenhuma escolhida",
	// estado que o rootFor recusa com texto próprio — não um valor a
	// consertar, e por isso não há nada sobre ele no config.Normalize. O
	// cmd/cli desserializa e ignora: a raiz dele é o diretório onde o usuário
	// está parado.
	Workspace string `json:"workspace,omitempty"`
}

func (c *Config) AddProvider(provider Provider) {
	c.Providers = append(c.Providers, provider)
}

func (c *Config) RemoveProvider(name string) (Provider, error) {
	for i, p := range c.Providers {
		if p.Name == name {
			c.Providers[i] = c.Providers[len(c.Providers)-1]
			return p, nil
		}
	}
	return Provider{}, errors.New("provider not found")
}
