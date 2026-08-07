package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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

// newRecordingServer retorna um servidor mockado da API de Responses
// que devolve sempre o mesmo texto e registra o corpo da requisição.
func newRecordingServer(t *testing.T, replyText string) (*httptest.Server, *[]byte) {
	t.Helper()
	var lastBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		lastBody = body
		w.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"id":     "resp_test_1",
			"object": "response",
			"output": []map[string]any{
				{
					"type":    "message",
					"role":    "assistant",
					"content": []map[string]any{{"type": "output_text", "text": replyText}},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(ts.Close)
	return ts, &lastBody
}

func TestToolCallReadsFileFromRepoRoot(t *testing.T) {
	// toolCall despacha para tools.CallFunction, que usa workingDirectory=".".
	root := repoRoot(t)
	t.Chdir(root)

	a := &Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_1",
		Name:   "ReadFile",
		Arguments: `{
			"filePath": "go.mod"
		}`,
	}

	res := a.toolCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" {
		t.Fatalf("esperava conteúdo de go.mod, veio vazio")
	}
}

func TestToolCallInvalidFunction(t *testing.T) {
	t.Chdir(repoRoot(t))

	a := &Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_x",
		Name:   "NaoExiste",
		Arguments: `{
			"qualquer": "coisa"
		}`,
	}

	res := a.toolCall(context.Background(), call)

	got := callOutput(t, res)
	if got != "Error: Called invalid function" {
		t.Fatalf("esperava 'Error: Called invalid function', veio %q", got)
	}
}

func TestSkillCallUseExistingSkill(t *testing.T) {
	// LoadSkills usa caminho relativo ./resources/agent/skills.
	root := repoRoot(t)
	t.Chdir(root)

	a := &Agent{}

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

	res := a.skillCall(context.Background(), call)

	got := callOutput(t, res)
	if got != sk.Body {
		t.Fatalf("skillCall deve devolver o body da skill.\nesperado: %q\nobtido:   %q", sk.Body, got)
	}
}

func TestSkillCallUnknownSkill(t *testing.T) {
	t.Chdir(repoRoot(t))

	a := &Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_skill",
		Name:   skills.ToolUseSkill,
		Arguments: `{
			"skill_name": "nao-existe"
		}`,
	}

	res := a.skillCall(context.Background(), call)

	got := callOutput(t, res)
	if got != `erro: skill "nao-existe" não encontrada` {
		t.Fatalf("mensagem inesperada: %q", got)
	}
}

func TestSkillCallInvalidArguments(t *testing.T) {
	// skillCall -> HandleSkillCall -> LoadSkills() usa caminho relativo.
	t.Chdir(repoRoot(t))

	a := &Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_skill",
		Name:   skills.ToolUseSkill,
		Arguments: `{
			"skill_name": 
		}`,
	}

	res := a.skillCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" || got[0:len("erro: argumentos inválidos")] != "erro: argumentos inválidos" {
		t.Fatalf("esperava erro de argumentos inválidos, veio %q", got)
	}
}

func TestSkillCallReadSkillFile(t *testing.T) {
	root := repoRoot(t)
	t.Chdir(root)

	a := &Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_read",
		Name:   skills.ToolReadSkillFile,
		Arguments: `{
			"skill_name": "frr-file-builder",
			"relative_path": "SKILL.md"
		}`,
	}

	res := a.skillCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" {
		t.Fatalf("esperava conteúdo de SKILL.md via read_skill_file")
	}
	// SKILL.md começa com frontmatter yaml (---)
	if got[0:3] != "---" {
		t.Fatalf("SKILL.md deveria começar com '---', veio %q", got[:min(3, len(got))])
	}
}

func TestSubAgentCallDelegatesToSubAgent(t *testing.T) {
	const reply = "resposta do sub-agente"

	ts, lastBody := newRecordingServer(t, reply)
	client := openai.NewClient(
		option.WithBaseURL(ts.URL),
		option.WithAPIKey("test-key"),
	)

	a := &Agent{
		Client:       client,
		ModelName:    "test-model",
		SystemPrompt: "você é um assistente",
	}

	call := responses.ResponseFunctionToolCall{
		CallID: "call_sub",
		Name:   "sub_agent",
		Arguments: `{
			"task": "faça alguma coisa"
		}`,
	}

	res := a.subAgentCall(context.Background(), call)

	got := callOutput(t, res)
	if got != reply {
		t.Fatalf("subAgentCall deve devolver a resposta do sub-agente.\nesperado: %q\nobtido:   %q", reply, got)
	}

	// o sub-agente deve ter enviado a task como mensagem para a API
	if !bytes.Contains(*lastBody, []byte("faça alguma coisa")) {
		t.Fatalf("corpo da requisição não contém a task: %s", *lastBody)
	}
}

func TestSubAgentCallInvalidArguments(t *testing.T) {
	a := &Agent{}
	call := responses.ResponseFunctionToolCall{
		CallID: "call_sub",
		Name:   "sub_agent",
		Arguments: `{
			"task": 
		}`,
	}

	res := a.subAgentCall(context.Background(), call)

	got := callOutput(t, res)
	if got == "" || got[0:len("erro: argumentos inválidos")] != "erro: argumentos inválidos" {
		t.Fatalf("esperava erro de argumentos inválidos, veio %q", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
