package turn

import (
	"context"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
)

// TestNeedsReaskTrigger pins the re-ask's trigger: the generic classification
// succeeded, named no Subagente, and still filled at least one filter list.
// Any of the three filter lists counts — the trigger does not presume the model
// gets a particular one right. All four empty is out-of-domain routing working,
// and a named Subagente is the happy path: neither re-asks.
func TestNeedsReaskTrigger(t *testing.T) {
	cases := []struct {
		name    string
		generic map[string]string
		err     error
		want    bool
	}{
		{"protocol filled", map[string]string{"protocol": "bgp", "daemon": "", "chunk_type": "", rag.SubagentListName: ""}, nil, true},
		{"daemon filled", map[string]string{"protocol": "", "daemon": "bgpd", "chunk_type": "", rag.SubagentListName: ""}, nil, true},
		{"chunk_type filled", map[string]string{"protocol": "", "daemon": "", "chunk_type": "concept", rag.SubagentListName: ""}, nil, true},
		{"all four empty", map[string]string{"protocol": "", "daemon": "", "chunk_type": "", rag.SubagentListName: ""}, nil, false},
		{"Subagente named", map[string]string{"protocol": "bgp", "daemon": "bgpd", "chunk_type": "concept", rag.SubagentListName: "modifier"}, nil, false},
		{"failed classification", nil, errFake("boom"), false},
		{"no classification at all", nil, nil, false},
	}
	for _, c := range cases {
		if got := needsReask(c.generic, c.err); got != c.want {
			t.Errorf("%s: needsReask = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestRunReaskRoutesAndUpdatesGeneric: an empty subagent with filters filled
// buys one single-list re-ask, whose answer both routes the turn and replaces
// the empty value in Outcome.Generic — display, routing and metadata cannot
// contradict each other. A routed re-ask is not a degradation: no warning.
func TestRunReaskRoutesAndUpdatesGeneric(t *testing.T) {
	fake := &fakeLLM{
		classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"command_reference","subagent":""}`,
		reask:          `{"subagent":"modifier"}`,
	}
	base := newFakeAgent(t, fake)
	stubRetrieval(t, []rag.Result{}, nil)

	out, _, err := Run(context.Background(), base, nil, "change the remote-as in frr.conf", State{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fake.reasks != 1 {
		t.Errorf("re-ask calls = %d, want exactly 1", fake.reasks)
	}
	if out.Subagent != "modifier" {
		t.Errorf("Subagent = %q, want %q", out.Subagent, "modifier")
	}
	if out.Generic[rag.SubagentListName] != "modifier" {
		t.Errorf("Generic[%s] = %q, want the re-ask's answer", rag.SubagentListName, out.Generic[rag.SubagentListName])
	}
	if len(out.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none for a routed re-ask", out.Warnings)
	}
	if got := systemFromBody(t, fake.lastResponseBody); !strings.Contains(got, "Modifier") {
		t.Error("the turn was not answered by the Subagente the re-ask named")
	}
}

// TestRunReaskEmptyWarnsOnceNamingTwoAttempts: a re-ask that comes back empty
// too — or that fails — answers with the generic agent and leaves exactly one
// warning, naming both attempts, so the transcript is countable in the
// reopening trigger.
func TestRunReaskEmptyWarnsOnceNamingTwoAttempts(t *testing.T) {
	cases := []struct {
		name string
		fake *fakeLLM
	}{
		{"empty twice", &fakeLLM{
			classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"command_reference","subagent":""}`,
			reask:          `{"subagent":""}`,
		}},
		{"re-ask failed", &fakeLLM{
			classification: `{"protocol":"bgp","daemon":"bgpd","chunk_type":"command_reference","subagent":""}`,
			failReask:      true,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := newFakeAgent(t, c.fake)
			stubRetrieval(t, []rag.Result{}, nil)

			out, _, err := Run(context.Background(), base, nil, "change the remote-as in frr.conf", State{})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if c.fake.reasks == 0 {
				t.Fatal("the re-ask never fired")
			}
			if out.Subagent != "" {
				t.Errorf("Subagent = %q, want the generic agent", out.Subagent)
			}
			if len(out.Warnings) != 1 {
				t.Fatalf("Warnings = %v, want exactly one", out.Warnings)
			}
			if !strings.Contains(out.Warnings[0], "two attempts") {
				t.Errorf("warning %q, want both attempts named in it", out.Warnings[0])
			}
		})
	}
}

// TestRunPaysNoReaskWhenNotTriggered: neither the happy path nor the
// out-of-domain input pays the extra call — the first is routed, the second
// falls back to the generic agent with today's single warning.
func TestRunPaysNoReaskWhenNotTriggered(t *testing.T) {
	cases := []struct {
		name           string
		classification string
		wantSubagent   string
		wantWarnings   int
	}{
		{"happy path", `{"protocol":"bgp","daemon":"bgpd","chunk_type":"concept","subagent":"teacher"}`, "teacher", 0},
		{"out of domain", `{"protocol":"","daemon":"","chunk_type":"","subagent":""}`, "", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &fakeLLM{classification: c.classification}
			base := newFakeAgent(t, fake)
			stubRetrieval(t, []rag.Result{}, nil)

			out, _, err := Run(context.Background(), base, nil, "input", State{})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if fake.reasks != 0 {
				t.Errorf("re-ask calls = %d, want none", fake.reasks)
			}
			if fake.classifications != 2 {
				t.Errorf("classification calls = %d, want 2 (generic + RAG)", fake.classifications)
			}
			if out.Subagent != c.wantSubagent {
				t.Errorf("Subagent = %q, want %q", out.Subagent, c.wantSubagent)
			}
			if len(out.Warnings) != c.wantWarnings {
				t.Errorf("Warnings = %v, want %d", out.Warnings, c.wantWarnings)
			}
		})
	}
}
