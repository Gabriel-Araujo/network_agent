package types

import "errors"

type Config struct {
	Selected  int8       `json:"selected"`
	Providers []Provider `json:"providers"`
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
