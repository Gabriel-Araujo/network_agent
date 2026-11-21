package turn

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
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
// It tells the turn's two classification calls apart by the lists that ride
// them: the routing call is the only one without a filter list, so it gets its
// own counter and, when set, its own answer and failure. classification serves
// both by default — the classifier ignores keys outside the lists it was
// given, so one canned object with all four keys answers both questions, and
// routing/failRouting are for the tests that need the two calls to disagree.
//
// The mutex is not decoration: the turn fires the two classification calls
// concurrently (route (b)), so the handler mutating these counters is two
// goroutines deep.
//
// The replies queue serves /responses one canned reply per call, in order,
// which is what lets a test drive the diagram re-ask (first reply broken,
// second one fixed). Empty queue answers "ok", so a test that does not care
// about the reply text sets nothing. failReplyFrom is the 1-based /responses
// call from which the provider starts failing (0 = never), which is how a test
// lets the turn's own call through and breaks only the re-ask. usage is what
// the provider reports per call.
type fakeLLM struct {
	mu                 sync.Mutex
	classification     string
	routing            string
	failClassification bool
	failRouting        bool
	classifications    int
	routings           int
	lastResponseBody   []byte
	replies            []string
	failReplyFrom      int
	usage              int64
	responseCalls      int

	// toolCall, when set, makes the FIRST /responses call answer a ReadFile
	// of that path instead of text — the only way a real tool failure
	// reaches the turn through the fake. It is served once and does not
	// consume a canned reply, so the replies queue keeps meaning "one per
	// text answer".
	toolCall       string
	toolCallServed bool
	// skillCall, when set, makes that first call a use_skill of this skill
	// instead — how a test sees what the activation hands the Subagente.
	skillCall string
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
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/chat/completions":
			body, _ := io.ReadAll(r.Body)
			routing := !bytes.Contains(body, []byte(`"protocol"`))
			answer := f.classification
			if routing {
				f.routings++
				answer = cmp.Or(f.routing, f.classification)
			} else {
				f.classifications++
			}
			if f.failClassification || (routing && f.failRouting) {
				http.Error(w, `{"error":{"message":"boom"}}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"chat_1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":`+
				strconv.Quote(answer)+`},"finish_reason":"stop"}]}`)
		case "/responses":
			body, _ := io.ReadAll(r.Body)
			f.lastResponseBody = body
			if (f.toolCall != "" || f.skillCall != "") && !f.toolCallServed {
				f.toolCallServed = true
				name, args := filetools.READ_TOOL_NAME, `{"filePath":"`+f.toolCall+`"}`
				if f.skillCall != "" {
					name, args = skills.ToolUseSkill, `{"skill_name":"`+f.skillCall+`"}`
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"id":"resp_tool","object":"response","created_at":0,"status":"completed","model":"test-model",
					"output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"`+name+`","arguments":`+
					strconv.Quote(args)+`,"status":"completed"}],
					"parallel_tool_calls":false,"tool_choice":"auto","tools":[]}`)
				return
			}
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
	return newFakeAgentIn(t, f, t.TempDir())
}

// newFakeAgentIn is newFakeAgent with the Workspace named by the caller: the
// Aviso of a tool failure says which folder the tool resolved against, and a
// t.TempDir() the test cannot see is a root it cannot assert.
//
// No toolset and no system prompt: this is the turn's **entry point**, and it
// never answers, so the two things a respondent needs are the two it does not
// carry. Every Chat call of a turn goes through a Subagente, whose toolset and
// prompt ForSubagent mounts.
func newFakeAgentIn(t *testing.T, f *fakeLLM, workspace string) *agentapi.Agent {
	t.Helper()
	ts := f.start(t)
	client := openai.NewClient(option.WithBaseURL(ts.URL), option.WithAPIKey("test-key"))
	return agentapi.Init(client, "", "", "test-model", 0, nil, "", workspace)
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

// TestAgentForTurnPicksTheSubagenteByValue: the routing classification's
// subagent value routes the turn — the Subagente answers with its own AGENT.md
// as the system prompt and file + skills tools, without the subagent tool.
func TestAgentForTurnPicksTheSubagenteByValue(t *testing.T) {
	fake := &fakeLLM{routing: `{"subagent":"modifier"}`}
	base := newFakeAgent(t, fake)

	routing, err := classifier.Do(context.Background(), "edit frr.conf", []classifier.List{rag.SubagentList()}, false, base)
	if err != nil {
		t.Fatalf("routing classification failed: %v", err)
	}
	if routing[rag.SubagentListName] != "modifier" {
		t.Fatalf("classification %q, want %q", routing[rag.SubagentListName], "modifier")
	}
	turnAgent, subagent, warning, err := agentForTurn(base, routing, nil, rag.Filters{})
	if err != nil {
		t.Fatalf("agentForTurn: %v", err)
	}
	if warning != "" {
		t.Fatalf("warning %q, want none for a routed turn", warning)
	}
	if turnAgent == base {
		t.Fatal("the turn was answered by the entry point, not by the Subagente")
	}
	if subagent != "modifier" {
		t.Fatalf("reported Subagente %q, want %q", subagent, "modifier")
	}

	if _, _, _, err := turnAgent.Chat(context.Background(), nil, "hello"); err != nil {
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
	// read_skill_file and the file tools, and no pipeline skill: ticket 31 cut
	// the full skill catalog from the dispatched Subagente's toolset
	// (.agent/roteamento/issues/31-skills-do-pipeline-fora-do-catalogo-despachado.md);
	// the Modifier's use_skill is narrowed to frr-file-builder.
	if !bytes.Contains(body, []byte(`"read_skill_file"`)) || !bytes.Contains(body, []byte(`"ReadFile"`)) {
		t.Error("the dispatched Subagente is missing file/skills tools")
	}
	for _, pipeline := range []string{"frr-intent-analyzer", "frr-rag-retriever"} {
		if bytes.Contains(body, []byte(pipeline)) {
			t.Errorf("the dispatched Subagente carries the skill catalog (%s), cut by ticket 31", pipeline)
		}
	}
}

// TestAgentForTurnNeverAnswersWithTheEntryPoint is the invariant of this
// ticket: no routing outcome — named, empty, or failed — comes back as the
// agent that was handed in. Every one of them mounts a Subagente, and the one
// for "no Subagente" is rag.DefaultSubagent.
func TestAgentForTurnNeverAnswersWithTheEntryPoint(t *testing.T) {
	base := newFakeAgent(t, &fakeLLM{})
	cases := []struct {
		name         string
		routing      map[string]string
		routingErr   error
		filters      rag.Filters
		wantSubagent string
	}{
		{"named", map[string]string{rag.SubagentListName: "modifier"}, nil, rag.Filters{}, "modifier"},
		{"empty, no filter", map[string]string{rag.SubagentListName: ""}, nil, rag.Filters{}, rag.DefaultSubagent},
		{"empty, filter filled", map[string]string{rag.SubagentListName: ""}, nil, rag.Filters{Protocol: "bgp"}, rag.DefaultSubagent},
		{"failed call", nil, errFake("boom"), rag.Filters{}, rag.DefaultSubagent},
		{"config did not load", map[string]string{rag.SubagentListName: "ghost"}, nil, rag.Filters{}, rag.DefaultSubagent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			turnAgent, subagent, _, err := agentForTurn(base, c.routing, c.routingErr, c.filters)
			if err != nil {
				t.Fatalf("agentForTurn: %v", err)
			}
			if turnAgent == base || turnAgent == nil {
				t.Error("the turn came back answered by the entry point")
			}
			if subagent != c.wantSubagent {
				t.Errorf("reported Subagente %q, want %q", subagent, c.wantSubagent)
			}
		})
	}
}

// TestRoutedNameSplitsAvisoFromSilence pins decision 6 of ticket 02: the Aviso
// exists exactly where the app has evidence that the empty routing is wrong,
// which after the re-ask's death is an explicit test over the filters that
// actually filtered — three Avisos and one silence.
func TestRoutedNameSplitsAvisoFromSilence(t *testing.T) {
	cases := []struct {
		name        string
		routing     map[string]string
		routingErr  error
		filters     rag.Filters
		wantName    string
		wantAviso   string
		wantSilence bool
	}{
		{
			name: "named Subagente is silent", routing: map[string]string{rag.SubagentListName: "builder"},
			filters: rag.Filters{Protocol: "bgp"}, wantName: "builder", wantSilence: true,
		},
		{
			name: "empty with no filter is the contract, silent", routing: map[string]string{rag.SubagentListName: ""},
			wantName: rag.DefaultSubagent, wantSilence: true,
		},
		{
			name: "empty with a filter filled warns", routing: map[string]string{rag.SubagentListName: ""},
			filters: rag.Filters{ChunkType: "concept"}, wantName: rag.DefaultSubagent, wantAviso: "=vazio",
		},
		{
			name: "failed routing call warns", routingErr: errFake("boom"),
			wantName: rag.DefaultSubagent, wantAviso: "boom",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			name, aviso := routedName(c.routing, c.routingErr, c.filters)
			if name != c.wantName {
				t.Errorf("name = %q, want %q", name, c.wantName)
			}
			if c.wantSilence {
				if aviso != "" {
					t.Errorf("Aviso = %q, want silence", aviso)
				}
				return
			}
			if !strings.Contains(aviso, c.wantAviso) {
				t.Errorf("Aviso = %q, want %q in it", aviso, c.wantAviso)
			}
			if !strings.Contains(aviso, rag.DefaultSubagent) {
				t.Errorf("Aviso = %q, want the default named in it", aviso)
			}
		})
	}
}

// TestAgentForTurnMissingConfigAnswersWithTheDefault: a Subagente whose folder
// or AGENT.md is missing is degradation like any other — the default answers
// for it, and the Aviso names both plus the loader's own error. Never silent.
func TestAgentForTurnMissingConfigAnswersWithTheDefault(t *testing.T) {
	base := newFakeAgent(t, &fakeLLM{})
	routing := map[string]string{rag.SubagentListName: "ghost"}

	_, subagent, warning, err := agentForTurn(base, routing, nil, rag.Filters{})
	if err != nil {
		t.Fatalf("agentForTurn: %v", err)
	}
	if subagent != rag.DefaultSubagent {
		t.Errorf("reported Subagente %q, want the default %q", subagent, rag.DefaultSubagent)
	}
	if !strings.Contains(warning, "ghost") || !strings.Contains(warning, "not found") {
		t.Errorf("warning %q, want the explicit configuration error in it", warning)
	}
	if !strings.Contains(warning, rag.DefaultSubagent) {
		t.Errorf("warning %q, want the default named as the one answering", warning)
	}
}

// TestRunDiesWhenTheDefaultDoesNotLoad is the last resort (decision 4 of
// ticket 01): with the entry point out of the answering role, a default that
// does not load leaves nobody to answer, and that is the one degradation that
// kills the turn. The error is visible and names the default; the Outcome still
// carries what the turn learned before it.
//
// The broken material root is the whole fault: a home whose parent is a file,
// so seeding the mirror fails and ForSubagent cannot resolve any name.
func TestRunDiesWhenTheDefaultDoesNotLoad(t *testing.T) {
	fake := &fakeLLM{classification: `{"protocol":"bgp","daemon":"","chunk_type":"","subagent":"teacher"}`}
	base := newFakeAgent(t, fake)
	stubRetrieval(t, []rag.Result{}, nil)

	notADir := filepath.Join(t.TempDir(), "material")
	if err := os.WriteFile(notADir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("seeding the broken material root: %v", err)
	}
	t.Cleanup(config.SetDir(filepath.Join(notADir, "home")))

	out, _, err := Run(context.Background(), base, nil, "explain bgp", State{})
	if err == nil {
		t.Fatal("Run returned no error: a turn with no one to answer it must die")
	}
	if !strings.Contains(err.Error(), rag.DefaultSubagent) {
		t.Errorf("err = %v, want the default Subagente named in it", err)
	}
	if out.Text != "" {
		t.Errorf("Text = %q, want none: nobody answered", out.Text)
	}
	if out.Filters != (rag.Filters{Protocol: "bgp"}) {
		t.Errorf("Filters = %+v, want what the turn learned before it died", out.Filters)
	}
}

// TestRunComposesTheTurn proves the whole composition in one call: the routing
// classification picks the Subagente, the RAG classification feeds the
// filters, the retrieval lands in the <recovered_context> block, and the
// metadata reports it all. The Subagente's activation carries the pack of the
// classified protocol and the reach of that retrieval. The second turn with the
// same classification skips the retrieval and re-injects the same context.
func TestRunComposesTheTurn(t *testing.T) {
	fake := &fakeLLM{
		classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"concept","subagent":"modifier"}`,
		skillCall:      "frr-file-builder",
	}
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
	if fake.routings != 1 || fake.classifications != 1 {
		t.Errorf("routing calls = %d, RAG calls = %d, want one each: two disjoint questions", fake.routings, fake.classifications)
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
	for _, want := range []string{"references/bgp/doctrine.md", "retrieved for bgp only"} {
		if !bytes.Contains(fake.lastResponseBody, []byte(want)) {
			t.Errorf("the activation output lacks %q: the turn did not reach the Subagente's use_skill", want)
		}
	}
	if !slices.Equal(out.SkillsActivated, []string{"frr-file-builder"}) {
		t.Errorf("SkillsActivated = %v, want the activation counted", out.SkillsActivated)
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
// failing still answers — with the default Subagente, one Aviso each, empty
// filters, and the search still running (the comparison zeroes).
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
	if out.Subagent != rag.DefaultSubagent {
		t.Errorf("Subagent = %q, want the default %q", out.Subagent, rag.DefaultSubagent)
	}
	if out.Filters != (rag.Filters{}) {
		t.Errorf("Filters = %+v, want empty", out.Filters)
	}
	if *calls != 1 {
		t.Errorf("retrieval calls = %d, want 1 (the comparison zeroes)", *calls)
	}
	if len(out.Warnings) != 2 {
		t.Errorf("Warnings = %v, want the routing and RAG failures warned", out.Warnings)
	}
}

// TestRunRoutesTheEmptyToTheDefault pins the two branches of the empty value
// end to end, through Run: with a filter filled the turn carries exactly one
// Aviso, with none at all it is silent — and both are answered by the default,
// never by the entry point (decision 6 of ticket 02).
func TestRunRoutesTheEmptyToTheDefault(t *testing.T) {
	cases := []struct {
		name         string
		filters      string
		wantWarnings int
	}{
		{"empty with a filter filled warns", `"protocol":"bgp","daemon":"","chunk_type":""`, 1},
		{"empty with no filter is silent", `"protocol":"","daemon":"","chunk_type":""`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &fakeLLM{
				classification: `{` + c.filters + `}`,
				routing:        `{"subagent":""}`,
			}
			base := newFakeAgent(t, fake)
			stubRetrieval(t, []rag.Result{}, nil)

			out, _, err := Run(context.Background(), base, nil, "bolo de cenoura", State{})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out.Subagent != rag.DefaultSubagent {
				t.Errorf("Subagent = %q, want the default %q", out.Subagent, rag.DefaultSubagent)
			}
			if len(out.Warnings) != c.wantWarnings {
				t.Errorf("Warnings = %v, want %d", out.Warnings, c.wantWarnings)
			}
			if got := systemFromBody(t, fake.lastResponseBody); !strings.Contains(got, "Teacher") {
				t.Error("the turn was not answered by the default Subagente")
			}
		})
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
