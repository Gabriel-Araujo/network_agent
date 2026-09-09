package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Model é um modelo pertencente a um provedor.
type Model struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	MaxContext  int64   `json:"maxContext"`
	MaxTokens   int64   `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"topP"`
	Default     bool    `json:"default"`
}

// Provider é um provedor de LLM (LM Studio, OpenAI, Ollama, ...).
type Provider struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	BaseURL string  `json:"baseURL"`
	APIKey  string  `json:"apiKey"`
	Enabled bool    `json:"enabled"`
	Models  []Model `json:"models"`
}

// Settings é o documento persistido em disco.
type Settings struct {
	Providers []Provider `json:"providers"`
}

var settingsMux sync.Mutex

// settingsPath devolve o caminho do settings.json no diretório de config
// do usuário (ex.: %APPDATA%/network_agent/settings.json).
func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "network_agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// idSeq garante unicidade dos IDs: time.Now().UnixNano() sozinho colide no
// Windows, cuja resolução de relógio é de ~100ns — dois registros criados no
// mesmo instante recebiam IDs iguais, e o update acabava editando o errado.
var idSeq atomic.Uint64

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), idSeq.Add(1))
}

func readSettings() (Settings, error) {
	var s Settings
	path, err := settingsPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Primeiro uso: semeia com o provedor local usado hoje pelo agente e
		// GRAVA imediatamente. Sem gravar, cada readSettings geraria IDs novos
		// para o seed, e um update chegaria com um ID que não existe mais.
		seed := Settings{Providers: []Provider{{
			ID:      newID("prov"),
			Name:    "LM Studio (local)",
			BaseURL: "http://localhost:1234/v1",
			Enabled: true,
			Models: []Model{{
				ID:          newID("mod"),
				Name:        "google/gemma-4-12b-qat",
				MaxContext:  128000,
				MaxTokens:   4096,
				Temperature: 0.7,
				TopP:        1,
				Default:     true,
			}},
		}}}
		if err := writeSettings(seed); err != nil {
			return seed, err
		}
		return seed, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("settings.json inválido: %w", err)
	}
	// Repara IDs duplicados herdados de versões anteriores: sem isso o update
	// editaria sempre o primeiro registro com aquele ID.
	seen := map[string]bool{}
	repaired := false
	for i := range s.Providers {
		if seen[s.Providers[i].ID] || s.Providers[i].ID == "" {
			s.Providers[i].ID = newID("prov")
			repaired = true
		}
		seen[s.Providers[i].ID] = true

		mseen := map[string]bool{}
		for j := range s.Providers[i].Models {
			id := s.Providers[i].Models[j].ID
			if mseen[id] || id == "" {
				s.Providers[i].Models[j].ID = newID("mod")
				repaired = true
			}
			mseen[s.Providers[i].Models[j].ID] = true
		}
	}
	if repaired {
		if err := writeSettings(s); err != nil {
			return s, err
		}
	}
	return s, nil
}

func writeSettings(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	// Escrita atômica: grava em arquivo temporário e renomeia, para não
	// corromper o settings.json se o app morrer no meio da gravação.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---------- CRUD de provedores ----------

// GetSettings devolve todos os provedores e seus modelos.
func (a *App) GetSettings() (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()
	return readSettings()
}

// CreateProvider adiciona um provedor e devolve o estado atualizado.
func (a *App) CreateProvider(p Provider) (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()

	if p.Name == "" {
		return Settings{}, fmt.Errorf("o nome do provedor é obrigatório")
	}
	s, err := readSettings()
	if err != nil {
		return s, err
	}
	p.ID = newID("prov")
	if p.Models == nil {
		p.Models = []Model{}
	}
	s.Providers = append(s.Providers, p)
	return s, writeSettings(s)
}

// UpdateProvider altera os campos de um provedor (preservando seus modelos).
func (a *App) UpdateProvider(p Provider) (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()

	if p.Name == "" {
		return Settings{}, fmt.Errorf("o nome do provedor é obrigatório")
	}
	s, err := readSettings()
	if err != nil {
		return s, err
	}
	for i := range s.Providers {
		if s.Providers[i].ID == p.ID {
			p.Models = s.Providers[i].Models
			s.Providers[i] = p
			return s, writeSettings(s)
		}
	}
	return s, fmt.Errorf("provedor não encontrado: %s", p.ID)
}

// DeleteProvider remove um provedor e todos os seus modelos.
func (a *App) DeleteProvider(id string) (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()

	s, err := readSettings()
	if err != nil {
		return s, err
	}
	for i := range s.Providers {
		if s.Providers[i].ID == id {
			s.Providers = append(s.Providers[:i], s.Providers[i+1:]...)
			return s, writeSettings(s)
		}
	}
	return s, fmt.Errorf("provedor não encontrado: %s", id)
}

// ---------- CRUD de modelos (aninhado no provedor) ----------

func findProvider(s *Settings, providerID string) (*Provider, error) {
	for i := range s.Providers {
		if s.Providers[i].ID == providerID {
			return &s.Providers[i], nil
		}
	}
	return nil, fmt.Errorf("provedor não encontrado: %s", providerID)
}

// clearOtherDefaults garante que só um modelo do provedor seja o padrão.
func clearOtherDefaults(p *Provider, keepID string) {
	for i := range p.Models {
		if p.Models[i].ID != keepID {
			p.Models[i].Default = false
		}
	}
}

// CreateModel adiciona um modelo dentro de um provedor.
func (a *App) CreateModel(providerID string, m Model) (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()

	if m.Name == "" {
		return Settings{}, fmt.Errorf("o nome do modelo é obrigatório")
	}
	s, err := readSettings()
	if err != nil {
		return s, err
	}
	p, err := findProvider(&s, providerID)
	if err != nil {
		return s, err
	}
	m.ID = newID("mod")
	p.Models = append(p.Models, m)
	if m.Default {
		clearOtherDefaults(p, m.ID)
	}
	return s, writeSettings(s)
}

// UpdateModel altera um modelo dentro de um provedor.
func (a *App) UpdateModel(providerID string, m Model) (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()

	if m.Name == "" {
		return Settings{}, fmt.Errorf("o nome do modelo é obrigatório")
	}
	s, err := readSettings()
	if err != nil {
		return s, err
	}
	p, err := findProvider(&s, providerID)
	if err != nil {
		return s, err
	}
	for i := range p.Models {
		if p.Models[i].ID == m.ID {
			p.Models[i] = m
			if m.Default {
				clearOtherDefaults(p, m.ID)
			}
			return s, writeSettings(s)
		}
	}
	return s, fmt.Errorf("modelo não encontrado: %s", m.ID)
}

// DeleteModel remove um modelo de um provedor.
func (a *App) DeleteModel(providerID string, modelID string) (Settings, error) {
	settingsMux.Lock()
	defer settingsMux.Unlock()

	s, err := readSettings()
	if err != nil {
		return s, err
	}
	p, err := findProvider(&s, providerID)
	if err != nil {
		return s, err
	}
	for i := range p.Models {
		if p.Models[i].ID == modelID {
			p.Models = append(p.Models[:i], p.Models[i+1:]...)
			return s, writeSettings(s)
		}
	}
	return s, fmt.Errorf("modelo não encontrado: %s", modelID)
}
