package turn

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
)

// TestFailureWarningDizOQueOAppFez fixa as quatro formas do Aviso (ticket 19,
// §3): ação + caminho pedido + pasta + motivo curto, a pasta de fora no caso
// que não tem pasta, o motivo cru só no caso inesperado, e a tool que não é
// file tool pelo nome.
//
// A terceira linha é a que mede o motivo tipado valendo mais que o texto do
// erro: o ErrOutsideRoot sobe embrulhado com o erro do SafePath, que é uma
// frase em inglês, e o Aviso é em português.
func TestFailureWarningDizOQueOAppFez(t *testing.T) {
	cases := []struct {
		nome string
		fail tools.Failure
		want string
	}{
		{
			nome: "leitura não encontrada diz a pasta onde a tool olhou",
			fail: tools.Failure{Tool: filetools.READ_TOOL_NAME, Path: "frr.conf", Root: `C:\configs`, Err: filetools.ErrNotFound},
			want: `leitura de "frr.conf" falhou na pasta "C:\configs": arquivo não encontrado`,
		},
		{
			nome: "sem pasta escolhida não há pasta a mostrar",
			fail: tools.Failure{Tool: filetools.WRITE_TOOL_NAME, Path: "frr.conf", Err: filetools.ErrNoWorkspace},
			want: `gravação de "frr.conf" falhou: nenhuma pasta escolhida para os arquivos do usuário`,
		},
		{
			nome: "o motivo tipado vence o erro embrulhado",
			fail: tools.Failure{
				Tool: filetools.SEARCH_TOOL_NAME,
				Path: "../..",
				Root: `C:\configs`,
				Err:  fmt.Errorf("%w: %w", filetools.ErrOutsideRoot, exception.InvalidFilePath),
			},
			want: `busca em "../.." falhou na pasta "C:\configs": caminho fora da raiz`,
		},
		{
			nome: "o caso inesperado mostra o erro cru",
			fail: tools.Failure{
				Tool: filetools.EDIT_TOOL_NAME,
				Path: "frr.conf",
				Root: `C:\configs`,
				Err:  fmt.Errorf("gravando o arquivo: %w", fs.ErrPermission),
			},
			want: `edição de "frr.conf" falhou na pasta "C:\configs": gravando o arquivo: permission denied`,
		},
		{
			nome: "tool que não é file tool aparece pelo nome",
			fail: tools.Failure{Tool: "use_skill", Err: errors.New("skill desconhecida")},
			want: "a tool use_skill falhou: skill desconhecida",
		},
	}

	for _, c := range cases {
		if got := failureWarning(c.fail); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.nome, got, c.want)
		}
	}
}

// TestWarnFailuresDedupeiaPorTextoIdentico fixa o §2: duas tentativas no mesmo
// arquivo pelo mesmo motivo são uma entrada, e os avisos que já estavam na
// lista (os fallbacks da composição) ficam onde estavam.
func TestWarnFailuresDedupeiaPorTextoIdentico(t *testing.T) {
	out := Outcome{Warnings: []string{"retrieval failed, continuing without RAG context: db down"}}
	mesma := tools.Failure{Tool: filetools.READ_TOOL_NAME, Path: "frr.conf", Root: `C:\configs`, Err: filetools.ErrNotFound}
	outra := tools.Failure{Tool: filetools.READ_TOOL_NAME, Path: "bgpd.conf", Root: `C:\configs`, Err: filetools.ErrNotFound}

	// Duas chamadas de propósito: o turno tem duas de Chat (a final e o
	// re-ask do diagrama), e a repetição atravessa as duas.
	warnFailures(&out, []tools.Failure{mesma, mesma, outra})
	warnFailures(&out, []tools.Failure{mesma})

	want := []string{
		"retrieval failed, continuing without RAG context: db down",
		`leitura de "frr.conf" falhou na pasta "C:\configs": arquivo não encontrado`,
		`leitura de "bgpd.conf" falhou na pasta "C:\configs": arquivo não encontrado`,
	}
	if len(out.Warnings) != len(want) {
		t.Fatalf("avisos = %q, esperava %d entradas (a mesma falha é uma)", out.Warnings, len(want))
	}
	for i, w := range want {
		if out.Warnings[i] != w {
			t.Errorf("aviso %d = %q, esperava %q", i, out.Warnings[i], w)
		}
	}
}

// TestRunDrenaAsFalhasDeToolParaOsAvisos é a trava do §1: o canal que o ticket
// 18 abriu sai do `_` e chega a Outcome.Warnings. Com um provedor que chama o
// ReadFile num arquivo que não existe, o turno segue vivo — falha de tool é
// degradação, não turno morto — e o Aviso nomeia o Workspace contra o qual a
// tool resolveu.
//
// Mede nas duas direções: o mesmo turno com o arquivo no lugar não avisa nada,
// porque sucesso não avisa.
func TestRunDrenaAsFalhasDeToolParaOsAvisos(t *testing.T) {
	workspace := t.TempDir()
	turno := func(t *testing.T) Outcome {
		t.Helper()
		fake := &fakeLLM{
			classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"concept","subagent":"modifier"}`,
			toolCall:       "frr.conf",
		}
		base := newFakeAgentIn(t, fake, workspace)
		stubRetrieval(t, []rag.Result{{Query: "q", Protocol: "bgp"}}, nil)

		out, _, err := Run(context.Background(), base, nil, "leia o frr.conf", State{})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if out.Text != "ok" {
			t.Fatalf("Text = %q, esperava o turno respondendo depois da tool", out.Text)
		}
		return out
	}

	out := turno(t)
	if len(out.Warnings) != 1 {
		t.Fatalf("avisos = %q, esperava exatamente o da leitura recusada", out.Warnings)
	}
	if !strings.Contains(out.Warnings[0], workspace) {
		t.Errorf("aviso = %q, esperava a pasta %q contra a qual a tool resolveu", out.Warnings[0], workspace)
	}
	if !strings.Contains(out.Warnings[0], filetools.ErrNotFound.Error()) {
		t.Errorf("aviso = %q, esperava o motivo %q", out.Warnings[0], filetools.ErrNotFound)
	}

	if err := os.WriteFile(filepath.Join(workspace, "frr.conf"), []byte("router bgp 65001\n"), 0o644); err != nil {
		t.Fatalf("semeando o arquivo: %v", err)
	}
	if out := turno(t); len(out.Warnings) != 0 {
		t.Errorf("leitura que deu certo avisou %q: sucesso não avisa", out.Warnings)
	}
}
