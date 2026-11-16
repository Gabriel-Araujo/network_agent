package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/openai/openai-go/v3/packages/respjson"
	"github.com/openai/openai-go/v3/responses"
)

func repairTestApp(t *testing.T) (*App, string) {
	t.Helper()
	a := &App{ctx: t.Context(), store: history.NewFileStore(t.TempDir()), agent: &agent.Agent{}}
	c, err := a.store.Create(history.NewConversation{Title: "test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.store.Append(c.ID, history.Message{ID: "question", Role: history.RoleUser, Status: history.StatusDone, Content: "draw"}, history.Message{ID: "answer", Role: history.RoleAgent, Status: history.StatusDone, Content: "prose\n\n```mermaid\ngraph LR\nA -- ERROR B\n```"})
	if err != nil {
		t.Fatal(err)
	}
	return a, c.ID
}

func TestRepairBridgeLifecycle(t *testing.T) {
	a, id := repairTestApp(t)
	old := callMermaidRepair
	defer func() { callMermaidRepair = old }()
	calls := 0
	callMermaidRepair = func(_ context.Context, _ *agent.Agent, failures []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
		calls++
		return agent.MermaidRepairResult{Replacements: []agent.MermaidReplacement{{Index: failures[0].Index, Source: "graph LR\nA --> B"}}}, nil
	}
	failures := []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}}
	start, err := a.StartMermaidRepair(id, "answer", failures)
	if err != nil || start.AttemptID == "" || calls != 1 {
		t.Fatalf("start=%+v calls=%d err=%v", start, calls, err)
	}
	before, _ := a.store.Get(id)
	if got := history.ProjectTranscript(before.Messages); strings.Contains(got[1].Content, "A --> B") {
		t.Fatal("candidate leaked into transcript before confirmation")
	}
	if _, err := a.StartMermaidRepair(id, "answer", failures); err == nil || calls != 1 {
		t.Fatal("duplicate attempt was allowed")
	}
	view, err := a.FinishMermaidRepair(id, start.AttemptID, true)
	if err != nil || len(view.Messages) != 2 || view.Messages[1].MermaidRepair.State != history.MermaidRepairAccepted {
		t.Fatalf("view=%+v err=%v", view, err)
	}
	stored, _ := a.store.Get(id)
	if stored.Messages[1].Content != "prose\n\n```mermaid\ngraph LR\nA -- ERROR B\n```" {
		t.Fatal("original was overwritten")
	}
	if got := history.ProjectTranscript(stored.Messages); len(got) != 2 || !strings.Contains(got[1].Content, "A --> B") {
		t.Fatalf("transcript=%+v", got)
	}
	if _, err := a.FinishMermaidRepair(id, start.AttemptID, true); err == nil {
		t.Fatal("duplicate confirmation was allowed")
	}
}

func TestRepairBridgeFailureAndStaleResponse(t *testing.T) {
	a, id := repairTestApp(t)
	old := callMermaidRepair
	defer func() { callMermaidRepair = old }()
	callMermaidRepair = func(_ context.Context, _ *agent.Agent, _ []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
		return agent.MermaidRepairResult{}, errors.New("provider failed")
	}
	failures := []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}}
	if _, err := a.StartMermaidRepair(id, "answer", failures); err == nil {
		t.Fatal("expected provider error")
	}
	stored, _ := a.store.Get(id)
	view := history.ProjectConversation(stored)
	if view.Messages[1].MermaidRepair.State != history.MermaidRepairFailed || view.Messages[1].Content != stored.Messages[1].Content {
		t.Fatalf("view=%+v", view.Messages)
	}
	if _, err := a.StartMermaidRepair(id, "answer", failures); err == nil {
		t.Fatal("failed attempt was allowed to repeat")
	}
	if _, err := a.FinishMermaidRepair(id, "wrong", true); err == nil {
		t.Fatal("unknown attempt was accepted")
	}
}

