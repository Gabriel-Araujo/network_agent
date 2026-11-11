package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// TestBuildInputProjetaOTranscript fixa a projeção transcript → itens de input:
// system prompt, depois o transcript na ordem do disco, depois a mensagem nova.
// Mensagens de conteúdo vazio e de status error ficam de fora — um turno que
// falhou não pode voltar como contexto.
func TestBuildInputProjetaOTranscript(t *testing.T) {
	a := &Agent{systemPrompt: "PROMPT DO SISTEMA"}

	transcript := []history.Message{
		{Role: history.RoleUser, Content: "pergunta", Status: history.StatusDone},
		{Role: history.RoleAgent, Content: "resposta", Status: history.StatusDone},
		{Role: history.RoleAgent, Content: "   ", Status: history.StatusDone},
		{Role: history.RoleAgent, Content: "quebrou", Status: history.StatusError},
	}

	got := a.buildInput(transcript, "nova")

	want := []struct{ role, content string }{
		{"system", "PROMPT DO SISTEMA"},
		{"user", "pergunta"},
		{"assistant", "resposta"},
		{"user", "nova"},
	}

	if len(got) != len(want) {
		t.Fatalf("esperava %d itens, veio %d", len(want), len(got))
	}

	for i, w := range want {
		msg := got[i].OfMessage
		if msg == nil {
			t.Fatalf("item %d não é mensagem", i)
		}
		if string(msg.Role) != w.role {
			t.Errorf("item %d: papel %q, esperava %q", i, msg.Role, w.role)
		}
		if msg.Content.OfString.Value != w.content {
			t.Errorf("item %d: conteúdo %q, esperava %q", i, msg.Content.OfString.Value, w.content)
		}
	}
}

// TestChatReenviaOTurnoSemEstadoNoServidor prova o que o ticket 12 decidiu: o
// turno não depende de estado no provedor. Com um provedor de mentira que
// responde uma tool call e depois a resposta final, a segunda requisição tem
// que levar a chamada que produziu o resultado — com o call id casando as duas
// — e nenhuma previous_response_id.
func TestChatReenviaOTurnoSemEstadoNoServidor(t *testing.T) {
	var requests []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("corpo da requisição ilegível: %v", err)
		}
		requests = append(requests, body)

		w.Header().Set("Content-Type", "application/json")

		if len(requests) == 1 {
			io.WriteString(w, `{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"modelo",
				"output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"ferramenta_inexistente","arguments":"{}","status":"completed"}],
				"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
			return
		}

		io.WriteString(w, `{"id":"resp_2","object":"response","created_at":0,"status":"completed","model":"modelo",
			"output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"pronto","annotations":[]}]}],
			"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
	}))
	defer server.Close()

	a := &Agent{
		Client:       openai.NewClient(option.WithBaseURL(server.URL), option.WithAPIKey("teste")),
		ModelName:    "modelo",
		systemPrompt: "PROMPT",
	}

	resp, _, err := a.Chat(context.Background(), nil, "use a ferramenta")
	if err != nil {
		t.Fatalf("Chat devolveu erro: %v", err)
	}
	if got := resp.OutputText(); got != "pronto" {
		t.Errorf("texto da resposta %q, esperava %q", got, "pronto")
	}

	if len(requests) != 2 {
		t.Fatalf("esperava 2 requisições ao provedor, veio %d", len(requests))
	}

	if _, ok := requests[1]["previous_response_id"]; ok {
		t.Error("a segunda requisição mandou previous_response_id: o turno não é sem estado")
	}

	input, ok := requests[1]["input"].([]any)
	if !ok {
		t.Fatalf("input da segunda requisição não é uma lista de itens: %v", requests[1]["input"])
	}

	// system, user, a chamada reenviada e o resultado dela.
	if len(input) != 4 {
		t.Fatalf("esperava 4 itens no input, veio %d: %v", len(input), input)
	}

	call, ok := input[2].(map[string]any)
	if !ok || call["type"] != "function_call" {
		t.Fatalf("item 2 não é a chamada de ferramenta: %v", input[2])
	}
	if call["call_id"] != "call_1" {
		t.Errorf("call id da chamada %v, esperava call_1", call["call_id"])
	}

	result, ok := input[3].(map[string]any)
	if !ok || result["type"] != "function_call_output" {
		t.Fatalf("item 3 não é o resultado da ferramenta: %v", input[3])
	}
	if result["call_id"] != "call_1" {
		t.Errorf("call id do resultado %v, esperava call_1 — sem ele o provedor não casa chamada e resultado", result["call_id"])
	}
}

