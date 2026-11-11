// Package turn composes and runs one chat turn: the generic classification
// (display + routing), the Subagente dispatch, RAG mode (skip rule, retrieval,
// the <recovered_context> block) and the LLM call — a single function, ready for
// the cmd/ui front to consume (decision:
// .agent/roteamento/issues/06-extrair-a-composicao-do-turno.md).
//
// The caller owns the Conversation around the turn: reading the input, the
// transcript and the printing. Nothing here prints, and no degradation kills
// the turn — every fallback comes back as a warning in Outcome.Warnings.
package turn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag/env"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
	"github.com/openai/openai-go/v3/responses"
)

var log = logger.Named("TURN")

// State is the composition's state between turns of one Conversation: the
// previous RAG classification (the skip rule compares against it) and the last
// real recovered context. The zero value is the first turn. It is opaque on
// purpose — callers hold one per Conversation and hand it back; only the
// composition reads it. Run returns the updated State even on error, so a
// caller that keeps going does not lose the turn's bookkeeping.
type State struct {
	prevRAG     map[string]string
	lastContext string
}

// Outcome is the turn's answer plus the metadata the front shows. It is valid
// even when Run returns an error: the composition ran up to the LLM call, and
// what it learned (classification, routing, context) is here.
type Outcome struct {
	// Text is the reply the turn's agent produced.
	Text string

	// Prompt is the composed user message the provider received: the input
	// plus the <recovered_context> block. Callers record it in the
	// transcript — that is what the next turns re-send.
	Prompt string

	// Usage is what the provider reported for the turn. The front projects
	// it (it owns the meter and the window probe); a provider that reports
	// nothing leaves the zero value.
	Usage responses.ResponseUsage

	// Subagent names the Subagente that answered ("" = the generic agent).
	// It is who actually answered: a classification that names a Subagente
	// whose configuration did not load answers with the generic agent and
	// stays "".
	Subagent string

	// Generic is the generic classification (display + routing), nil when
	// that call failed. The display line per list derives from it.
	Generic map[string]string

	// Filters is the retrieval filter set of this turn (zero = none, either
	// "nothing fits" or a failed RAG classification).
	Filters rag.Filters

	// RecoveredContext is the <recovered_context> JSON injected in the
	// prompt ("" = none this turn: retrieval failed or never ran).
	RecoveredContext string

	// Warnings are the degradation warnings of the turn, in order. Empty
	// means a clean turn; a warning is never silent.
	Warnings []string

	// MermaidChecked records that the diagram scan ran on the shipped
	// reply. It is the difference between "no block broke a rule" and "the
	// check never happened".
	MermaidChecked bool

	// MermaidBlocks is one record per ```mermaid block of the shipped
	// reply, in order. An empty Violation means the three rules did not
	// fault the block — **not** that it renders (see MermaidBlock).
	MermaidBlocks []MermaidBlock

	// MermaidRetried records that a faulted block bought the turn one
	// extra Chat call. It does not say the retry shipped: that is
	// MermaidBlocks plus Warnings.
	MermaidRetried bool
}

// Run composes and runs one turn for the given transcript and input, and
// returns the reply with its metadata plus the updated State. The agent is the
// generic one; the turn may be answered by a Subagente mounted from it.
//
// Degradation never fails the turn: a failed classification answers with the
// generic agent and empty filters, a failed retrieval answers without context —
// each with a warning. The error is what stopped the turn from existing: a
// failed LLM call. The State is valid either way.
func Run(ctx context.Context, agent *agentapi.Agent, transcript []history.Message, input string, state State) (Outcome, State, error) {
	if agent == nil {
		return Outcome{}, state, errors.New("turn: nil agent")
	}
	var out Outcome

	// Generic classification — display + routing: the subagent list's value
	// picks the Subagente that answers the turn.
	generic, classificationErr := classifier.Do(ctx, input, rag.Lists(), false, agent)
	if classificationErr == nil {
		out.Generic = generic
	}
	turnAgent, subagent, warning := routeTurn(ctx, agent, input, generic, classificationErr)
	if warning != "" {
		out.Warnings = append(out.Warnings, warning)
	}
	out.Subagent = subagent

	// RAG classification — the retrieval's single filter set. Only the
	// three filter lists ride here (rag.FilterLists): the subagent list
	// rides the generic call only, so switching Subagente does not
	// re-trigger the search with no gain. Failure: empty filters (identical
	// to "nothing fits") and the comparison zeroes — no half-updated state,
	// the next turn searches again.
	classification, err := classifier.Do(ctx, input, rag.FilterLists(), true, agent)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("RAG classification failed — continuing with empty filters: %v", err))
		classification = nil
	} else {
		out.Filters = rag.FiltersFrom(classification)
	}

	// Skip rule: the retrieval only runs when the RAG classification differs
	// from the previous turn; skipped, the context is the last real
	// retrieval's ("" = there never was one, so no block at all).
	recovered := ""
	if shouldSearch(classification, state.prevRAG) {
		results, mirror, err := retrieval(ctx, agent, input, out.Filters)
		if err != nil && !errors.Is(err, ragretriever.ErrMirrorNotSaved) {
			out.Warnings = append(out.Warnings, fmt.Sprintf("retrieval failed, continuing without RAG context: %v", err))
		} else {
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("retrieval mirror not saved, continuing with the in-memory context: %v", err))
			} else {
				log.Info("retrieval saved", "path", mirror)
			}
			recovered = contextJSON(results)
			state.lastContext = recovered
		}
	} else {
		recovered = state.lastContext
	}
	out.RecoveredContext = recovered
	state.prevRAG = classification

	out.Prompt = input
	if recovered != "" {
		out.Prompt = input + "\n\n<recovered_context>\n" + recovered + "\n</recovered_context>"
	}

	// As falhas de tool do Chat ficam descartadas aqui de propósito: o ticket 18
	// só abriu o canal, e quem as drena para out.Warnings é o 19, a fatia do
	// Aviso na tela.
	resp, _, err := turnAgent.Chat(ctx, transcript, out.Prompt)
	if err != nil {
		return out, state, err
	}
	out.Text = resp.OutputText()
	out.Usage = resp.Usage

	recheckMermaid(ctx, turnAgent, transcript, &out)
	return out, state, nil
}

