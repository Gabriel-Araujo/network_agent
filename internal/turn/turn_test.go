package turn

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// TestShouldSearch pins the skip rule: search when the RAG classification
// differs from the previous turn, skip when it is equal (exact map equality,
// empty values included), and always search with no previous turn (first turn
// or zeroed comparison) or a failed classification.
func TestShouldSearch(t *testing.T) {
	empty := map[string]string{"protocol": "", "daemon": "", "chunk_type": ""}
	full := map[string]string{"protocol": "bgp", "daemon": "bgpd", "chunk_type": "concept"}

	cases := []struct {
		name              string
		current, previous map[string]string
		want              bool
	}{
		{"first turn always searches", full, nil, true},
		{"equal classification skips", full, full, false},
		{"exact equality includes empty", empty, map[string]string{"protocol": "", "daemon": "", "chunk_type": ""}, false},
		{"different classification searches", full, empty, true},
		{"failed classification searches", nil, full, true},
		{"failure with no previous searches", nil, nil, true},
		{"zeroed comparison searches", full, nil, true},
	}
	for _, c := range cases {
		if got := shouldSearch(c.current, c.previous); got != c.want {
			t.Errorf("%s: shouldSearch = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestContextJSON: the <recovered_context> block gets the []rag.Result JSON
// exactly as the mirror saves it.
func TestContextJSON(t *testing.T) {
	if got := contextJSON(nil); got != "[]" {
		t.Errorf("contextJSON(nil) = %q, want %q", got, "[]")
	}
}

// fakeLLM is a fake OpenAI-compatible provider: /chat/completions answers the
// canned classification JSON (or fails when failClassification is set), and
// /responses answers a canned reply while recording the request body.
//
// It tells the turn's two kinds of classification call apart by the lists that
// ride them: the single-list re-ask is the only call without a filter list, so
// it gets the reask answer (or failReask) and its own counter — which is what
// pins "the happy path pays no extra call".
//
// The replies queue serves /responses one canned reply per call, in order,
// which is what lets a test drive the diagram re-ask (first reply broken,
// second one fixed). Empty queue answers "ok", so a test that does not care
// about the reply text sets nothing. failReplyFrom is the 1-based /responses
// call from which the provider starts failing (0 = never), which is how a test
// lets the turn's own call through and breaks only the re-ask. usage is what
// the provider reports per call.
type fakeLLM struct {
	classification     string
	reask              string
	failClassification bool
	failReask          bool
	classifications    int
	reasks             int
	lastResponseBody   []byte
	replies            []string
	failReplyFrom      int
	usage              int64
	responseCalls      int
}

// nextReply pops the next canned reply, repeating the last one once the queue
// runs out.
func (f *fakeLLM) nextReply() string {
	f.responseCalls++
	switch {
	case len(f.replies) == 0:
		return "ok"
	case f.responseCalls <= len(f.replies):
		return f.replies[f.responseCalls-1]
	default:
		return f.replies[len(f.replies)-1]
	}
}

func (f *fakeLLM) start(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat/completions":
			body, _ := io.ReadAll(r.Body)
			reask := !bytes.Contains(body, []byte(`"protocol"`))
			answer := f.classification
			if reask {
				f.reasks++
				answer = f.reask
			} else {
				f.classifications++
			}
			if f.failClassification || (reask && f.failReask) {
				http.Error(w, `{"error":{"message":"boom"}}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"chat_1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":`+
				strconv.Quote(answer)+`},"finish_reason":"stop"}]}`)
		case "/responses":
			body, _ := io.ReadAll(r.Body)
			f.lastResponseBody = body
			if f.failReplyFrom > 0 && f.responseCalls+1 >= f.failReplyFrom {
				f.responseCalls++
				http.Error(w, `{"error":{"message":"boom"}}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"resp_1","object":"response","created_at":0,"status":"completed","model":"test-model",
				"output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":`+
				strconv.Quote(f.nextReply())+`,"annotations":[]}]}],
				"usage":{"input_tokens":`+strconv.FormatInt(f.usage, 10)+`,"input_tokens_details":{"cached_tokens":0},
				"output_tokens":0,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":`+strconv.FormatInt(f.usage, 10)+`},
				"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

// newFakeAgent wires an Agent to the fake provider.
func newFakeAgent(t *testing.T, f *fakeLLM) *agentapi.Agent {
	t.Helper()
	ts := f.start(t)
	client := openai.NewClient(option.WithBaseURL(ts.URL), option.WithAPIKey("test-key"))
	return agentapi.Init(client, "", "", "test-model", 0, tools.GetSkillAndTools(), "generic prompt", t.TempDir())
}

// systemFromBody extracts the system prompt sent in a recorded /responses
// request body.
func systemFromBody(t *testing.T, body []byte) string {
	t.Helper()
	var payload struct {
		Input []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"input"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unreadable request body: %v", err)
	}
	for _, item := range payload.Input {
		if item.Role != "system" {
			continue
		}
		var text string
		if err := json.Unmarshal(item.Content, &text); err == nil {
			return text
		}
		var parts []struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(item.Content, &parts); err == nil {
			var b strings.Builder
			for _, p := range parts {
				b.WriteString(p.Text)
			}
			return b.String()
		}
	}
	t.Fatalf("no system message in the request body: %s", body)
	return ""
}

// stubRetrieval replaces the retrieval seam for the test and counts the calls.
func stubRetrieval(t *testing.T, results []rag.Result, err error) *int {
	t.Helper()
	calls := 0
	previous := retrieval
	retrieval = func(context.Context, *agentapi.Agent, string, rag.Filters) ([]rag.Result, string, error) {
		calls++
		return results, "mirror.json", err
	}
	t.Cleanup(func() { retrieval = previous })
	return &calls
}

// TestAgentForTurnPicksTheSubagenteByValue: the generic classification's
// subagent value routes the turn — the Subagente answers with its own AGENT.md
// as the system prompt and file + skills tools, without the subagent tool.
func TestAgentForTurnPicksTheSubagenteByValue(t *testing.T) {
	fake := &fakeLLM{classification: `{"protocol":"","daemon":"","chunk_type":"","subagent":"modifier"}`}
	base := newFakeAgent(t, fake)

	generic, err := classifier.Do(context.Background(), "edit frr.conf", rag.Lists(), false, base)
	if err != nil {
		t.Fatalf("generic classification failed: %v", err)
	}
	if generic[rag.SubagentListName] != "modifier" {
		t.Fatalf("classification %q, want %q", generic[rag.SubagentListName], "modifier")
	}
	turnAgent, subagent, warning := agentForTurn(base, generic, err)
	if warning != "" {
		t.Fatalf("warning %q, want none for a routed turn", warning)
	}
	if turnAgent == base {
		t.Fatal("the turn was answered by the generic agent, not by the Subagente")
	}
	if subagent != "modifier" {
		t.Fatalf("reported Subagente %q, want %q", subagent, "modifier")
	}

	if _, err := turnAgent.Chat(context.Background(), nil, "hello"); err != nil {
		t.Fatalf("the turn chat failed: %v", err)
	}
	body := fake.lastResponseBody
	if len(body) == 0 {
		t.Fatal("the fake provider recorded no request body")
	}

	configPath, err := paths.RepoFile("resources/agents/modifier/AGENT.md")
	if err != nil {
		t.Fatalf("resolving the modifier config: %v", err)
	}
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading the modifier config: %v", err)
	}
	if got := systemFromBody(t, body); got != string(config) {
		t.Error("the turn's system prompt is not the modifier AGENT.md mounted whole")
	}
	if bytes.Contains(body, []byte(`"sub_agent"`)) {
		t.Error("the dispatched Subagente carries the subagent tool (nested routing)")
	}
	// read_skill_file and the file tools, and no use_skill: ticket 31 cut
	// the skill catalog from the dispatched Subagente's toolset whole
	// (.agent/roteamento/issues/31-skills-do-pipeline-fora-do-catalogo-despachado.md).
	if !bytes.Contains(body, []byte(`"read_skill_file"`)) || !bytes.Contains(body, []byte(`"ReadFile"`)) {
		t.Error("the dispatched Subagente is missing file/skills tools")
	}
	if bytes.Contains(body, []byte(`"use_skill"`)) {
		t.Error("the dispatched Subagente carries the skill catalog (use_skill), cut by ticket 31")
	}
}

// TestAgentForTurnEmptyFallsBackToGeneric: an empty subagent value answers
// with the generic agent, with a warning — the turn never dies.
func TestAgentForTurnEmptyFallsBackToGeneric(t *testing.T) {
	fake := &fakeLLM{classification: `{"protocol":"","daemon":"","chunk_type":"","subagent":""}`}
	base := newFakeAgent(t, fake)

	generic, err := classifier.Do(context.Background(), "hello", rag.Lists(), false, base)
	if err != nil {
		t.Fatalf("generic classification failed: %v", err)
	}
	turnAgent, subagent, warning := agentForTurn(base, generic, err)
	if turnAgent != base {
		t.Error("an empty subagent value did not fall back to the generic agent")
	}
	if subagent != "" {
		t.Errorf("reported Subagente %q, want none for the generic agent", subagent)
	}
	if warning == "" {
		t.Error("the fallback did not warn")
	}
}

// TestAgentForTurnClassificationErrorFallsBackWithWarning: a failed generic
// classification answers with the generic agent, with the error in the
// warning — the turn never dies.
func TestAgentForTurnClassificationErrorFallsBackWithWarning(t *testing.T) {
	fake := &fakeLLM{failClassification: true}
	base := newFakeAgent(t, fake)

	generic, err := classifier.Do(context.Background(), "hello", rag.Lists(), false, base)
	if err == nil {
		t.Fatal("expected the generic classification to fail")
	}
	turnAgent, _, warning := agentForTurn(base, generic, err)
	if turnAgent != base {
		t.Error("a failed classification did not fall back to the generic agent")
	}
	if !strings.Contains(warning, "generic classification failed") || !strings.Contains(warning, err.Error()) {
		t.Errorf("warning %q, want the classification error in it", warning)
	}
}

// TestAgentForTurnMissingConfigFallsBackWithExplicitError: a Subagente whose
// folder or AGENT.md is missing degrades to the generic agent with the
// explicit loader error in the warning — never silent.
func TestAgentForTurnMissingConfigFallsBackWithExplicitError(t *testing.T) {
	base := &agentapi.Agent{}
	generic := map[string]string{rag.SubagentListName: "ghost"}

	turnAgent, _, warning := agentForTurn(base, generic, nil)
	if turnAgent != base {
		t.Error("a missing configuration did not fall back to the generic agent")
	}
	if !strings.Contains(warning, "ghost") || !strings.Contains(warning, "not found") {
		t.Errorf("warning %q, want the explicit configuration error in it", warning)
	}
}

// TestRunComposesTheTurn proves the whole composition in one call: the generic
// classification routes to the Subagente, the RAG classification feeds the
// filters, the retrieval lands in the <recovered_context> block, and the
// metadata reports it all. The second turn with the same classification skips
// the retrieval and re-injects the same context.
func TestRunComposesTheTurn(t *testing.T) {
	fake := &fakeLLM{classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"concept","subagent":"modifier"}`}
	base := newFakeAgent(t, fake)
	results := []rag.Result{{Query: "q", Protocol: "bgp", Daemon: "bgpd", ChunkType: "concept"}}
	calls := stubRetrieval(t, results, nil)

	out, state, err := Run(context.Background(), base, nil, "edit frr.conf", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if *calls != 1 {
		t.Fatalf("retrieval calls = %d, want 1", *calls)
	}
	if out.Text != "ok" {
		t.Errorf("Text = %q, want %q", out.Text, "ok")
	}
	if out.Subagent != "modifier" {
		t.Errorf("Subagent = %q, want %q", out.Subagent, "modifier")
	}
	if out.Generic[rag.SubagentListName] != "modifier" {
		t.Errorf("Generic = %v, want the subagent value in it", out.Generic)
	}
	wantFilters := rag.Filters{Protocol: "bgp", Daemon: "bgpd", ChunkType: "concept"}
	if out.Filters != wantFilters {
		t.Errorf("Filters = %+v, want %+v", out.Filters, wantFilters)
	}
	if out.RecoveredContext != contextJSON(results) {
		t.Errorf("RecoveredContext = %q, want the retrieval results JSON", out.RecoveredContext)
	}
	if !strings.Contains(out.Prompt, "<recovered_context>\n"+out.RecoveredContext+"\n</recovered_context>") {
		t.Errorf("Prompt = %q, want the recovered_context block in it", out.Prompt)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", out.Warnings)
	}
	if got := systemFromBody(t, fake.lastResponseBody); !strings.Contains(got, "Modifier") {
		t.Errorf("system prompt = %q, want the modifier AGENT.md", got)
	}

	// Second turn, same classification: the skip rule keeps the retrieval
	// at one call and the context is the last real one.
	out2, _, err := Run(context.Background(), base, nil, "now write it up", state)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if *calls != 1 {
		t.Errorf("retrieval calls = %d after the skipped turn, want 1", *calls)
	}
	if out2.RecoveredContext != out.RecoveredContext {
		t.Errorf("RecoveredContext = %q, want the previous turn's %q", out2.RecoveredContext, out.RecoveredContext)
	}
}

// TestRunRetrievalFailureWarnsAndKeepsTheTurnAlive: a failed retrieval is a
// good turn without context, warned — and the skipped turn after it re-injects
// nothing, because there never was a real retrieval.
func TestRunRetrievalFailureWarnsAndKeepsTheTurnAlive(t *testing.T) {
	fake := &fakeLLM{classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"concept","subagent":"modifier"}`}
	base := newFakeAgent(t, fake)
	calls := stubRetrieval(t, nil, errFake("db down"))

	out, state, err := Run(context.Background(), base, nil, "edit frr.conf", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Text != "ok" {
		t.Errorf("Text = %q, want the turn to answer anyway", out.Text)
	}
	if out.RecoveredContext != "" {
		t.Errorf("RecoveredContext = %q, want none", out.RecoveredContext)
	}
	if strings.Contains(out.Prompt, "<recovered_context>") {
		t.Errorf("Prompt = %q, want no block after a failed retrieval", out.Prompt)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "retrieval failed") {
		t.Errorf("Warnings = %v, want the retrieval failure warned", out.Warnings)
	}

	// Same classification again: skipped retrieval, and a context that never
	// existed stays absent ("" — never an empty block).
	out2, _, err := Run(context.Background(), base, nil, "and now?", state)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if *calls != 1 {
		t.Errorf("retrieval calls = %d after the skipped turn, want 1", *calls)
	}
	if out2.RecoveredContext != "" || strings.Contains(out2.Prompt, "<recovered_context>") {
		t.Errorf("RecoveredContext = %q, Prompt = %q, want no context at all", out2.RecoveredContext, out2.Prompt)
	}
}

// TestRunFailedClassificationDegradesWithWarnings: both classification calls
// failing still answers with the generic agent — generic classification
// warning, empty filters, and the search still runs (the comparison zeroes).
func TestRunFailedClassificationDegradesWithWarnings(t *testing.T) {
	fake := &fakeLLM{failClassification: true}
	base := newFakeAgent(t, fake)
	calls := stubRetrieval(t, []rag.Result{}, nil)

	out, _, err := Run(context.Background(), base, nil, "hello", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Text != "ok" {
		t.Errorf("Text = %q, want the turn to answer anyway", out.Text)
	}
	if out.Generic != nil {
		t.Errorf("Generic = %v, want nil for a failed classification", out.Generic)
	}
	if out.Subagent != "" {
		t.Errorf("Subagent = %q, want the generic agent", out.Subagent)
	}
	if out.Filters != (rag.Filters{}) {
		t.Errorf("Filters = %+v, want empty", out.Filters)
	}
	if *calls != 1 {
		t.Errorf("retrieval calls = %d, want 1 (the comparison zeroes)", *calls)
	}
	if len(out.Warnings) != 2 {
		t.Errorf("Warnings = %v, want the generic and RAG failures warned", out.Warnings)
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