func TestConnectConfiguraJanelaDoLMStudioAntesDoChat(t *testing.T) {
	const contextLength = int64(65536)
	loaded := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"models":[{"key":"modelo","loaded_instances":[{"config":{"context_length":8192}}]}]}`)
			return

		case "/api/v1/models/load":
			var body struct {
				Model         string `json:"model"`
				ContextLength int64  `json:"context_length"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("corpo de carga ilegível: %v", err)
			}
			if body.Model != "modelo" {
				t.Errorf("modelo carregado %q, esperava modelo", body.Model)
			}
			if body.ContextLength != contextLength {
				t.Errorf("janela carregada %d, esperava %d", body.ContextLength, contextLength)
			}
			loaded = true
			w.WriteHeader(http.StatusOK)
			return

		case "/v1/responses":
			if !loaded {
				http.Error(w, `{"error":{"message":"request exceeds the available context size (8192)"}}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"modelo","output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"pronto","annotations":[]}]}],"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
		}
	}))
	defer server.Close()

	ag, err := Connect(types.Config{
		Providers: []types.Provider{{
			Name:   "LM Studio",
			Url:    server.URL + "/v1",
			Models: []types.Model{{Id: "modelo", ContextSize: contextLength}},
		}},
	}, "")
	if err != nil {
		t.Fatalf("Connect devolveu erro: %v", err)
	}
	if _, _, err := ag.Chat(context.Background(), nil, "pergunta"); err != nil {
		t.Fatalf("Chat devolveu erro: %v", err)
	}
}

// scriptedProvider serves the given response bodies in order and records every
// request body it got. The last body is reused if the turn asks for more.
func scriptedProvider(t *testing.T, bodies ...string) (*openai.Client, *[]map[string]any) {
	t.Helper()
	var requests []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("corpo da requisição ilegível: %v", err)
		}
		requests = append(requests, body)

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, bodies[min(len(requests), len(bodies))-1])
	}))
	t.Cleanup(server.Close)

	client := openai.NewClient(option.WithBaseURL(server.URL), option.WithAPIKey("teste"))
	return &client, &requests
}

// callsOneTool is a provider response with a single function call.
func callsOneTool(name, arguments string) string {
	args, _ := json.Marshal(arguments)
	return `{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"modelo",
		"output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"` + name + `","arguments":` + string(args) + `,"status":"completed"}],
		"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`
}

// answers is a provider response with a final message.
func answers(text string) string {
	return `{"id":"resp_2","object":"response","created_at":0,"status":"completed","model":"modelo",
		"output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"` + text + `","annotations":[]}]}],
		"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`
}

// toolOutputs returns the function_call_output texts the turn sent back to the
// provider in its last request — what the model actually reads.
func toolOutputs(t *testing.T, request map[string]any) []string {
	t.Helper()
	input, ok := request["input"].([]any)
	if !ok {
		t.Fatalf("input não é uma lista de itens: %v", request["input"])
	}
	var out []string
	for _, item := range input {
		m, ok := item.(map[string]any)
		if ok && m["type"] == "function_call_output" {
			out = append(out, fmt.Sprint(m["output"]))
		}
	}
	return out
}

// subagentAgent builds the dispatched Agent of the named Subagente against a
// throwaway configs root, so the test does not depend on the repo's AGENT.md.
func subagentAgent(t *testing.T, client *openai.Client, name string) *Agent {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating the %s folder: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte("You are the "+name+"."), 0o644); err != nil {
		t.Fatalf("writing %s/AGENT.md: %v", name, err)
	}
	base := &Agent{Client: *client, ModelName: "modelo"}
	sub, err := base.forSubagentAt(root, name)
	if err != nil {
		t.Fatalf("forSubagentAt(%q): %v", name, err)
	}
	return sub
}

// TestCallToolRefusesTheToolNotOffered: the Modifier that names WriteFile reads
// the refusal and nothing reaches the disk. The prompt was not enough (ticket
// 13, K2); the toolset plus this check is.
func TestCallToolRefusesTheToolNotOffered(t *testing.T) {
	client, requests := scriptedProvider(t,
		callsOneTool("WriteFile", `{"filePath":"frr.conf","content":"router bgp 65001\n"}`),
		answers("pronto"))
	modifier := subagentAgent(t, client, "modifier")

	// The file tools resolve relative paths against the working directory.
	work := t.TempDir()
	t.Chdir(work)

	if _, _, err := modifier.Chat(context.Background(), nil, "edite o frr.conf"); err != nil {
		t.Fatalf("Chat devolveu erro: %v", err)
	}

	if len(*requests) != 2 {
		t.Fatalf("esperava 2 requisições ao provedor, veio %d", len(*requests))
	}
	got := toolOutputs(t, (*requests)[1])
	if len(got) != 1 || !strings.Contains(got[0], "was not offered") {
		t.Errorf("resultado da tool %q, esperava a recusa de WriteFile", got)
	}
	if _, err := os.Stat(filepath.Join(work, "frr.conf")); !os.IsNotExist(err) {
		t.Errorf("o WriteFile recusado gravou o arquivo (stat err = %v)", err)
	}
}

// TestCallToolRefusesSubAgentToADispatchedSubagente: "no nesting" is in the
// code, not only in the map's Notes — the sub_agent named by a dispatched
// Subagente never spawns anything (a spawned sub would hit the provider).
func TestCallToolRefusesSubAgentToADispatchedSubagente(t *testing.T) {
	client, requests := scriptedProvider(t,
		callsOneTool("sub_agent", `{"task":"escreva o frr.conf"}`),
		answers("pronto"))
	builder := subagentAgent(t, client, "builder")

	if _, _, err := builder.Chat(context.Background(), nil, "delegue"); err != nil {
		t.Fatalf("Chat devolveu erro: %v", err)
	}

	if len(*requests) != 2 {
		t.Fatalf("esperava 2 requisições (nenhum sub-agente rodou), veio %d", len(*requests))
	}
	got := toolOutputs(t, (*requests)[1])
	if len(got) != 1 || !strings.Contains(got[0], "was not offered") {
		t.Errorf("resultado da tool %q, esperava a recusa de sub_agent", got)
	}
}

// TestCallToolKeepsSubAgentForTheGeneric: the refusal cuts only what was not
// offered. The generic agent carries sub_agent, so it still spawns — the sub's
// own trip to the provider is the proof.
func TestCallToolKeepsSubAgentForTheGeneric(t *testing.T) {
	client, requests := scriptedProvider(t,
		callsOneTool("sub_agent", `{"task":"escreva o frr.conf"}`),
		answers("sub pronto"),
		answers("pronto"))
	generic := &Agent{Client: *client, ModelName: "modelo", availableTools: tools.GetSkillAndTools()}

	if _, _, err := generic.Chat(context.Background(), nil, "delegue"); err != nil {
		t.Fatalf("Chat devolveu erro: %v", err)
	}

	if len(*requests) != 3 {
		t.Fatalf("esperava 3 requisições (a do meio é do sub-agente), veio %d", len(*requests))
	}
	got := toolOutputs(t, (*requests)[2])
	if len(got) != 1 || got[0] != "sub pronto" {
		t.Errorf("resultado da tool %q, esperava a resposta do sub-agente", got)
	}
}

// TestPromptSkillsMountedInTheSubagentPrompt: the Illustrator's system prompt
// carries the mermaid syntax before the turn starts, without the model calling
// anything. The model never runs a consultative read (use_skill 0/15,
// ReadFile of the tech pack 0/4), so an activation the model has to choose is
// not a delivery mechanism. The control is the table: with the illustrator
// entry gone the prompt loses the syntax and this test fails.
func TestPromptSkillsMountedInTheSubagentPrompt(t *testing.T) {
	root := t.TempDir()
	agentMD := "You are the Illustrator."
	dir := filepath.Join(root, "illustrator")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating the illustrator folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(agentMD), 0o644); err != nil {
		t.Fatalf("writing illustrator/AGENT.md: %v", err)
	}

	sub, err := (&Agent{}).forSubagentAt(root, "illustrator")
	if err != nil {
		t.Fatalf("forSubagentAt(illustrator): %v", err)
	}

	mermaid, ok := skills.LoadSkills().Get("mermaid")
	if !ok {
		t.Fatal("the mermaid skill is not in the registry")
	}
	if want := agentMD + "\n\n" + mermaid.Body; sub.systemPrompt != want {
		t.Errorf("systemPrompt = %q,\nwant the AGENT.md then the mermaid body, blank line between", sub.systemPrompt)
	}
	if !strings.Contains(sub.systemPrompt, "Parentheses outside quotes break the diagram") {
		t.Error("the mounted prompt does not carry the quotes rule, the one ticket 16 corrected")
	}
}

