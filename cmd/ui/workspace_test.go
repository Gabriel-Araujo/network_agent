package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
)

// comConnect troca a composição que monta o agente por uma falsa e devolve o
// canal onde o Workspace recebido aparece. O Connect de verdade carrega o
// modelo no provedor (dezenas de segundos, ticket 13); aqui só interessa o
// argumento.
func comConnect(t *testing.T) <-chan string {
	t.Helper()

	recebido := make(chan string, 1)
	anterior := connect
	connect = func(_ types.Config, workspace string) (*agent.Agent, error) {
		recebido <- workspace
		return &agent.Agent{}, nil
	}
	t.Cleanup(func() { connect = anterior })
	return recebido
}

func configComModelo(workspace string) types.Config {
	return types.Config{
		Providers: []types.Provider{{
			Name:   "local",
			Models: []types.Model{{Id: "gemma-4-e4b"}},
		}},
		Workspace: workspace,
	}
}

// O Workspace da configuração chega ao Agent. Era o literal "" até este
// ticket, e um literal de volta aqui é o modelo recusando toda escrita do
// usuário — ou, pior, gravando contra outra raiz — em silêncio.
func TestReconnectPassaOWorkspaceDaConfiguracao(t *testing.T) {
	recebido := comConnect(t)
	pasta := t.TempDir()

	a := &App{}
	a.reconnect(configComModelo(pasta))

	select {
	case got := <-recebido:
		if got != pasta {
			t.Fatalf("Workspace entregue ao Connect = %q, want %q", got, pasta)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("o Connect não foi chamado")
	}
}

// A outra direção: sem pasta escolhida o que viaja é vazio, que é o estado
// que o rootFor recusa com texto próprio (ticket 05, §3) — não um caminho
// inventado para o lugar do vazio.
func TestReconnectSemWorkspaceEntregaVazio(t *testing.T) {
	recebido := comConnect(t)

	a := &App{}
	a.reconnect(configComModelo(""))

	select {
	case got := <-recebido:
		if got != "" {
			t.Fatalf("Workspace entregue ao Connect = %q, want vazio", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("o Connect não foi chamado")
	}
}

// A pasta que sumiu é recusada e a configuração fica como estava; a que existe
// vira o Workspace e sobe nos recentes.
func TestUseRecentRecusaPastaSumida(t *testing.T) {
	viva, sumida := t.TempDir(), filepath.Join(t.TempDir(), "sumiu")
	cfg := types.Config{Workspace: viva, RecentWorkspaces: []string{viva, sumida}}

	if err := useRecent(&cfg, sumida); err == nil {
		t.Fatal("a pasta que não existe foi aceita")
	}
	if cfg.Workspace != viva || cfg.RecentWorkspaces[0] != viva {
		t.Fatalf("a recusa mudou a configuração: %+v", cfg)
	}

	outra := t.TempDir()
	if err := useRecent(&cfg, outra); err != nil {
		t.Fatal(err)
	}
	if cfg.Workspace != outra || cfg.RecentWorkspaces[0] != outra || cfg.RecentWorkspaces[1] != viva {
		t.Fatalf("a escolha não subiu: %+v", cfg)
	}
}
