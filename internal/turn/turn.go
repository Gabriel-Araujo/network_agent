// Package turn composes and runs one chat turn: the two classifications
// (routing and RAG mode), the Subagente dispatch, the retrieval with its skip
// rule and <recovered_context> block, and the LLM call — a single function,
// ready for the cmd/ui front to consume (decision:
// .agent/roteamento/issues/06-extrair-a-composicao-do-turno.md).
//
// The agent handed to Run is the turn's **entry point**: it classifies,
// dispatches and composes, and never authors the reply. Every path routes to a
// Subagente — the one the classification named, or rag.DefaultSubagent
// (decision: .agent/generico/issues/03-tirar-o-generico-do-papel-de-resposta.md).
//
// The caller owns the Conversation around the turn: reading the input, the
// transcript and the printing. Nothing here prints, and degradation does not
// kill the turn — fallbacks come back as warnings in Outcome.Warnings. There is
// exactly one exception, and it is the one case that leaves nobody to answer:
// the default Subagente's own configuration failing to load.
package turn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	agentapi "github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag/env"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
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

	// Subagent names the Subagente that answered, and it is an invariant of
	// a turn that answered: whenever Run returns err == nil this is one of
	// the four, never "". A classification that named none, named one whose
	// configuration did not load, or failed outright, answers with
	// rag.DefaultSubagent and reports that name. "" only ever reaches a
	// caller alongside an error.
	Subagent string

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
	// reply, in order. An empty Violation means the scanner's rules did not
	// fault the block — **not** that it renders (see MermaidBlock).
	MermaidBlocks []MermaidBlock

	// MermaidRetried records that a faulted block bought the turn one
	// extra Chat call. It does not say the retry shipped: that is
	// MermaidBlocks plus Warnings.
	MermaidRetried bool

	// SkillsActivated names each Skill the model ordered activated in this
	// turn, in call order, both calls included (the final one and the
	// diagram re-ask). Empty means the model activated nothing — which for
	// a reply carrying a diagram is the whole point of recording it: the
	// prompts of the three agents that draw order the mermaid activation,
	// and a drawing turn with nothing here is an order that did not land
	// (.agent/ui/issues/32-o-generico-desenha-sem-a-skill.md).
	SkillsActivated []string
}

// Run composes and runs one turn for the given transcript and input, and
// returns the reply with its metadata plus the updated State. The agent is the
// turn's entry point; a Subagente mounted from it is who answers.
//
// Degradation does not fail the turn: a failed classification answers with
// rag.DefaultSubagent and empty filters, a failed retrieval answers without
// context — each with a warning. Two things return an error: a failed LLM call,
// which stopped the turn from existing, and the default Subagente's
// configuration failing to load, which leaves nobody to answer it. The State is
// valid either way.
func Run(ctx context.Context, agent *agentapi.Agent, transcript []history.Message, input string, state State) (Outcome, State, error) {
	if agent == nil {
		return Outcome{}, state, errors.New("turn: nil agent")
	}
	var out Outcome

	// The turn asks two disjoint questions, and asks them concurrently
	// (route (b) of
	// .agent/generico/issues/02-caminho-de-classificacao-do-ponto-de-entrada.md):
	// the routing call carries the subagent list alone and picks who
	// answers, the RAG call carries the three filter lists and produces the
	// retrieval's single filter set. The subagent list stays out of the RAG
	// call, so switching Subagente does not re-trigger the search with no
	// gain; the filter lists stay out of the routing call, which is what
	// retired the conditional re-ask — the narrow question *is* the routing
	// call now.
	var (
		routing           map[string]string
		routingErr        error
		classification    map[string]string
		classificationErr error
		classifying       sync.WaitGroup
	)
	classifying.Go(func() {
		routing, routingErr = classifier.Do(ctx, input, []classifier.List{rag.SubagentList()}, false, agent)
	})
	classifying.Go(func() {
		classification, classificationErr = classifier.Do(ctx, input, rag.FilterLists(), true, agent)
	})
	classifying.Wait()

	// The RAG answer is read first: it is a data dependency, not a time
	// one — the routing's Aviso-vs-silence split reads the filters that
	// actually filtered, which is also why its Aviso lands second. Failure:
	// empty filters (identical to "nothing fits") and the comparison zeroes
	// — no half-updated state, the next turn searches again.
	if classificationErr != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("RAG classification failed — continuing with empty filters: %v", classificationErr))
		classification = nil
	} else {
		out.Filters = rag.FiltersFrom(classification)
	}

	turnAgent, subagent, warning, err := agentForTurn(agent, routing, routingErr, out.Filters)
	if err != nil {
		return out, state, err
	}
	if warning != "" {
		out.Warnings = append(out.Warnings, warning)
	}
	out.Subagent = subagent

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

	// The Subagente was mounted before the context existed; the activation
	// needs both facts to deliver the pack of the turn's protocol and say how
	// far the retrieval reached (.agent/frr-file-builder/issues/06-decidir-como-o-pack-chega-ao-modelo.md).
	turnAgent = turnAgent.WithSkillTurn(skills.Turn{Protocol: out.Filters.Protocol, Retrieved: recovered != ""})

	// As falhas de tool do Chat são drenadas antes do erro, não depois: a
	// lista é o que aconteceu no turno, e uma tool pode ter falhado no
	// caminho de um turno que o provedor derrubou depois (ticket 18, §4).
	resp, reports, activated, err := turnAgent.Chat(ctx, transcript, out.Prompt)
	warnReports(&out, reports)
	out.SkillsActivated = append(out.SkillsActivated, activated...)
	if err != nil {
		return out, state, err
	}
	out.Text = resp.OutputText()
	out.Usage = resp.Usage

	recheckMermaid(ctx, turnAgent, transcript, &out)

	// Os dois fatos de diagrama do turno saem juntos numa linha: qual skill
	// o modelo ativou e quantos blocos ele emitiu. É o que transforma
	// "desenhou sem a skill" num grep — a pergunta que o dono teve de fazer
	// ao próprio modelo antes desta linha existir (ticket 32). Vale para os
	// dois fronts, que é por que ela mora aqui e não no logTurn da UI.
	//
	// Turno sem bloco e sem ativação não loga: a linha é o par, não um
	// contador de turnos. O turno que o provedor derrubou também não chega
	// aqui — o erro é logado por quem o recebe.
	if len(out.SkillsActivated) > 0 || len(out.MermaidBlocks) > 0 {
		log.Info("diagram turn",
			"subagent", out.Subagent,
			"skills_activated", out.SkillsActivated,
			"mermaid_blocks", len(out.MermaidBlocks),
		)
	}
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

	// As falhas do re-ask entram no mesmo campo da chamada final, e o dedup
	// é o que impede a mesma recusa de aparecer duas vezes quando o modelo
	// repete a tool na segunda tentativa.
	resp, reports, activated, err := turnAgent.Chat(ctx, retryTranscript, mermaidNotice(first))
	warnReports(out, reports)
	out.SkillsActivated = append(out.SkillsActivated, activated...)
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

