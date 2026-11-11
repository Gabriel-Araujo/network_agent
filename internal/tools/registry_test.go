package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

// TestRootForPicksRootByPrefix mede a regra das duas raízes nas duas direções,
// que é o que o ticket 10 pediu: sem a direção contrária, "passa" não prova
// nada aqui. Um t.TempDir() faz de raiz só-leitura e outro de Workspace, e cada
// caso exige a raiz certa **ou** a recusa certa — um prefixo que caísse no
// Workspace, ou uma escrita que caísse no espelho, falha aqui.
func TestRootForPicksRootByPrefix(t *testing.T) {
	appRoot := t.TempDir()
	workspace := t.TempDir()
	gone := filepath.Join(t.TempDir(), "drive-que-saiu")

	tests := []struct {
		name        string
		tool        string
		filePath    string
		workspace   string
		wantRoot    string
		wantRefusal string
	}{{
		name:     "leitura com o prefixo do espelho resolve no material do agente",
		tool:     filetools.READ_TOOL_NAME,
		filePath: "resources/agents/teacher/tech-packs/bgp.md",
		// Sem Workspace de propósito: é o caso que o ticket 05 (§3) decidiu —
		// Teacher e Illustrator trabalham inteiros sem pasta escolhida.
		workspace: "",
		wantRoot:  appRoot,
	}, {
		name:      "busca com o prefixo do espelho também resolve nele",
		tool:      filetools.SEARCH_TOOL_NAME,
		filePath:  "resources/agents",
		workspace: workspace,
		wantRoot:  appRoot,
	}, {
		name:      "separador do Windows no prefixo resolve igual",
		tool:      filetools.READ_TOOL_NAME,
		filePath:  filepath.Join("resources", "agents", "teacher", "AGENT.md"),
		workspace: workspace,
		wantRoot:  appRoot,
	}, {
		name:        "escrita no prefixo do espelho é recusada",
		tool:        filetools.WRITE_TOOL_NAME,
		filePath:    "resources/agents/teacher/AGENT.md",
		workspace:   workspace,
		wantRefusal: filetools.READ_ONLY_ROOT_NOTICE,
	}, {
		name:        "edição no prefixo do espelho é recusada: ela lê e escreve",
		tool:        filetools.EDIT_TOOL_NAME,
		filePath:    "resources/agents/teacher/AGENT.md",
		workspace:   workspace,
		wantRefusal: filetools.READ_ONLY_ROOT_NOTICE,
	}, {
		name:      "caminho sem prefixo resolve no Workspace",
		tool:      filetools.WRITE_TOOL_NAME,
		filePath:  "frr.conf",
		workspace: workspace,
		wantRoot:  workspace,
	}, {
		name:        "sem Workspace escolhido, o caminho do usuário é recusado",
		tool:        filetools.WRITE_TOOL_NAME,
		filePath:    "frr.conf",
		workspace:   "",
		wantRefusal: filetools.NO_WORKSPACE_NOTICE,
	}, {
		name:        "leitura sem Workspace escolhido é recusada pela mesma razão",
		tool:        filetools.READ_TOOL_NAME,
		filePath:    "frr.conf",
		workspace:   "",
		wantRefusal: filetools.NO_WORKSPACE_NOTICE,
	}, {
		// O search devolve absolutos, e a descrição manda reusá-los: um absoluto
		// do espelho tem de voltar ao espelho, não ao Workspace.
		name:      "absoluto dentro do espelho resolve nele, sem prefixo nenhum",
		tool:      filetools.READ_TOOL_NAME,
		filePath:  filepath.Join(appRoot, "resources", "agents", "teacher", "AGENT.md"),
		workspace: workspace,
		wantRoot:  appRoot,
	}, {
		name:        "absoluto dentro do espelho também recusa escrita",
		tool:        filetools.WRITE_TOOL_NAME,
		filePath:    filepath.Join(appRoot, "resources", "agents", "teacher", "AGENT.md"),
		workspace:   workspace,
		wantRefusal: filetools.READ_ONLY_ROOT_NOTICE,
	}, {
		name:      "absoluto dentro do Workspace resolve nele",
		tool:      filetools.WRITE_TOOL_NAME,
		filePath:  filepath.Join(workspace, "frr.conf"),
		workspace: workspace,
		wantRoot:  workspace,
	}, {
		name:        "Workspace que sumiu tem recusa própria",
		tool:        filetools.WRITE_TOOL_NAME,
		filePath:    "frr.conf",
		workspace:   gone,
		wantRefusal: filetools.WORKSPACE_GONE_NOTICE,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, refusal := rootFor(appRoot, tc.tool, tc.filePath, tc.workspace)

			if tc.wantRefusal != "" {
				if !strings.HasPrefix(refusal, tc.wantRefusal) {
					t.Errorf("recusa = %q, want começando com %q", refusal, tc.wantRefusal)
				}
				if root != "" {
					t.Errorf("raiz = %q junto de uma recusa, want vazia", root)
				}
				return
			}

			if refusal != "" {
				t.Fatalf("recusa inesperada: %q", refusal)
			}
			if root != tc.wantRoot {
				t.Errorf("raiz = %q, want %q", root, tc.wantRoot)
			}
		})
	}
}