// recheckMermaid scans the reply's ```mermaid blocks and, when one is faulted,
// buys the turn **one** retry with the violation pointed at by line
// (decisions 3-5 of
// .agent/roteamento/issues/23-validar-os-blocos-mermaid-no-turno.md).
//
// The retry is the final call again and nothing else: no reclassification and
// no new retrieval, so out.Prompt, the filters and the <recovered_context>
// stay standing. The retry's reply ships only if it carries at least one block
// and all of them pass; any other case ships the original reply with a warning
// naming both attempts. Success does not warn — Warnings is the degradation
// channel and the CLI prints every entry, so filling it with successful events
// blinds it (decision 9).
//
// A failed retry call is not a turn failure: the original reply is already in
// hand, so it ships with the warning.
func recheckMermaid(ctx context.Context, turnAgent *agentapi.Agent, transcript []history.Message, out *Outcome) {
	out.MermaidChecked = true
	out.MermaidBlocks = scanMermaid(out.Text)
	if !faulted(out.MermaidBlocks) {
		return
	}

	first := out.MermaidBlocks
	log.Info("faulted diagram block — re-asking once", "violation", first[0].Violation, "line", first[0].Line)
	out.MermaidRetried = true

	// The broken reply rides the retry's transcript, never the
	// Conversation's: the caller is who appends out.Prompt and out.Text, so
	// clipping here keeps the broken text out of what gets recorded.
	retryTranscript := append(slices.Clip(transcript),
		history.Message{Role: history.RoleUser, Content: out.Prompt, Status: history.StatusDone},
		history.Message{Role: history.RoleAgent, Content: out.Text, Status: history.StatusDone},
	)

	// Falhas de tool descartadas, como na chamada final: ticket 19.
	resp, _, err := turnAgent.Chat(ctx, retryTranscript, mermaidNotice(first))
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("diagram block faulted (%s, line %d) and the re-ask failed — shipping the first reply: %v", first[0].Violation, first[0].Line, err))
		return
	}

	// The turn consumed both calls, so Usage is their sum: a meter that
	// under-reports is the one that lets the window overflow.
	out.Usage = sumUsage(out.Usage, resp.Usage)

	retried := scanMermaid(resp.OutputText())
	if !shippable(retried) {
		out.Warnings = append(out.Warnings, fmt.Sprintf("diagram block faulted (%s, line %d) in two attempts — shipping the first reply", first[0].Violation, first[0].Line))
		return
	}
	out.Text = resp.OutputText()
	out.MermaidBlocks = retried
}

// sumUsage adds two providers' usage reports. Only the totals are summed: the
// front projects those, and the nested breakdowns have no meaning added across
// two calls.
func sumUsage(a, b responses.ResponseUsage) responses.ResponseUsage {
	a.InputTokens += b.InputTokens
	a.OutputTokens += b.OutputTokens
	a.TotalTokens += b.TotalTokens
	return a
}