func TestRepairBridgeRejectsStaleAndInterrupted(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(map[bool]string{false: "turn advanced", true: "restart"}[interrupted], func(t *testing.T) {
			a, id := repairTestApp(t)
			old := callMermaidRepair
			defer func() { callMermaidRepair = old }()
			callMermaidRepair = func(_ context.Context, _ *agent.Agent, _ []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
				return agent.MermaidRepairResult{Replacements: []agent.MermaidReplacement{{Index: 1, Source: "graph LR\nA-->B"}}}, nil
			}
			start, err := a.StartMermaidRepair(id, "answer", []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}})
			if err != nil {
				t.Fatal(err)
			}
			if interrupted {
				a = &App{ctx: t.Context(), store: a.store, agent: &agent.Agent{}}
			} else if _, err := a.store.Append(id, history.Message{ID: "next", Role: history.RoleUser, Status: history.StatusDone, Content: "another turn"}); err != nil {
				t.Fatal(err)
			}
			if _, err := a.FinishMermaidRepair(id, start.AttemptID, true); err == nil {
				t.Fatal("accepted obsolete or interrupted candidate")
			}
			if _, err := a.StartMermaidRepair(id, "answer", []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}}); err == nil {
				t.Fatal("repeated repair after restart or new turn")
			}
			stored, _ := a.store.Get(id)
			if strings.Contains(history.ProjectTranscript(stored.Messages)[1].Content, "A-->B") {
				t.Fatal("obsolete candidate leaked into transcript")
			}
		})
	}
}

func TestRepairBridgeChecksFailureSourcesAndBusy(t *testing.T) {
	a, id := repairTestApp(t)
	for _, failures := range [][]history.MermaidFailure{
		{{Index: 1, Source: "different", Error: "bad"}},
		{{Index: 2, Source: "graph LR\nA -- ERROR B", Error: "bad"}},
		{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "bad"}, {Index: 1, Source: "graph LR\nA -- ERROR B", Error: "bad"}},
	} {
		if _, err := a.StartMermaidRepair(id, "answer", failures); err == nil {
			t.Fatalf("accepted invalid failures %+v", failures)
		}
	}
	a.busyMux.Lock()
	defer a.busyMux.Unlock()
	if _, err := a.StartMermaidRepair(id, "answer", nil); err == nil {
		t.Fatal("accepted concurrent repair")
	}
}

func TestRepairBridgeRejectsTurnAdvancingDuringCall(t *testing.T) {
	a, id := repairTestApp(t)
	old := callMermaidRepair
	defer func() { callMermaidRepair = old }()
	callMermaidRepair = func(_ context.Context, _ *agent.Agent, _ []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
		if _, err := a.store.Append(id, history.Message{ID: "next", Role: history.RoleUser, Status: history.StatusDone, Content: "new turn"}); err != nil {
			t.Fatal(err)
		}
		return agent.MermaidRepairResult{Replacements: []agent.MermaidReplacement{{Index: 1, Source: "graph LR\nA-->B"}}}, nil
	}
	if _, err := a.StartMermaidRepair(id, "answer", []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}}); err == nil {
		t.Fatal("accepted result after conversation advanced")
	}
	stored, err := a.store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got := history.ProjectTranscript(stored.Messages); strings.Contains(got[1].Content, "A-->B") {
		t.Fatal("obsolete candidate leaked into transcript")
	}
}