// TestPromptSkillsUnknownNameIsAnExplicitError: a name in the table that the
// registry does not have fails the dispatch, like a missing AGENT.md. The
// table is code; a prompt silently without the syntax is the failure the
// mounting fixes, so it must never be the quiet outcome.
func TestPromptSkillsUnknownNameIsAnExplicitError(t *testing.T) {
	previous := promptSkills
	promptSkills = map[string][]string{"illustrator": {"no-such-skill"}}
	t.Cleanup(func() { promptSkills = previous })

	root := t.TempDir()
	dir := filepath.Join(root, "illustrator")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating the illustrator folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte("You are the Illustrator."), 0o644); err != nil {
		t.Fatalf("writing illustrator/AGENT.md: %v", err)
	}

	sub, err := (&Agent{}).forSubagentAt(root, "illustrator")
	if err == nil {
		t.Fatalf("forSubagentAt returned a prompt %q, want an error naming the missing skill", sub.systemPrompt)
	}
	if !strings.Contains(err.Error(), "no-such-skill") {
		t.Errorf("err = %v, want the missing skill named", err)
	}
}

// TestPromptSkillsLeavesTheBuilderAlone: the Builder does not draw, so its
// prompt is its AGENT.md and nothing else. Mounting everywhere would be the
// easy mistake; the table says who draws.
func TestPromptSkillsLeavesTheBuilderAlone(t *testing.T) {
	root := t.TempDir()
	agentMD := "You are the Builder."
	dir := filepath.Join(root, "builder")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating the builder folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(agentMD), 0o644); err != nil {
		t.Fatalf("writing builder/AGENT.md: %v", err)
	}

	sub, err := (&Agent{}).forSubagentAt(root, "builder")
	if err != nil {
		t.Fatalf("forSubagentAt(builder): %v", err)
	}
	if sub.systemPrompt != agentMD {
		t.Errorf("systemPrompt = %q, want the AGENT.md untouched", sub.systemPrompt)
	}
}

