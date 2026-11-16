package agent

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func repairResponse(t *testing.T, text, usage string) string {
	t.Helper()
	encoded, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`{"id":"chatcmpl","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":"stop"}]%s}`, encoded, usage)
}

// chatCallsOneTool is a chat completion whose message carries a tool call.
func chatCallsOneTool(name, arguments string) string {
	args, _ := json.Marshal(arguments)
	return `{"id":"chatcmpl","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"` + name + `","arguments":` + string(args) + `}}]},"finish_reason":"tool_calls"}]}`
}

func TestRepairMermaidSuccess(t *testing.T) {
	client, requests := scriptedProvider(t, repairResponse(t, `{"replacements":[{"index":2,"source":"graph LR\n B-->C"},{"index":0,"source":"graph TD\n A-->B"}]}`, `,"usage":{"prompt_tokens":0,"completion_tokens":7}`))
	a := &Agent{Client: *client, ModelName: "modelo", availableTools: tools.Tools, systemPrompt: "PROMPT DO CHAT"}
	failures := []MermaidFailure{{Index: 0, Source: "graph TD\n A-->", Error: "erro 0"}, {Index: 2, Source: "graph LR\n B-->", Error: "erro 2"}}
	result, err := a.RepairMermaid(t.Context(), failures)
	if err != nil {
		t.Fatal(err)
	}
	want := []MermaidReplacement{{Index: 2, Source: "graph LR\n B-->C"}, {Index: 0, Source: "graph TD\n A-->B"}}
	if !reflect.DeepEqual(result.Replacements, want) {
		t.Fatalf("substituições = %+v, want %+v", result.Replacements, want)
	}
	if result.Usage == nil || !result.Usage.JSON.InputTokens.Valid() || result.Usage.InputTokens != 0 || !result.Usage.JSON.OutputTokens.Valid() || result.Usage.OutputTokens != 7 || result.Usage.JSON.TotalTokens.Valid() {
		t.Fatalf("uso parcial = %+v", result.Usage)
	}
	if len(*requests) != 1 {
		t.Fatalf("requisições = %d, want 1", len(*requests))
	}
	body := (*requests)[0]
	if _, ok := body["tools"]; ok {
		t.Fatalf("reparo ofereceu ferramentas: %v", body["tools"])
	}
	format := body["response_format"].(map[string]any)
	envelope := format["json_schema"].(map[string]any)
	if format["type"] != "json_schema" || envelope["strict"] != true || envelope["name"] != "mermaid_repair" {
		t.Errorf("formato estruturado = %v", format)
	}
	encoded, err := json.Marshal(body["messages"])
	if err != nil {
		t.Fatal(err)
	}
	input := string(encoded)
	for _, text := range []string{"erro 0", "erro 2", "graph TD", "graph LR", "Mermaid diagrams"} {
		if !strings.Contains(input, text) {
			t.Errorf("input não contém %q: %s", text, input)
		}
	}
	if strings.Contains(input, "PROMPT DO CHAT") {
		t.Fatalf("reparo carregou o system prompt do agente: %v", body)
	}
}

