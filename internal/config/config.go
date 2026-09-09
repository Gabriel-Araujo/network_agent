package config

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
)

var log = logger.Named("CONFIG")

// ErrNotFound indica que o arquivo de configuração ainda não existe.
// Cada front-end decide o que fazer: o CLI abre o wizard no terminal, a UI
// abre a tela de Configurações.
var ErrNotFound = errors.New("config: arquivo de configuração não encontrado")

// ErrInvalid indica que o arquivo existe mas não pôde ser interpretado.
var ErrInvalid = errors.New("config: arquivo de configuração inválido")

// Creator produz uma configuração quando nenhuma existe. É o ponto de
// extensão de cada front-end: o CLI registra um wizard de terminal; a UI
// não registra nada e trata ErrNotFound abrindo a própria tela.
type Creator func() (types.Config, error)

var creator Creator

// SetCreator registra o criador usado por Load quando não há configuração.
// Chame no início do main de cada cmd/.
func SetCreator(c Creator) {
	creator = c
}

// Read carrega a configuração do disco sem efeitos colaterais.
// Devolve ErrNotFound ou ErrInvalid para o chamador decidir o que fazer.
func Read() (types.Config, error) {
	var cfg types.Config

	raw, err := os.ReadFile(fileLocation)
	if os.IsNotExist(err) {
		return cfg, ErrNotFound
	}
	if err != nil {
		return cfg, fmt.Errorf("config: falha ao ler %s: %w", fileLocation, err)
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("%w: %s", ErrInvalid, err)
	}

	Normalize(&cfg)
	log.Infof("Loaded config from: %s\n", fileLocation)
	return cfg, nil
}

// Save grava a configuração de forma atômica (arquivo temporário + rename),
// para não corromper o arquivo caso o processo morra no meio da escrita.
func Save(cfg types.Config) error {
	Normalize(&cfg)

	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("config: falha ao serializar: %w", err)
	}

	tmp := fileLocation + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("config: falha ao escrever %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, fileLocation); err != nil {
		return fmt.Errorf("config: falha ao gravar %s: %w", fileLocation, err)
	}
	return nil
}

// Load lê a configuração e, se não houver nenhuma válida, delega ao Creator
// registrado pelo front-end e grava o resultado.
//
// Sem Creator registrado, devolve ErrNotFound/ErrInvalid em vez de bloquear
// — essencial para GUIs, onde ler do stdin travaria a aplicação.
func Load() (types.Config, error) {
	cfg, err := Read()
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrInvalid) {
		return cfg, err
	}

	if creator == nil {
		return cfg, err
	}

	if errors.Is(err, ErrNotFound) {
		log.Warn("No config file detected. Starting config creator.")
	} else {
		log.Warn("Failed to load existing config. Starting config creator.")
	}

	created, cErr := creator()
	if cErr != nil {
		return cfg, fmt.Errorf("config: falha ao criar configuração: %w", cErr)
	}
	if sErr := Save(created); sErr != nil {
		return created, sErr
	}
	return created, nil
}

// Normalize mantém os índices Selected/DefaultModel dentro dos limites e
// garante slices não-nulos. Chamado em toda leitura e gravação, já que os
// front-ends podem remover o provedor ou modelo apontado por um índice.
func Normalize(cfg *types.Config) {
	if cfg.Providers == nil {
		cfg.Providers = []types.Provider{}
	}
	if len(cfg.Providers) == 0 {
		cfg.Selected = 0
		return
	}
	if int(cfg.Selected) >= len(cfg.Providers) || cfg.Selected < 0 {
		cfg.Selected = 0
	}
	for i := range cfg.Providers {
		p := &cfg.Providers[i]
		if p.Models == nil {
			p.Models = []types.Model{}
		}
		if len(p.Models) == 0 {
			p.DefaultModel = 0
			continue
		}
		if int(p.DefaultModel) >= len(p.Models) || p.DefaultModel < 0 {
			p.DefaultModel = 0
		}
	}
}
