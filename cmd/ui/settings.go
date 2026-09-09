package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
)

var configMux sync.Mutex

// readConfig lê a configuração tolerando a ausência do arquivo: no primeiro
// uso a UI deve abrir vazia para o usuário cadastrar, não exibir erro.
// (config.Load() não serve aqui: sem Creator registrado ele devolve o erro,
// e registrar um wizard de stdin travaria a GUI.)
func readConfig() (types.Config, error) {
	cfg, err := config.Read()
	if errors.Is(err, config.ErrNotFound) {
		return types.Config{Providers: []types.Provider{}}, nil
	}
	return cfg, err
}

// persist grava e reconecta o agente, para que a alteração valha na hora.
func (a *App) persist(cfg types.Config) (types.Config, error) {
	config.Normalize(&cfg)
	if err := config.Save(cfg); err != nil {
		return cfg, err
	}
	a.reconnect(cfg)
	return cfg, nil
}

// ---------- API exposta ao frontend ----------

// GetConfig devolve a configuração atual (provedores e modelos).
func (a *App) GetConfig() (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	config.Normalize(&cfg)
	return cfg, nil
}

// SelectProvider define qual provedor está em uso (índice Selected).
func (a *App) SelectProvider(index int) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if index < 0 || index >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", index)
	}
	cfg.Selected = int8(index)
	return a.persist(cfg)
}

// ---------- CRUD de provedores ----------

func (a *App) CreateProvider(p types.Provider) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	if p.Name == "" {
		return types.Config{}, fmt.Errorf("o nome do provedor é obrigatório")
	}
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if p.Models == nil {
		p.Models = []types.Model{}
	}
	cfg.AddProvider(p)
	return a.persist(cfg)
}

func (a *App) UpdateProvider(index int, p types.Provider) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	if p.Name == "" {
		return types.Config{}, fmt.Errorf("o nome do provedor é obrigatório")
	}
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if index < 0 || index >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", index)
	}
	// Preserva os modelos: a edição do provedor não mexe na lista deles.
	p.Models = cfg.Providers[index].Models
	p.DefaultModel = cfg.Providers[index].DefaultModel
	cfg.Providers[index] = p
	return a.persist(cfg)
}

func (a *App) DeleteProvider(index int) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if index < 0 || index >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", index)
	}
	// Remoção preservando a ordem (types.RemoveProvider troca com o último).
	cfg.Providers = append(cfg.Providers[:index], cfg.Providers[index+1:]...)
	return a.persist(cfg)
}

// ---------- CRUD de modelos (dentro de um provedor) ----------

func (a *App) CreateModel(providerIndex int, m types.Model) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	if m.Id == "" {
		return types.Config{}, fmt.Errorf("o id do modelo é obrigatório")
	}
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if providerIndex < 0 || providerIndex >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", providerIndex)
	}
	cfg.Providers[providerIndex].AddModel(m)
	return a.persist(cfg)
}

func (a *App) UpdateModel(providerIndex, modelIndex int, m types.Model) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	if m.Id == "" {
		return types.Config{}, fmt.Errorf("o id do modelo é obrigatório")
	}
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if providerIndex < 0 || providerIndex >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", providerIndex)
	}
	models := cfg.Providers[providerIndex].Models
	if modelIndex < 0 || modelIndex >= len(models) {
		return cfg, fmt.Errorf("modelo inexistente: %d", modelIndex)
	}
	models[modelIndex] = m
	return a.persist(cfg)
}

func (a *App) DeleteModel(providerIndex, modelIndex int) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if providerIndex < 0 || providerIndex >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", providerIndex)
	}
	p := &cfg.Providers[providerIndex]
	if modelIndex < 0 || modelIndex >= len(p.Models) {
		return cfg, fmt.Errorf("modelo inexistente: %d", modelIndex)
	}
	p.Models = append(p.Models[:modelIndex], p.Models[modelIndex+1:]...)
	return a.persist(cfg)
}

// SetDefaultModel define o modelo padrão do provedor (índice DefaultModel).
func (a *App) SetDefaultModel(providerIndex, modelIndex int) (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()

	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}
	if providerIndex < 0 || providerIndex >= len(cfg.Providers) {
		return cfg, fmt.Errorf("provedor inexistente: %d", providerIndex)
	}
	p := &cfg.Providers[providerIndex]
	if modelIndex < 0 || modelIndex >= len(p.Models) {
		return cfg, fmt.Errorf("modelo inexistente: %d", modelIndex)
	}
	p.DefaultModel = int8(modelIndex)
	return a.persist(cfg)
}
