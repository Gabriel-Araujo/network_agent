package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// tempHome aponta o diretório de dados do app para um t.TempDir() e descarta a
// memoização da semeadura, para cada teste medir uma "primeira abertura" sua
// sem tocar no ~/.network_agent da máquina. A costura é o SetDir exportado —
// os dois espelhos que gravam no home moram fora deste pacote e precisam da
// mesma troca.
func tempHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Cleanup(SetDir(home))

	return home
}

// reopen descarta a memoização sem mexer no home: é o que um processo novo vê
// na primeira leitura de material, com o espelho já no disco.
func reopen() {
	materialRoot = sync.OnceValues(seedMaterial)
}

// TestMaterialFileSeedsTheMirror é a primeira direção: a primeira chamada
// projeta o material embutido no home, nos caminhos exatos que os três
// leitores juntam — resources/agents/skills (LoadSkills),
// resources/agents (ForSubagent) e resources/agents/teacher/tech-packs (a
// file tool do Teacher).
func TestMaterialFileSeedsTheMirror(t *testing.T) {
	home := tempHome(t)

	root, err := MaterialFile("resources/agents")
	if err != nil {
		t.Fatalf("MaterialFile: %v", err)
	}
	if want := filepath.Join(home, "resources", "agents"); root != want {
		t.Errorf("MaterialFile(\"resources/agents\") = %q, want %q", root, want)
	}

	// Um por leitor, mais o references/ que entra sem leitor.
	for _, rel := range []string{
		"resources/agents/skills/mermaid/SKILL.md",
		"resources/agents/teacher/AGENT.md",
		"resources/agents/teacher/tech-packs",
		"resources/references/examples/BGP.md",
	} {
		path := filepath.Join(home, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("espelho sem %s: %v", rel, err)
			continue
		}
		if !info.IsDir() && info.Size() == 0 {
			t.Errorf("espelho com %s vazio", rel)
		}
	}

	// O corpus do ingester não viaja: 9,7M no home para servir leitor nenhum.
	if _, err := os.Stat(filepath.Join(home, "resources", "frrouting")); err == nil {
		t.Error("espelho carregou resources/frrouting, que nenhum turno lê")
	}
}

// TestMaterialFileOverwritesDivergentFile é a segunda direção, sem a qual
// "passa" não prova que a projeção projeta: um arquivo do espelho editado por
// fora volta ao conteúdo embutido na abertura seguinte. Edição do usuário no
// espelho não é preservada, por decisão (ADR 0004).
func TestMaterialFileOverwritesDivergentFile(t *testing.T) {
	home := tempHome(t)

	if _, err := MaterialFile("resources/agents"); err != nil {
		t.Fatalf("primeira semeadura: %v", err)
	}

	mirrored := filepath.Join(home, "resources", "agents", "teacher", "AGENT.md")
	embedded, err := os.ReadFile(mirrored)
	if err != nil {
		t.Fatalf("lendo o arquivo semeado: %v", err)
	}
	if err := os.WriteFile(mirrored, []byte("editado pelo usuário"), 0o600); err != nil {
		t.Fatalf("divergindo o arquivo: %v", err)
	}

	reopen()
	if _, err := MaterialFile("resources/agents"); err != nil {
		t.Fatalf("segunda abertura: %v", err)
	}

	got, err := os.ReadFile(mirrored)
	if err != nil {
		t.Fatalf("lendo o arquivo reprojetado: %v", err)
	}
	if string(got) != string(embedded) {
		t.Errorf("AGENT.md do espelho = %q..., want o conteúdo embutido de volta", string(got[:min(len(got), 40)]))
	}
}

// TestDirIsTheAppDataDirectory fixa o helper que a decisão nomeia: Dir() é o
// diretório que contém o config.json, e é a raiz contra a qual o espelho e
// todo artefato do app no home se derivam — sem um terceiro literal.
func TestDirIsTheAppDataDirectory(t *testing.T) {
	if got, want := Dir(), filepath.Dir(Path()); got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
	if base := filepath.Base(Dir()); base != configDirName && base != "network_agent" {
		t.Errorf("Dir() = %q, want terminar em %q ou %q", Dir(), configDirName, "network_agent")
	}
}
