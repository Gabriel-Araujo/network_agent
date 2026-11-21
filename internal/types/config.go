package types

import (
	"errors"
	"path/filepath"
	"strings"
)

// MaxRecentWorkspaces é quantas pastas o chip do Workspace lembra.
const MaxRecentWorkspaces = 5

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

	// RecentWorkspaces são as pastas que já foram o Workspace, a de uso mais
	// recente primeiro, no máximo MaxRecentWorkspaces. Não é raiz de nada: só
	// o Workspace é. O UseWorkspace é quem a mantém.
	RecentWorkspaces []string `json:"recentWorkspaces,omitempty"`
}

// UseWorkspace faz de dir o Workspace e o põe no topo dos recentes, sem
// duplicata e cortando em MaxRecentWorkspaces. O caminho é comparado sem
// diferenciar maiúsculas, porque o app roda no Windows.
// ponytail: o mesmo vale no Linux, onde duas pastas só na caixa são duas; sobe se o app sair do Windows.
func (c *Config) UseWorkspace(dir string) {
	c.Workspace = dir
	if dir == "" {
		return
	}
	same := func(a string) bool {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(dir))
	}
	recents := []string{dir}
	for _, r := range c.RecentWorkspaces {
		if !same(r) {
			recents = append(recents, r)
		}
	}
	c.RecentWorkspaces = recents[:min(len(recents), MaxRecentWorkspaces)]
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