// TestRootForRefusesWriteIntoTheAppDirectory fecha a regra 2 onde o prefixo não
// alcança: com o Workspace **acima** do diretório do app — o home escolhido como
// pasta —, "config.json" não tem prefixo nenhum e ainda assim cai dentro do
// material do agente. A escrita ali é a falha que aviso nenhum recupera, e a
// leitura continua liberada, porque a pasta é do usuário.
func TestRootForRefusesWriteIntoTheAppDirectory(t *testing.T) {
	home := t.TempDir()
	appRoot := filepath.Join(home, ".network_agent")
	if err := os.MkdirAll(appRoot, 0o700); err != nil {
		t.Fatal(err)
	}

	const inApp = ".network_agent/config.json"

	for _, tool := range []string{filetools.WRITE_TOOL_NAME, filetools.EDIT_TOOL_NAME} {
		root, refusal := rootFor(appRoot, tool, inApp, home)
		if !strings.HasPrefix(refusal, filetools.READ_ONLY_ROOT_NOTICE) {
			t.Errorf("%s em %q: recusa = %q, want começando com %q", tool, inApp, refusal, filetools.READ_ONLY_ROOT_NOTICE)
		}
		if root != "" {
			t.Errorf("%s em %q: raiz = %q, want vazia", tool, inApp, root)
		}
	}

	// O Workspace escolhido sendo o próprio diretório do app, nem um nome solto
	// escapa: toda escrita resolve dentro da raiz só-leitura.
	if _, refusal := rootFor(appRoot, filetools.WRITE_TOOL_NAME, "frr.conf", appRoot); !strings.HasPrefix(refusal, filetools.READ_ONLY_ROOT_NOTICE) {
		t.Errorf("Workspace = raiz só-leitura: recusa = %q, want começando com %q", refusal, filetools.READ_ONLY_ROOT_NOTICE)
	}

	// A direção contrária: ler é do usuário, e a pasta é dele.
	root, refusal := rootFor(appRoot, filetools.READ_TOOL_NAME, inApp, home)
	if refusal != "" {
		t.Errorf("leitura em %q recusada: %q", inApp, refusal)
	}
	if root != home {
		t.Errorf("leitura em %q: raiz = %q, want %q", inApp, root, home)
	}
}

// toolCall monta a chamada de tool que o modelo manda, para exercitar o
// CallFunction pela porta de entrada de verdade.
func toolCall(t *testing.T, name string, args map[string]string) responses.ResponseFunctionToolCall {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return responses.ResponseFunctionToolCall{Name: name, Arguments: string(raw), CallID: "call-1"}
}

func toolOutput(t *testing.T, item responses.ResponseInputItemUnionParam) string {
	t.Helper()
	if item.OfFunctionCallOutput == nil {
		t.Fatal("resultado da tool sem function_call_output")
	}
	return item.OfFunctionCallOutput.Output.OfString.Value
}

// TestCallFunctionResolvesBothRoots mede a regra pela porta de entrada, contra o
// espelho de verdade — o que prova o defeito que o drive do ticket 07 pegou: o
// tech-pack do Teacher resolvia pelo CWD do processo (cmd/ui) e falhava. Aqui o
// CWD é internal/tools, e o material tem de ser achado do mesmo jeito.
func TestCallFunctionResolvesBothRoots(t *testing.T) {
	workspace := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	const material = "resources/agents/teacher/AGENT.md"

	// Primeira direção: a leitura pelo prefixo acha o material, sem Workspace e
	// sem depender de onde o processo começou.
	got := toolOutput(t, CallFunction(t.Context(), toolCall(t, filetools.READ_TOOL_NAME, map[string]string{"filePath": material}), ""))
	if strings.HasPrefix(got, "Error") || !strings.Contains(got, "Teacher") {
		t.Fatalf("leitura do material do agente = %q..., want o AGENT.md do Teacher", got[:min(len(got), 120)])
	}

	// Segunda direção, a que o ticket exige: a escrita pelo mesmo prefixo falha,
	// e o arquivo no espelho fica como estava.
	mirrored, err := config.MaterialFile(material)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(mirrored)
	if err != nil {
		t.Fatal(err)
	}

	refusal := toolOutput(t, CallFunction(t.Context(), toolCall(t, filetools.WRITE_TOOL_NAME, map[string]string{
		"filePath": material,
		"content":  "sobrescrito pelo modelo",
	}), workspace))
	if !strings.HasPrefix(refusal, filetools.READ_ONLY_ROOT_NOTICE) {
		t.Errorf("escrita no material = %q, want começando com %q", refusal, filetools.READ_ONLY_ROOT_NOTICE)
	}

	after, err := os.ReadFile(mirrored)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("o material do agente mudou depois de uma escrita recusada")
	}

	// E a escrita do usuário cai no Workspace, não no CWD: é a medição do ticket
	// 02 com duas raízes no lugar de uma.
	if out := toolOutput(t, CallFunction(t.Context(), toolCall(t, filetools.WRITE_TOOL_NAME, map[string]string{
		"filePath": "frr.conf",
		"content":  "router bgp 65001",
	}), workspace)); strings.HasPrefix(out, "Error") {
		t.Fatalf("escrita no Workspace falhou: %q", out)
	}

	if _, err := os.Stat(filepath.Join(workspace, "frr.conf")); err != nil {
		t.Errorf("frr.conf não apareceu no Workspace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, "frr.conf")); err == nil {
		t.Error("frr.conf apareceu no CWD do processo: a raiz voltou a ser o CWD")
	}
}