func TestRepairMermaidRejectsInvalidResponses(t *testing.T) {
	const usage = `,"usage":{"prompt_tokens":11,"completion_tokens":3,"total_tokens":14}`
	tests := []struct{ name, text string }{
		{"json inválido", `não é JSON`},
		{"prosa antes do json", `Aqui está: {"replacements":[]}`},
		{"prosa depois da cerca", "```json\n{\"replacements\":[]}\n```\nfim"},
		{"array com campo extra", `[{"index":0,"source":"A","prose":"B"}]`},
		{"texto adicional", `{"replacements":[{"index":0,"source":"graph TD"}]} lixo`},
		{"objeto ausente", `{}`},
		{"null", `{"replacements":null}`},
		{"índice ausente", `{"replacements":[{"source":"graph TD"}]}`},
		{"fonte ausente", `{"replacements":[{"index":0}]}`},
		{"fonte nula", `{"replacements":[{"index":0,"source":null}]}`},
		{"fonte vazia", `{"replacements":[{"index":0,"source":"  "}]}`},
		{"bloco ausente", `{"replacements":[]}`},
		{"índice duplicado", `{"replacements":[{"index":0,"source":"A"},{"index":0,"source":"B"}]}`},
		{"índice não solicitado", `{"replacements":[{"index":1,"source":"A"}]}`},
		{"campo extra", `{"replacements":[{"index":0,"source":"A","prose":"B"}]}`},
		{"campo duplicado", `{"replacements":[{"index":0,"index":0,"source":"A"}]}`},
		{"cerca com prosa", "{\"replacements\":[{\"index\":0,\"source\":\"graph TD\\n```\\nprosa injetada\"}]}"},
		{"cerca de til", `{"replacements":[{"index":0,"source":"graph TD\n~~~\nprosa injetada"}]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, requests := scriptedProvider(t, repairResponse(t, tc.text, usage))
			result, err := (&Agent{Client: *client, ModelName: "modelo"}).RepairMermaid(t.Context(), []MermaidFailure{{Index: 0, Source: "ruim", Error: "erro"}})
			if err == nil || len(result.Replacements) != 0 {
				t.Fatalf("resultado = %+v, erro = %v", result, err)
			}
			if result.Usage == nil || result.Usage.InputTokens != 11 || result.Usage.TotalTokens != 14 || !result.Usage.JSON.TotalTokens.Valid() {
				t.Fatalf("uso perdido: %+v", result.Usage)
			}
			if len(*requests) != 1 {
				t.Fatalf("requisições = %d", len(*requests))
			}
		})
	}
}

// TestRepairMermaidAcceptsAnEnvelopeDrift: the model may wrap the structured
// payload in shapes the schema does not describe — two measured live on
// 2026-11-15, where each made an automatic repair fail on a block the renderer
// had rejected: a Markdown fence around the JSON, and a bare JSON array instead
// of the object. Both are envelope drift, never content: inside them the
// payload is parsed as strictly as before, and prose before or after it still
// fails (the cases above).
func TestRepairMermaidAcceptsAnEnvelopeDrift(t *testing.T) {
	const inner = `{"replacements":[{"index":0,"source":"graph TD\n A-->B"}]}`
	tests := []struct{ name, text string }{
		{"cerca json", "```json\n" + inner + "\n```"},
		{"cerca sem linguagem", "```\n" + inner + "\n```"},
		{"cerca com espaços", "  ```json\r\n" + inner + "\r\n```  "},
		{"array puro", `[{"index":0,"source":"graph TD\n A-->B"}]`},
		{"cerca com array", "```json\n[{\"index\":0,\"source\":\"graph TD\\n A-->B\"}]\n```"},
		{"objeto solto", `{"index":0,"source":"graph TD\n A-->B"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, requests := scriptedProvider(t, repairResponse(t, tc.text, `,"usage":{"completion_tokens":3}`))
			result, err := (&Agent{Client: *client, ModelName: "modelo"}).RepairMermaid(t.Context(), []MermaidFailure{{Index: 0, Source: "ruim", Error: "erro"}})
			if err != nil {
				t.Fatalf("payload cercado recusado: %v", err)
			}
			if len(result.Replacements) != 1 || result.Replacements[0].Index != 0 || result.Replacements[0].Source != "graph TD\n A-->B" {
				t.Fatalf("substituições = %+v", result.Replacements)
			}
			if len(*requests) != 1 {
				t.Fatalf("requisições = %d", len(*requests))
			}
		})
	}
}

func TestRepairMermaidRejectsToolCallWithoutExecuting(t *testing.T) {
	workspace := t.TempDir()
	client, requests := scriptedProvider(t, chatCallsOneTool("WriteFile", `{"filePath":"invasao.txt","content":"invadido"}`))
	a := &Agent{Client: *client, ModelName: "modelo", availableTools: tools.Tools, workspace: workspace}
	result, err := a.RepairMermaid(t.Context(), []MermaidFailure{{Index: 0, Source: "ruim", Error: "erro"}})
	if err == nil || len(result.Replacements) != 0 || result.Usage != nil {
		t.Fatalf("resultado = %+v, erro = %v", result, err)
	}
	if len(*requests) != 1 {
		t.Fatalf("requisições = %d", len(*requests))
	}
	if _, err := os.Stat(filepath.Join(workspace, "invasao.txt")); !os.IsNotExist(err) {
		t.Fatalf("ferramenta gravou arquivo: %v", err)
	}
}

func TestRepairMermaidRejectsUnexpectedOutput(t *testing.T) {
	const usage = `,"usage":{"completion_tokens":0}`
	tests := []struct{ name, output string }{
		{"tool com texto", `"message":{"role":"assistant","content":"texto","tool_calls":[{"id":"call_1","type":"function","function":{"name":"web_search","arguments":"{}"}}]},"finish_reason":"tool_calls"`},
		{"recusa", `"message":{"role":"assistant","content":"","refusal":"não"},"finish_reason":"stop"`},
		{"sem conteúdo", `"message":{"role":"assistant","content":""},"finish_reason":"stop"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"id":"chatcmpl","object":"chat.completion","choices":[{%s}]%s}`, tc.output, usage)
			client, requests := scriptedProvider(t, body)
			result, err := (&Agent{Client: *client, ModelName: "modelo"}).RepairMermaid(t.Context(), []MermaidFailure{{Index: 0}})
			if err == nil || len(result.Replacements) != 0 || result.Usage == nil || !result.Usage.JSON.OutputTokens.Valid() || result.Usage.OutputTokens != 0 || len(*requests) != 1 {
				t.Fatalf("resultado %+v, erro %v, requisições %d", result, err, len(*requests))
			}
		})
	}
	t.Run("sem escolhas", func(t *testing.T) {
		client, requests := scriptedProvider(t, `{"id":"chatcmpl","object":"chat.completion","choices":[]`+usage+`}`)
		result, err := (&Agent{Client: *client, ModelName: "modelo"}).RepairMermaid(t.Context(), []MermaidFailure{{Index: 0}})
		if err == nil || len(result.Replacements) != 0 || len(*requests) != 1 {
			t.Fatalf("resultado %+v, erro %v, requisições %d", result, err, len(*requests))
		}
	})
}

func TestRepairMermaidInvalidInputAndProviderError(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	a := &Agent{Client: openai.NewClient(option.WithBaseURL(server.URL), option.WithAPIKey("teste")), ModelName: "modelo"}
	for _, failures := range [][]MermaidFailure{nil, {{Index: -1}}, {{Index: 0}, {Index: 0}}} {
		result, err := a.RepairMermaid(t.Context(), failures)
		if err == nil || len(result.Replacements) != 0 || result.Usage != nil || requests != 0 {
			t.Fatalf("entrada %+v: resultado %+v, erro %v, requisições %d", failures, result, err, requests)
		}
	}
	result, err := a.RepairMermaid(t.Context(), []MermaidFailure{{Index: 0, Source: "ruim", Error: "erro"}})
	if err == nil || len(result.Replacements) != 0 || result.Usage != nil || requests != 1 {
		t.Fatalf("erro do provedor: resultado %+v, erro %v, requisições %d", result, err, requests)
	}
}
