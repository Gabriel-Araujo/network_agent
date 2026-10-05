package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/history"
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

	resp, err := a.Chat(context.Background(), nil, "use a ferramenta")
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
