package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llm2 "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/openai/openai-go/v3/responses"
)

// repoRoot sobe até o diretório que contém go.mod (raiz do projeto).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod não encontrado acima do cwd")
		}
		dir = parent
	}
}

// callOutput extrai o texto de saída de um function call output.
func callOutput(t *testing.T, res responses.ResponseInputItemUnionParam) string {
	t.Helper()
	anyOut := res.GetOutput().AsAny()
	out, ok := anyOut.(*string)
	if !ok {
		t.Fatalf("saída da chamada não é *string, é %T", anyOut)
	}
	return *out
}

func TestToolCallReadsFileFromRepoRoot(t *testing.T) {
	// toolCall despacha para tools.CallFunction, que usa workingDirectory=".".
	root := repoRoot(t)
	t.Chdir(root)

	a := &llm2.Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_1",
		Name:   "ReadFile",
		Arguments: `{
			"filePath": "go.mod"
		}`,
	}

	res, _ := a.ToolCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" {
		t.Fatalf("esperava conteúdo de go.mod, veio vazio")
	}
}

func TestToolCallInvalidFunction(t *testing.T) {
	t.Chdir(repoRoot(t))

	a := &llm2.Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_x",
		Name:   "NaoExiste",
		Arguments: `{
			"qualquer": "coisa"
		}`,
	}

	res, _ := a.ToolCall(context.Background(), call)

	got := callOutput(t, res)
	if got != "Error: Called invalid function" {
		t.Fatalf("esperava 'Error: Called invalid function', veio %q", got)
	}
}

func TestSkillCallUseExistingSkill(t *testing.T) {
	// LoadSkills usa caminho relativo ./resources/agent/skills.
	root := repoRoot(t)
	t.Chdir(root)

	a := &llm2.Agent{}

	// existe e tem body conhecido na raiz do repo
	reg := skills.LoadSkills()
	sk, ok := reg.Get("frr-file-builder")
	if !ok {
		t.Fatalf("skill frr-file-builder não encontrada na raiz do repo")
	}

	call := responses.ResponseFunctionToolCall{
		CallID: "call_skill",
		Name:   skills.ToolUseSkill,
		Arguments: `{
			"skill_name": "frr-file-builder"
		}`,
	}

	res, _ := a.SkillCall(context.Background(), call)

	// A ativação da frr-file-builder abre com o corpo; o que vem depois (pack
	// do turno ou a linha de protocolo vazio) é de TestActivationDeliversTheTurnPack.
	got := callOutput(t, res)
	if !strings.HasPrefix(got, sk.Body) {
		t.Fatalf("skillCall deve devolver o body da skill.\nesperado: %q\nobtido:   %q", sk.Body, got)
	}
}

func TestSkillCallUnknownSkill(t *testing.T) {
	t.Chdir(repoRoot(t))

	a := &llm2.Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_skill",
		Name:   skills.ToolUseSkill,
		Arguments: `{
			"skill_name": "nao-existe"
		}`,
	}

	res, _ := a.SkillCall(context.Background(), call)

	got := callOutput(t, res)
	if got != `erro: skill "nao-existe" não encontrada` {
		t.Fatalf("mensagem inesperada: %q", got)
	}
}

func TestSkillCallInvalidArguments(t *testing.T) {
	// skillCall -> HandleSkillCall -> LoadSkills() usa caminho relativo.
	t.Chdir(repoRoot(t))

	a := &llm2.Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_skill",
		Name:   skills.ToolUseSkill,
		Arguments: `{
			"skill_name": 
		}`,
	}

	res, _ := a.SkillCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" || got[0:len("erro: argumentos inválidos")] != "erro: argumentos inválidos" {
		t.Fatalf("esperava erro de argumentos inválidos, veio %q", got)
	}
}

func TestSkillCallReadSkillFile(t *testing.T) {
	root := repoRoot(t)
	t.Chdir(root)

	a := &llm2.Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_read",
		Name:   skills.ToolReadSkillFile,
		Arguments: `{
			"skill_name": "frr-file-builder",
			"relative_path": "SKILL.md"
		}`,
	}

	res, _ := a.SkillCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" {
		t.Fatalf("esperava conteúdo de SKILL.md via read_skill_file")
	}
	// SKILL.md começa com frontmatter yaml (---)
	if got[0:3] != "---" {
		t.Fatalf("SKILL.md deveria começar com '---', veio %q", got[:min(3, len(got))])
	}
}
