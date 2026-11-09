package turn

import (
	"context"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
)

// the two replies every re-ask test is built from: the broken one is the
// measured mode (parens in a node label, 4 of the 7), the fixed one is the same
// diagram with the label quoted.
const (
	brokenReply = "Here it is:\n\n```mermaid\nflowchart LR\n    C2[Router C2 (ABR)]\n```\n"
	fixedReply  = "Here it is:\n\n```mermaid\nflowchart LR\n    C2[\"Router C2 (ABR)\"]\n```\n"
)

// routedFake is a fake wired to route every turn to the illustrator, with the
// retrieval seam stubbed out — the re-ask must not touch either.
func routedFake(t *testing.T, replies ...string) (*fakeLLM, *int) {
	t.Helper()
	fake := &fakeLLM{
		classification: `{"protocol":"bgp","daemon":"","chunk_type":"concept","subagent":"illustrator"}`,
		replies:        replies,
	}
	return fake, stubRetrieval(t, []rag.Result{{Query: "q", Protocol: "bgp"}}, nil)
}

// TestRunReasksOnceAndShipsTheFix: a faulted block buys one extra Chat call,
// and a retry that comes back with every block passing is what ships. Success
// does not warn — the repair lives in the metadata, not in the degradation
// channel (decision 9 of ticket 23).
func TestRunReasksOnceAndShipsTheFix(t *testing.T) {
	fake, retrievals := routedFake(t, brokenReply, fixedReply)
	base := newFakeAgent(t, fake)

	out, _, err := Run(context.Background(), base, nil, "draw the OSPF areas", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Text != fixedReply {
		t.Errorf("Text = %q, want the fixed reply", out.Text)
	}
	if fake.responseCalls != 2 {
		t.Errorf("provider replies = %d, want exactly 2 (the turn plus one re-ask)", fake.responseCalls)
	}
	if !out.MermaidChecked || !out.MermaidRetried {
		t.Errorf("MermaidChecked = %v, MermaidRetried = %v, want both true", out.MermaidChecked, out.MermaidRetried)
	}
	if len(out.MermaidBlocks) != 1 || out.MermaidBlocks[0].Violation != "" {
		t.Errorf("MermaidBlocks = %+v, want one not-faulted record for the shipped reply", out.MermaidBlocks)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none: a re-ask that fixed the diagram is not degradation", out.Warnings)
	}
	// The re-ask reclassifies nothing and searches nothing.
	if fake.classifications != 2 {
		t.Errorf("classification calls = %d, want 2 (the generic and the RAG one, neither repeated)", fake.classifications)
	}
	if *retrievals != 1 {
		t.Errorf("retrievals = %d, want 1 — the re-ask must not search again", *retrievals)
	}
}

// TestRunReaskStillBrokenShipsTheOriginalWithOneWarning: the non-regression
// gate. A retry that is still faulted ships the **original** reply, with a
// warning naming both attempts (decision 5).
func TestRunReaskStillBrokenShipsTheOriginalWithOneWarning(t *testing.T) {
	fake, _ := routedFake(t, brokenReply, brokenReply)
	base := newFakeAgent(t, fake)

	out, _, err := Run(context.Background(), base, nil, "draw the OSPF areas", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Text != brokenReply {
		t.Errorf("Text = %q, want the original reply", out.Text)
	}
	if !out.MermaidRetried {
		t.Error("MermaidRetried = false, want the re-ask recorded")
	}
	if len(out.MermaidBlocks) != 1 || out.MermaidBlocks[0].Violation != violationSpecialInLabel {
		t.Errorf("MermaidBlocks = %+v, want the shipped reply's faulted record", out.MermaidBlocks)
	}
	if len(out.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want exactly one naming the two attempts", out.Warnings)
	}
	if !strings.Contains(out.Warnings[0], "two attempts") {
		t.Errorf("warning %q does not name the two attempts", out.Warnings[0])
	}
}

// TestRunReaskWithoutBlockShipsTheOriginal: "at least one block" is half the
// gate. A retry that drops the diagram and answers in prose is degradation, not
// a fix, even though prose breaks none of the rules.
func TestRunReaskWithoutBlockShipsTheOriginal(t *testing.T) {
	fake, _ := routedFake(t, brokenReply, "Sorry — here is the explanation in prose instead.")
	base := newFakeAgent(t, fake)

	out, _, err := Run(context.Background(), base, nil, "draw the OSPF areas", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Text != brokenReply {
		t.Errorf("Text = %q, want the original reply — a retry with no block does not ship", out.Text)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("Warnings = %v, want exactly one", out.Warnings)
	}
}

// TestRunReaskFailureShipsTheOriginal: a failed re-ask call is not a turn
// failure. The first reply is already in hand, so it ships with a warning —
// nothing silent, and the turn never dies.
func TestRunReaskFailureShipsTheOriginal(t *testing.T) {
	fake, _ := routedFake(t, brokenReply)
	fake.failReplyFrom = 2 // the turn's own call goes through; the re-ask does not
	base := newFakeAgent(t, fake)

	out, _, err := Run(context.Background(), base, nil, "draw the OSPF areas", State{})
	if err != nil {
		t.Fatalf("Run: %v — a failed re-ask must not fail the turn", err)
	}
	if out.Text != brokenReply {
		t.Errorf("Text = %q, want the original reply", out.Text)
	}
	if !out.MermaidRetried {
		t.Error("MermaidRetried = false, want the attempt recorded")
	}
	if len(out.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want exactly one naming the failed re-ask", out.Warnings)
	}
	if !strings.Contains(out.Warnings[0], "re-ask failed") {
		t.Errorf("warning %q does not say the re-ask failed", out.Warnings[0])
	}
}

// TestRunUsageSumsBothCalls: Usage is the sum of the turn's call and the
// re-ask's. A meter that under-reports is the one that lets the window
// overflow (decision 10).
func TestRunUsageSumsBothCalls(t *testing.T) {
	fake, _ := routedFake(t, brokenReply, fixedReply)
	fake.usage = 1000
	base := newFakeAgent(t, fake)

	out, _, err := Run(context.Background(), base, nil, "draw the OSPF areas", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Usage.TotalTokens != 2000 {
		t.Errorf("Usage.TotalTokens = %d, want 2000 (both calls summed)", out.Usage.TotalTokens)
	}
	if out.Usage.InputTokens != 2000 {
		t.Errorf("Usage.InputTokens = %d, want 2000 (both calls summed)", out.Usage.InputTokens)
	}
}

// TestRunPaysNoReaskForACleanReply: the happy path pays nothing. A reply whose
// blocks pass — and a reply with no block at all — gets one provider call and
// no retry, which is what keeps the check off the cost of every turn.
func TestRunPaysNoReaskForACleanReply(t *testing.T) {
	for name, reply := range map[string]string{
		"clean diagram": fixedReply,
		"no diagram":    "Just prose, no diagram at all.",
	} {
		t.Run(name, func(t *testing.T) {
			fake, _ := routedFake(t, reply)
			base := newFakeAgent(t, fake)

			out, _, err := Run(context.Background(), base, nil, "explain OSPF", State{})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if fake.responseCalls != 1 {
				t.Errorf("provider replies = %d, want 1 — a clean reply pays no re-ask", fake.responseCalls)
			}
			if out.MermaidRetried {
				t.Error("MermaidRetried = true for a clean reply")
			}
			if !out.MermaidChecked {
				t.Error("MermaidChecked = false — the check has to record that it ran")
			}
			if len(out.Warnings) != 0 {
				t.Errorf("Warnings = %v, want none", out.Warnings)
			}
		})
	}
}

// TestRecheckKeepsTheBrokenReplyOutOfTheConversation: the re-ask's transcript
// carries the broken reply, but Outcome does not — the caller appends
// out.Prompt and out.Text to the Conversation, so only the shipped reply gets
// recorded (fact 2 of ticket 23). It also pins that the caller's transcript
// slice is never mutated.
func TestRecheckKeepsTheBrokenReplyOutOfTheConversation(t *testing.T) {
	fake, _ := routedFake(t, brokenReply, fixedReply)
	base := newFakeAgent(t, fake)

	transcript := []history.Message{
		{Role: history.RoleUser, Content: "earlier question", Status: history.StatusDone},
		{Role: history.RoleAgent, Content: "earlier answer", Status: history.StatusDone},
	}
	before := append([]history.Message(nil), transcript...)

	out, _, err := Run(context.Background(), base, transcript, "draw the OSPF areas", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(transcript) != len(before) {
		t.Fatalf("the caller's transcript grew from %d to %d messages", len(before), len(transcript))
	}
	for i := range before {
		if transcript[i].Content != before[i].Content || transcript[i].Role != before[i].Role {
			t.Errorf("the caller's transcript was mutated at %d: %+v, want %+v", i, transcript[i], before[i])
		}
	}
	if strings.Contains(out.Text, "(ABR)]") && !strings.Contains(out.Text, `"Router C2 (ABR)"`) {
		t.Error("the broken reply is what shipped")
	}

	// The re-ask's request carried the broken reply and the pointed
	// violation — that is the whole mechanism.
	body := string(fake.lastResponseBody)
	if !strings.Contains(body, "C2[Router C2 (ABR)]") {
		t.Error("the re-ask did not send the broken reply")
	}
	if !strings.Contains(body, "line 5") {
		t.Errorf("the re-ask did not point at the violation's line:\n%s", body)
	}
}