// TestActivationSentenceGoneFromTheDrawingPrompts: the two AGENT.md files no
// longer tell the model to activate the skill. The sentence asked for
// something already in the prompt and that the model never did; leaving it
// would teach a call that buys nothing.
func TestActivationSentenceGoneFromTheDrawingPrompts(t *testing.T) {
	for _, name := range []string{"illustrator", "teacher"} {
		path, err := paths.RepoFile(filepath.Join("resources/agents", name, "AGENT.md"))
		if err != nil {
			t.Fatalf("locating %s/AGENT.md: %v", name, err)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s/AGENT.md: %v", name, err)
		}
		if strings.Contains(string(body), skills.ToolUseSkill) {
			t.Errorf("%s/AGENT.md still names %s: the skill is mounted, not activated", name, skills.ToolUseSkill)
		}
	}
}

// TestChatCarriesTheToolFailureOutOfTheTurn is ticket 18's proof of arrival: a
// refused read has to come out of Chat as a typed record, and the test fails if
// the channel is cut on any of its three floors — the file tool swallowing the
// error, CallFunction dropping the record, or Chat not accumulating it. The two
// directions are in the same test: the same call with the file in place must
// report nothing, because a success that warns blinds the channel.
func TestChatCarriesTheToolFailureOutOfTheTurn(t *testing.T) {
	workspace := t.TempDir()

	chat := func(t *testing.T) ([]tools.Failure, []string) {
		t.Helper()
		client, requests := scriptedProvider(t,
			callsOneTool(filetools.READ_TOOL_NAME, `{"filePath":"frr.conf"}`),
			answers("pronto"))
		ag := &Agent{Client: *client, ModelName: "modelo", availableTools: tools.Tools, workspace: workspace}

		_, failures, err := ag.Chat(context.Background(), nil, "leia o frr.conf")
		if err != nil {
			t.Fatalf("Chat devolveu erro: %v", err)
		}
		if len(*requests) != 2 {
			t.Fatalf("esperava 2 requisições ao provedor, veio %d", len(*requests))
		}
		return failures, toolOutputs(t, (*requests)[1])
	}

	// Primeira direção: o arquivo não existe, e a falha chega.
	failures, outputs := chat(t)
	if len(failures) != 1 {
		t.Fatalf("falhas = %+v, want exatamente uma: a leitura recusada", failures)
	}
	fail := failures[0]
	if fail.Tool != filetools.READ_TOOL_NAME {
		t.Errorf("tool da falha = %q, want %q", fail.Tool, filetools.READ_TOOL_NAME)
	}
	// O caminho é o que o modelo pediu, não o absoluto que a tool resolveu: o
	// absoluto já está no log.Error, e o Aviso fala com quem pediu.
	if fail.Path != "frr.conf" {
		t.Errorf("caminho da falha = %q, want %q", fail.Path, "frr.conf")
	}
	if fail.Root != workspace {
		t.Errorf("raiz da falha = %q, want o Workspace %q", fail.Root, workspace)
	}
	if !errors.Is(fail.Err, filetools.ErrNotFound) {
		t.Errorf("erro da falha = %v, want ErrNotFound", fail.Err)
	}

	// E a string que o modelo lê não mudou: o erro é adicional, não substituto.
	if len(outputs) != 1 || !strings.HasPrefix(outputs[0], filetools.READ_NOT_FOUND_NOTICE) {
		t.Errorf("resultado da tool = %q, want começando com o notice de não-encontrado", outputs)
	}

	// Segunda direção: com o arquivo no lugar, o mesmo turno não reporta nada.
	if err := os.WriteFile(filepath.Join(workspace, "frr.conf"), []byte("router bgp 65001\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if failures, outputs := chat(t); len(failures) != 0 {
		t.Errorf("leitura que deu certo reportou %+v (resultado %q): sucesso não avisa", failures, outputs)
	}
}

// TestChatComposesTheNestedSubagentFailures fixa a §2 do ticket 18: o Chat do
// sub-agente roda dentro de uma tool do de fora, então uma leitura recusada lá
// dentro é degradação deste turno também. Sem a composição, a falha morreria no
// retorno do aninhado, que ninguém lê.
func TestChatComposesTheNestedSubagentFailures(t *testing.T) {
	workspace := t.TempDir()
	client, requests := scriptedProvider(t,
		callsOneTool(tools.SUB_AGENT_TOOL_NAME, `{"task":"leia o frr.conf"}`),
		callsOneTool(filetools.READ_TOOL_NAME, `{"filePath":"frr.conf"}`),
		answers("sub pronto"),
		answers("pronto"))
	generic := &Agent{Client: *client, ModelName: "modelo", availableTools: tools.GetSkillAndTools(), workspace: workspace}

	_, failures, err := generic.Chat(context.Background(), nil, "delegue a leitura")
	if err != nil {
		t.Fatalf("Chat devolveu erro: %v", err)
	}
	if len(*requests) != 4 {
		t.Fatalf("esperava 4 requisições (duas do sub-agente), veio %d", len(*requests))
	}
	if len(failures) != 1 || !errors.Is(failures[0].Err, filetools.ErrNotFound) {
		t.Fatalf("falhas = %+v, want a leitura recusada dentro do sub-agente", failures)
	}
	if failures[0].Tool != filetools.READ_TOOL_NAME {
		t.Errorf("tool da falha = %q, want a do aninhado (%q), não a do sub_agent", failures[0].Tool, filetools.READ_TOOL_NAME)
	}
}
