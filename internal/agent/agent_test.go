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
	"slices"
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
	"github.com/openai/openai-go/v3/responses"
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

	resp, _, _, err := a.Chat(context.Background(), nil, "use a ferramenta")
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
	if _, _, _, err := ag.Chat(context.Background(), nil, "pergunta"); err != nil {
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

	if _, _, _, err := modifier.Chat(context.Background(), nil, "edite o frr.conf"); err != nil {
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

	if _, _, _, err := builder.Chat(context.Background(), nil, "delegue"); err != nil {
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

// TestSubagentSkillActivationNarrowedToThePrompt: each Subagente's skill
// reaches the model through the use_skill call, not through the prompt — the
// body is absent from the system prompt and the activation tool is narrowed to
// exactly the skill its AGENT.md names. The owner chose the call as the only
// path (ticket 35); the narrowing keeps other descriptions from ordering other
// activations — the 12 of 30 wasted calls of the ticket 28 batch. The expected
// catalogs are spelled out here rather than read from subagentSkills, so a
// row added, dropped or widened fails this test instead of agreeing with it.
func TestSubagentSkillActivationNarrowedToThePrompt(t *testing.T) {
	want := map[string][]string{
		"builder":     {"frr-file-builder"},
		"illustrator": {"mermaid"},
		"teacher":     {"mermaid"},
		"modifier":    {"frr-file-builder"},
	}
	root := t.TempDir()
	for name, wantEnum := range want {
		agentMD := "You are the " + name + "."
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating the %s folder: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(agentMD), 0o644); err != nil {
			t.Fatalf("writing %s/AGENT.md: %v", name, err)
		}

		sub, err := (&Agent{}).forSubagentAt(root, name)
		if err != nil {
			t.Fatalf("forSubagentAt(%s): %v", name, err)
		}
		if sub.systemPrompt != agentMD {
			t.Errorf("%s systemPrompt = %q, want the AGENT.md and nothing else — the body travels by use_skill", name, sub.systemPrompt)
		}

		var enum []string
		for _, tool := range sub.availableTools {
			if fn := tool.OfFunction; fn != nil && fn.Name == skills.ToolUseSkill {
				props, _ := fn.Parameters["properties"].(map[string]any)
				skill, _ := props["skill_name"].(map[string]any)
				enum, _ = skill["enum"].([]string)
				if enum == nil {
					t.Errorf("%s offers %s with no enum", name, skills.ToolUseSkill)
				}
			}
		}
		if !slices.Equal(enum, wantEnum) {
			t.Errorf("%s use_skill enum = %v, want exactly %v — only what the prompt names", name, enum, wantEnum)
		}
	}
}

// TestSubagentSkillsUnknownNameIsAnExplicitError: a name in the table that the
// registry does not have fails the dispatch, like a missing AGENT.md. The
// table is code; a toolset silently without the activation is the failure the
// narrowing exists to avoid, so it must never be the quiet outcome.
func TestSubagentSkillsUnknownNameIsAnExplicitError(t *testing.T) {
	previous := subagentSkills
	subagentSkills = map[string][]string{"illustrator": {"no-such-skill"}}
	t.Cleanup(func() { subagentSkills = previous })

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

// TestActivationSentenceInThePrompts: every AGENT.md orders the activation its
// toolset offers, by the tool and by each skill name. The sentence died with
// the mounting (ticket 22) because the body was already in the prompt; with
// the call as the only path (ticket 35) the sentence is what makes it happen.
func TestActivationSentenceInThePrompts(t *testing.T) {
	for name, names := range subagentSkills {
		body := repoAgentMD(t, name)
		if !strings.Contains(body, skills.ToolUseSkill) {
			t.Errorf("%s/AGENT.md does not name %s: the body travels by activation and nothing orders it", name, skills.ToolUseSkill)
		}
		for _, skill := range names {
			if !strings.Contains(body, "`"+skill+"`") {
				t.Errorf("%s/AGENT.md does not name the skill %s its toolset activates", name, skill)
			}
		}
	}
}

// TestBuilderPromptKeepsOnlyTheFlow pins the split of the frr-file-builder
// contract: the Builder's AGENT.md keeps the flow (activate, write, report)
// and no technical rule, which lives in the skill and its pack. The two
// WriteFile openings it tells the model to read are the tool's own constants,
// so a reworded notice cannot leave the "replaced only when" rule reading a
// line that never comes.
func TestBuilderPromptKeepsOnlyTheFlow(t *testing.T) {
	body := repoAgentMD(t, "builder")
	for _, quoted := range []string{filetools.WRITE_CREATED_NOTICE, filetools.WRITE_REPLACED_NOTICE, "! WARNING:"} {
		if !strings.Contains(body, strings.TrimSpace(quoted)) {
			t.Errorf("builder/AGENT.md does not quote %q", strings.TrimSpace(quoted))
		}
	}
	for _, rule := range []string{"router bgp", "neighbor", "prefix-list", "address-family", "frr defaults", "frr version", "remote-as", "ebgp-requires-policy"} {
		if strings.Contains(body, rule) {
			t.Errorf("builder/AGENT.md still carries the technical rule %q: it belongs to the frr-file-builder skill", rule)
		}
	}
}

// TestSubagentFlowsConsumeTheSkillBody drives the real Builder and Modifier —
// AGENT.md and toolset from the mirror — through the flows of the
// frr-file-builder contract, with a provider that scripts the model's calls.
// What is real is everything the model reads back: the body use_skill delivers
// before the WriteFile or EditFile call is decided, with the turn's pack after
// it when the turn has a protocol, the pack read_skill_file reaches when it has
// none, the tool openings the reply has to copy, the refusals, and the disk.
func TestSubagentFlowsConsumeTheSkillBody(t *testing.T) {
	sk, ok := skills.LoadSkills().Get("frr-file-builder")
	if !ok {
		t.Fatal("skill frr-file-builder not loaded")
	}
	const (
		written  = "router bgp 65001\n ! WARNING: no maximum-prefix on 203.0.113.2\n"
		write    = `{"filePath":"frr.conf","content":"router bgp 65001\n ! WARNING: no maximum-prefix on 203.0.113.2\n"}`
		existing = "router bgp 65001\n neighbor 203.0.113.2 remote-as 65010\n"
		edited   = existing + " ! WARNING: no password on 203.0.113.6\n neighbor 203.0.113.6 remote-as 65020\n"
		edit     = `{"filePath":"frr.conf","oldText":" neighbor 203.0.113.2 remote-as 65010\n","newText":" neighbor 203.0.113.2 remote-as 65010\n ! WARNING: no password on 203.0.113.6\n neighbor 203.0.113.6 remote-as 65020\n"}`
		replaced = "router bgp 65001\n neighbor 203.0.113.6 remote-as 65020\n"
		replace  = `{"filePath":"frr.conf","oldText":" neighbor 203.0.113.2 remote-as 65010","newText":" neighbor 203.0.113.6 remote-as 65020"}`
	)
	activate := callsOneTool(skills.ToolUseSkill, `{"skill_name":"frr-file-builder"}`)
	read := callsOneTool(filetools.READ_TOOL_NAME, `{"filePath":"frr.conf"}`)
	readTECH := callsOneTool(skills.ToolReadSkillFile, `{"skill_name":"frr-file-builder","relative_path":"references/bgp/TECH.md"}`)
	failActivation := callsOneTool(skills.ToolUseSkill, `{"skill_name":"no-such-skill"}`)
	bgpTurn := skills.Turn{Protocol: "bgp", Retrieved: true}

	cases := []struct {
		agent, name string
		turn        skills.Turn // what Run hands the Subagente; zero is "nothing classified"
		before      string      // frr.conf before the turn; "" is no file
		script      []string
		wantOutputs []string // prefix of each tool output, in call order
		wantActive  []string
		wantReports int
		after       string // frr.conf after the turn; "" is no file
	}{
		{
			agent: "builder", name: "arquivo novo",
			turn:        bgpTurn,
			script:      []string{activate, callsOneTool(filetools.WRITE_TOOL_NAME, write), answers("gravado")},
			wantOutputs: []string{sk.Body, filetools.WRITE_CREATED_NOTICE},
			wantActive:  []string{"frr-file-builder"},
			after:       written,
		},
		{
			agent: "builder", name: "arquivo substituído",
			before:      "hostname old\n",
			script:      []string{activate, callsOneTool(filetools.WRITE_TOOL_NAME, write), answers("gravado")},
			wantOutputs: []string{sk.Body, filetools.WRITE_REPLACED_NOTICE},
			wantActive:  []string{"frr-file-builder"},
			after:       written,
		},
		{
			agent: "builder", name: "pedido incompleto",
			turn:        bgpTurn,
			script:      []string{activate, answers("qual é o AS local?")},
			wantOutputs: []string{sk.Body},
			wantActive:  []string{"frr-file-builder"},
		},
		{
			agent: "builder", name: "falha de ativação",
			script:      []string{failActivation, answers("a skill falhou")},
			wantOutputs: []string{"erro:"},
			wantReports: 1,
		},
		{
			agent: "builder", name: "escrita recusada",
			script: []string{activate,
				callsOneTool(filetools.WRITE_TOOL_NAME, `{"filePath":"resources/frr.conf","content":"router bgp 65001\n"}`),
				answers("a escrita falhou")},
			wantOutputs: []string{sk.Body, filetools.READ_ONLY_ROOT_NOTICE},
			wantActive:  []string{"frr-file-builder"},
			wantReports: 1,
		},
		{
			agent: "modifier", name: "arquivo existe",
			turn:        bgpTurn,
			before:      existing,
			script:      []string{read, activate, callsOneTool(filetools.EDIT_TOOL_NAME, edit), read, answers("editado")},
			wantOutputs: []string{existing, sk.Body, "Success: File edited", edited},
			wantActive:  []string{"frr-file-builder"},
			after:       edited,
		},
		{
			agent: "modifier", name: "substituição que remove linha",
			turn:        bgpTurn,
			before:      existing,
			script:      []string{read, activate, callsOneTool(filetools.EDIT_TOOL_NAME, replace), read, answers("editado")},
			wantOutputs: []string{existing, sk.Body, "Success: File edited. Removed lines: neighbor 203.0.113.2 remote-as 65010", replaced},
			wantActive:  []string{"frr-file-builder"},
			wantReports: 1,
			after:       replaced,
		},
		{
			agent: "modifier", name: "arquivo inexistente",
			script:      []string{read, answers("o arquivo não existe")},
			wantOutputs: []string{filetools.READ_NOT_FOUND_NOTICE},
			wantReports: 1,
		},
		{
			agent: "modifier", name: "falha de ativação",
			before:      existing,
			script:      []string{read, failActivation, answers("a skill falhou")},
			wantOutputs: []string{existing, "erro:"},
			wantReports: 1,
			after:       existing,
		},
		{
			agent: "modifier", name: "pedido incompleto",
			before:      existing,
			script:      []string{read, activate, readTECH, answers("qual é o AS do peer?")},
			wantOutputs: []string{existing, sk.Body, "# BGP"},
			wantActive:  []string{"frr-file-builder"},
			after:       existing,
		},
	}
	for _, tc := range cases {
		t.Run(tc.agent+"/"+tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			conf := filepath.Join(workspace, "frr.conf")
			if tc.before != "" {
				if err := os.WriteFile(conf, []byte(tc.before), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			client, requests := scriptedProvider(t, tc.script...)
			sub, err := (&Agent{Client: *client, ModelName: "modelo", workspace: workspace}).ForSubagent(tc.agent)
			if err != nil {
				t.Fatalf("ForSubagent(%s): %v", tc.agent, err)
			}
			sub = sub.WithSkillTurn(tc.turn)

			_, reports, activated, err := sub.Chat(context.Background(), nil, "frr.conf")
			if err != nil {
				t.Fatalf("Chat devolveu erro: %v", err)
			}

			// Request i+1 is the first to carry tool output i, so the body is
			// in the input of the request whose answer is the WriteFile or
			// EditFile call; the last request carries them all.
			if len(*requests) != len(tc.script) {
				t.Fatalf("%d requisições, esperava %d", len(*requests), len(tc.script))
			}
			outputs := toolOutputs(t, (*requests)[len(*requests)-1])
			if len(outputs) != len(tc.wantOutputs) {
				t.Fatalf("saídas de tool %q, esperava %d", outputs, len(tc.wantOutputs))
			}
			for i, want := range tc.wantOutputs {
				if !strings.HasPrefix(outputs[i], want) {
					t.Errorf("saída %d = %.80q, esperava começar com %.80q", i, outputs[i], want)
				}
			}
			// The turn's pack rides the activation: with a protocol, the
			// model has the doctrine without calling read_skill_file.
			if tc.turn.Protocol != "" && !slices.ContainsFunc(outputs, func(o string) bool {
				return strings.HasPrefix(o, sk.Body) && strings.Contains(o, `<file path="references/`+tc.turn.Protocol+`/doctrine.md">`)
			}) {
				t.Errorf("nenhuma ativação trouxe o pack %s depois do corpo", tc.turn.Protocol)
			}
			if !slices.Equal(activated, tc.wantActive) {
				t.Errorf("skills ativadas = %v, esperava %v", activated, tc.wantActive)
			}
			if len(reports) != tc.wantReports {
				t.Errorf("reports = %+v, esperava %d", reports, tc.wantReports)
			}
			got, err := os.ReadFile(conf)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if string(got) != tc.after {
				t.Errorf("frr.conf = %q, esperava %q", got, tc.after)
			}
		})
	}
}

// repoAgentMD reads a Subagente's AGENT.md from the source tree.
func repoAgentMD(t *testing.T, name string) string {
	t.Helper()
	path, err := paths.RepoFile(filepath.Join("resources/agents", name, "AGENT.md"))
	if err != nil {
		t.Fatalf("locating %s/AGENT.md: %v", name, err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s/AGENT.md: %v", name, err)
	}
	return string(body)
}

// TestChatCarriesTheToolFailureOutOfTheTurn is ticket 18's proof of arrival: a
// refused read has to come out of Chat as a typed record, and the test fails if
// the channel is cut on any of its three floors — the file tool swallowing the
// error, CallFunction dropping the record, or Chat not accumulating it. The two
// directions are in the same test: the same call with the file in place must
// report nothing, because a success that warns blinds the channel.
func TestChatCarriesTheToolFailureOutOfTheTurn(t *testing.T) {
	workspace := t.TempDir()

	chat := func(t *testing.T) ([]tools.Report, []string) {
		t.Helper()
		client, requests := scriptedProvider(t,
			callsOneTool(filetools.READ_TOOL_NAME, `{"filePath":"frr.conf"}`),
			answers("pronto"))
		ag := &Agent{Client: *client, ModelName: "modelo", availableTools: tools.Tools, workspace: workspace}

		_, failures, _, err := ag.Chat(context.Background(), nil, "leia o frr.conf")
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

// TestChatRegistraAAtivacaoEntregue prova o terceiro retorno do Chat (ticket 32
// do mapa da UI): o nome da skill que o modelo ordenou ativar sobe para o turno
// — e sobe só quando o corpo foi entregue. O provedor de mentira responde uma
// chamada de use_skill e depois a resposta final; a tool está oferecida, então o
// corpo carrega e o nome aparece. O segundo caso é o mesmo turno com a tool
// fora da lista: o callTool recusa, o modelo nunca lê a regra, e registrar
// "ativou" ali seria a mentira que esta linha existe para não contar.
func TestChatRegistraAAtivacaoEntregue(t *testing.T) {
	provider := func(t *testing.T) *httptest.Server {
		var calls int
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			if calls == 1 {
				io.WriteString(w, `{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"modelo",
					"output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"use_skill","arguments":"{\"skill_name\": \"mermaid\"}","status":"completed"}],
					"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
				return
			}
			io.WriteString(w, `{"id":"resp_2","object":"response","created_at":0,"status":"completed","model":"modelo",
				"output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"desenhado","annotations":[]}]}],
				"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
		}))
	}

	useSkill := responses.ToolUnionParam{OfFunction: &responses.FunctionToolParam{Name: skills.ToolUseSkill}}

	for _, tc := range []struct {
		name  string
		tools []responses.ToolUnionParam
		want  []string
	}{
		{name: "tool oferecida: o corpo carrega e o nome sobe", tools: []responses.ToolUnionParam{useSkill}, want: []string{"mermaid"}},
		{name: "tool não oferecida: recusada, nada sobe", tools: nil, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := provider(t)
			defer server.Close()

			a := &Agent{
				Client:         openai.NewClient(option.WithBaseURL(server.URL), option.WithAPIKey("teste")),
				ModelName:      "modelo",
				systemPrompt:   "PROMPT",
				availableTools: tc.tools,
			}

			_, _, activated, err := a.Chat(t.Context(), nil, "desenhe a topologia")
			if err != nil {
				t.Fatalf("Chat devolveu erro: %v", err)
			}
			if !slices.Equal(activated, tc.want) {
				t.Errorf("skills ativadas = %v, esperava %v", activated, tc.want)
			}
		})
	}
}