// routeTurn picks the Agent that answers the turn, with the targeted re-ask in
// between: "no Subagente but some filter list filled" is the measured signature
// of the empty routing (9 of 9 cases), and it buys one extra classification
// call before the fallback to the generic agent
// (decision: .agent/roteamento/issues/24-remedio-para-a-classificacao-vazia.md).
// The happy path pays nothing: a classification that names a Subagente never
// gets here.
func routeTurn(ctx context.Context, base *agentapi.Agent, input string, generic map[string]string, classificationErr error) (*agentapi.Agent, string, string) {
	if !needsReask(generic, classificationErr) {
		return agentForTurn(base, generic, classificationErr)
	}

	log.Info("empty routing with filters filled — re-asking the subagent list alone", "input", input)
	name, err := reaskSubagent(ctx, base, input)
	if err != nil {
		return base, "", fmt.Sprintf("no Subagente in two attempts (%s=vazio, then the re-ask failed) — answering with the generic agent: %v", rag.SubagentListName, err)
	}
	if name == "" {
		return base, "", fmt.Sprintf("no Subagente in two attempts (%s=vazio in the generic classification and in the re-ask) — answering with the generic agent", rag.SubagentListName)
	}

	// The re-ask's answer replaces the empty one in the classification
	// itself — the map Outcome.Generic holds — so the display line, the
	// routing and the metadata the front consumes cannot contradict
	// each other.
	generic[rag.SubagentListName] = name
	log.Info("the re-ask routed the turn", "subagent", name)
	return agentForTurn(base, generic, nil)
}

// needsReask is the re-ask's trigger: the generic classification succeeded,
// named no Subagente, and still filled at least one filter list — it understood
// the input as in-domain and only the routing dimension came back empty. Any of
// the three filter lists counts; all four empty is out-of-domain routing
// working, and never re-asks.
func needsReask(generic map[string]string, classificationErr error) bool {
	if classificationErr != nil || generic[rag.SubagentListName] != "" {
		return false
	}
	return rag.FiltersFrom(generic) != rag.Filters{}
}

// reaskSubagent runs the re-ask: one classification with the routing list alone
// (rag.SubagentList), the same classifier untouched. The call has Temperature 0,
// so re-asking the same prompt would return the same answer — the second call
// has to be a different question, and a single list is the same task, narrower,
// without asking any model for a capability it may not have.
func reaskSubagent(ctx context.Context, base *agentapi.Agent, input string) (string, error) {
	result, err := classifier.Do(ctx, input, []classifier.List{rag.SubagentList()}, false, base)
	if err != nil {
		return "", err
	}
	return result[rag.SubagentListName], nil
}

// agentForTurn picks the Agent that answers the turn from the generic
// classification (display + routing): the subagent list's value mounts the
// Subagente's configuration at resources/agents/<name>/AGENT.md. An empty value
// or a classification error falls back to the generic agent, with an explicit
// warning — the turn never dies. It also reports the name of the Subagente that
// actually answers ("" when the generic agent does).
func agentForTurn(base *agentapi.Agent, generic map[string]string, classificationErr error) (*agentapi.Agent, string, string) {
	if classificationErr != nil {
		return base, "", fmt.Sprintf("generic classification failed — answering with the generic agent: %v", classificationErr)
	}
	name := generic[rag.SubagentListName]
	if name == "" {
		return base, "", fmt.Sprintf("no Subagente in the classification (%s=vazio) — answering with the generic agent", rag.SubagentListName)
	}
	sub, err := base.ForSubagent(name)
	if err != nil {
		return base, "", fmt.Sprintf("configuration of Subagente %q did not load — answering with the generic agent: %v", name, err)
	}
	return sub, name, ""
}

// shouldSearch applies the skip rule: the retriever only runs when the RAG
// classification differs from the previous turn's — exact map equality, empty
// values included. current == nil is a failed classification and previous ==
// nil is "no previous turn" (first turn or zeroed comparison): both always
// search.
func shouldSearch(current, previous map[string]string) bool {
	return current == nil || previous == nil || !maps.Equal(current, previous)
}

// contextJSON serializes the retrieval results for the <recovered_context>
// block (and for re-injecting when the search is skipped).
func contextJSON(results []rag.Result) string {
	if results == nil {
		results = []rag.Result{}
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Warn("failed to serialize the recovered context", "err", err)
		return ""
	}
	return string(data)
}

// retrieval runs one retrieval for the input with the filter set. It is a
// package variable as the composition's test seam: a real retrieval needs the
// RAG database and the embedding server, which no unit test has.
var retrieval = retrieve

// retrieve builds the retriever Config (DATABASE_URL DSN, the embedding client
// from LoadEmbbedAgent, an LLM query generator) and runs the retrieval for the
// text input with the classifier's filter set.
func retrieve(ctx context.Context, agent *agentapi.Agent, input string, filters rag.Filters) ([]rag.Result, string, error) {
	dsn, _ := env.LoadEnv()
	if dsn == "" {
		return nil, "", fmt.Errorf("DATABASE_URL missing — add it to .env or export it in the environment")
	}

	// Embeddings come from LoadEmbbedAgent (LM Studio on localhost:1234),
	// which serves the SAME model the ingester used (2560 dims).
	embAgent := agentapi.LoadEmbbedAgent()
	cfg := rag.Config{
		DSN:            dsn,
		EmbeddingModel: embAgent.ModelName,
		Embedder:       embAgent.Client,
		QueryGen: &ragretriever.LLMQueryGenerator{
			Client:    agent.Client,
			ModelName: agent.ModelName,
		},
	}

	return ragretriever.Do(ctx, input, filters, cfg)
}
