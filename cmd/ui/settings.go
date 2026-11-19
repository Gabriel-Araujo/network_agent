package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/wailsapp/wails/v3/pkg/application"
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

// PickWorkspace abre o diálogo nativo de pasta e grava a escolha como o
// Workspace do app: a única raiz de escrita das file tools. A pasta é do app e
// não da Conversa (ticket 05, §4), então é um valor só.
//
// Cancelar devolve a configuração como está: cancelar o diálogo não apaga a
// pasta que já estava escolhida.
func (a *App) PickWorkspace() (types.Config, error) {
	configMux.Lock()
	defer configMux.Unlock()
	cfg, err := readConfig()
	if err != nil {
		return cfg, err
	}

	dialog := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("Escolha a pasta de trabalho do agente")
	// Diretório inicial inexistente é erro do próprio diálogo — e "o Workspace
	// sumiu" é justamente um dos casos que trazem o usuário até aqui.
	if info, err := os.Stat(cfg.Workspace); err == nil && info.IsDir() {
		dialog.SetDirectory(cfg.Workspace)
	}

	dir, err := dialog.PromptForSingleSelection()
	// No Windows o v3 devolve o cancelar como erro (cfd.ErrorCancelled, de um
	// pacote internal, então só a mensagem dá para comparar). Cancelar não é
	// falha: é o "" do v2.
	if err != nil && err.Error() == "cancelled by user" {
		dir, err = "", nil
	}
	if err != nil {
		log.Error("falha ao abrir o diálogo de pasta", "err", err)
		return cfg, fmt.Errorf("falha ao abrir o diálogo de pasta: %w", err)
	}
	if dir == "" {
		return cfg, nil
	}

	log.Info("Workspace escolhido", "path", dir)
	cfg.Workspace = dir
	return a.persist(cfg)
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