// TestRepairBridgeRetryRounds: one attempt, several model rounds. Each retry
// carries the renderer's feedback against the COMPOSED candidate, later rounds
// win per block, the usage accumulates across rounds, and the budget stops at
// maxRepairRounds model calls. The measured shape (2026-11-15): one call fixed
// one defect class per round, so the correction needs the loop and its bounds.
func TestRepairBridgeRetryRounds(t *testing.T) {
	a, id := repairTestApp(t)
	old := callMermaidRepair
	defer func() { callMermaidRepair = old }()
	stillBroken1 := "graph LR\nA --> B %% ainda quebrado 1"
	stillBroken2 := "graph LR\nA --> B %% ainda quebrado 2"
	fixed := "graph LR\nA --> B"
	rounds := []string{stillBroken1, stillBroken2, fixed}
	calls := 0
	callMermaidRepair = func(_ context.Context, _ *agent.Agent, failures []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
		source := rounds[min(calls, len(rounds)-1)]
		calls++
		usage := &responses.ResponseUsage{InputTokens: 5, OutputTokens: 2, TotalTokens: 7}
		usage.JSON.InputTokens = respjson.NewField("5")
		usage.JSON.OutputTokens = respjson.NewField("2")
		usage.JSON.TotalTokens = respjson.NewField("7")
		return agent.MermaidRepairResult{Replacements: []agent.MermaidReplacement{{Index: failures[0].Index, Source: source}}, Usage: usage}, nil
	}
	original := []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}}
	start, err := a.StartMermaidRepair(id, "answer", original)
	if err != nil {
		t.Fatal(err)
	}
	// A retry reports the CANDIDATE's failures, not the original's: the
	// original source no longer matches the composed text and is refused.
	if _, err := a.RetryMermaidRepair(id, start.AttemptID, original); err == nil {
		t.Fatal("retry accepted failures of the original text")
	}
	second, err := a.RetryMermaidRepair(id, start.AttemptID, []history.MermaidFailure{{Index: 1, Source: stillBroken1, Error: "parse error"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Replacements) != 1 || second.Replacements[0].Source != stillBroken2 {
		t.Fatalf("round 2 merged replacements = %+v", second.Replacements)
	}
	third, err := a.RetryMermaidRepair(id, start.AttemptID, []history.MermaidFailure{{Index: 1, Source: stillBroken2, Error: "parse error"}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if len(third.Replacements) != 1 || third.Replacements[0].Source != fixed {
		t.Fatalf("round 3 merged replacements = %+v", third.Replacements)
	}
	stored, _ := a.store.Get(id)
	last := stored.Messages[len(stored.Messages)-1].MermaidRepair
	if last.State != history.MermaidRepairCandidate || len(last.Replacements) != 1 || last.Replacements[0].Source != fixed {
		t.Fatalf("persisted candidate = %+v", last)
	}
	if last.Usage == nil || last.Usage.Input != 15 || last.Usage.Output != 6 || last.Usage.Total != 21 {
		t.Fatalf("accumulated usage = %+v", last.Usage)
	}
	// Budget: maxRepairRounds model calls per attempt, then refused.
	if _, err := a.RetryMermaidRepair(id, start.AttemptID, []history.MermaidFailure{{Index: 1, Source: fixed, Error: "x"}}); err == nil || !strings.Contains(err.Error(), "esgotou") {
		t.Fatalf("budget retry err = %v", err)
	}
	// Identity and terminal state.
	if _, err := a.RetryMermaidRepair(id, "outra-tentativa", original); err == nil {
		t.Fatal("retry with a foreign attempt was allowed")
	}
	view, err := a.FinishMermaidRepair(id, start.AttemptID, true)
	if err != nil || view.Messages[1].MermaidRepair.State != history.MermaidRepairAccepted {
		t.Fatalf("view=%+v err=%v", view, err)
	}
	if got := history.ProjectTranscript(view.Messages); !strings.Contains(got[1].Content, fixed) {
		t.Fatalf("transcript does not carry the fixed block: %q", got[1].Content)
	}
	if _, err := a.RetryMermaidRepair(id, start.AttemptID, original); err == nil {
		t.Fatal("retry after the terminal state was allowed")
	}
}

func TestRepairBridgePersistsReportedUsageOnFailure(t *testing.T) {
	a, id := repairTestApp(t)
	old := callMermaidRepair
	defer func() { callMermaidRepair = old }()
	callMermaidRepair = func(_ context.Context, _ *agent.Agent, _ []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
		usage := &responses.ResponseUsage{InputTokens: 7, OutputTokens: 3, TotalTokens: 10}
		usage.JSON.InputTokens = respjson.NewField("7")
		usage.JSON.OutputTokens = respjson.NewField("3")
		usage.JSON.TotalTokens = respjson.NewField("10")
		return agent.MermaidRepairResult{Usage: usage}, errors.New("invalid candidate")
	}
	_, err := a.StartMermaidRepair(id, "answer", []history.MermaidFailure{{Index: 1, Source: "graph LR\nA -- ERROR B", Error: "invalid edge"}})
	if err == nil {
		t.Fatal("expected failure")
	}
	stored, err := a.store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	usage := history.ProjectConversation(stored).Messages[1].MermaidRepair.Usage
	if usage == nil || usage.Input != 7 || usage.Output != 3 || usage.Total != 10 {
		t.Fatalf("usage = %+v", usage)
	}
}
