package config

import (
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/types"
)

// O Workspace sobrevive ao fechar o app: ele é um campo do config.json, e a
// gravação/leitura é o único mecanismo de persistência da escolha (ticket 05,
// §4). Sem isto, escolher a pasta valeria só até o processo morrer.
func TestWorkspaceSobreviveARodadaDeDisco(t *testing.T) {
	tempHome(t)

	pasta := t.TempDir()
	if err := Save(types.Config{Workspace: pasta}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	lida, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if lida.Workspace != pasta {
		t.Fatalf("Workspace lido = %q, want %q", lida.Workspace, pasta)
	}
}

// E o Normalize não tem nada a dizer sobre ele: vazio é "nenhuma escolhida",
// um estado que as file tools tratam, não um valor a consertar. Um default
// aqui — o home, o CWD — poria o agente escrevendo numa pasta que o usuário
// não escolheu.
func TestNormalizeNaoMexeNoWorkspace(t *testing.T) {
	for _, valor := range []string{"", `D:\lab`} {
		cfg := types.Config{Workspace: valor}
		Normalize(&cfg)
		if cfg.Workspace != valor {
			t.Errorf("Normalize mudou o Workspace de %q para %q", valor, cfg.Workspace)
		}
	}
}