// agentForTurn mounts the Subagente that answers the turn and reports its name,
// the Aviso of getting there when there is one, and the one error that kills
// the turn. It never returns base: the entry point classifies, dispatches and
// composes, and is never the author of a reply
// (.agent/generico/issues/03-tirar-o-generico-do-papel-de-resposta.md).
//
// The name comes from routedName; mounting it reads
// resources/agents/<name>/AGENT.md. A named Subagente that does not load is
// degradation like any other — the default answers for it, with an Aviso naming
// both. The default failing to load is the last resort, and the only case with
// nobody left to answer: the error goes up through Run (decision 4 of
// .agent/generico/issues/01-quem-responde-o-nenhum.md).
func agentForTurn(base *agentapi.Agent, routing map[string]string, routingErr error, filters rag.Filters) (*agentapi.Agent, string, string, error) {
	name, warning := routedName(routing, routingErr, filters)
	sub, err := base.ForSubagent(name)
	if err != nil && name != rag.DefaultSubagent {
		warning = fmt.Sprintf("configuration of Subagente %q did not load — answering with the default (%s): %v", name, rag.DefaultSubagent, err)
		name = rag.DefaultSubagent
		sub, err = base.ForSubagent(name)
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("configuration of the default Subagente %q did not load, so no one can answer the turn: %w", name, err)
	}
	return sub, name, warning, nil
}

// routedName is the routing decision alone: which of the four the turn goes to,
// and the Aviso when the app has evidence that getting there was a defect.
// Every path names one — "no Subagente" is answered by rag.DefaultSubagent, not
// by nobody.
//
// The Aviso exists exactly where the app has that evidence (decision 6 of
// .agent/generico/issues/02-caminho-de-classificacao-do-ponto-de-entrada.md):
// a routing call that failed, and an empty value that still filled a search
// filter — the input was read as in-domain, so an empty routing dimension is a
// defect. An empty value with no filter at all is the measured out-of-domain
// signature and stays **silent**: there, vazio ≡ the default is the routing's
// contract, not degradation.
func routedName(routing map[string]string, routingErr error, filters rag.Filters) (string, string) {
	switch {
	case routingErr != nil:
		return rag.DefaultSubagent, fmt.Sprintf("routing classification failed — answering with the default (%s): %v", rag.DefaultSubagent, routingErr)
	case routing[rag.SubagentListName] != "":
		return routing[rag.SubagentListName], ""
	case filters != (rag.Filters{}):
		return rag.DefaultSubagent, fmt.Sprintf("no Subagente in the classification (%s=vazio) with a search filter filled — answering with the default (%s)", rag.SubagentListName, rag.DefaultSubagent)
	default:
		return rag.DefaultSubagent, ""
	}
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
