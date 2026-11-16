package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/google/uuid"
)

type MermaidRepairStart struct {
	AttemptID    string                       `json:"attemptId"`
	Replacements []history.MermaidReplacement `json:"replacements"`
}

// maxRepairRounds bounds the model calls inside ONE repair attempt: the first
// plus the retries that carry the renderer's feedback back to the model.
// Measured 2026-11-15: one call fixed one defect class per round (the edge
// form, then the subgraph title), so the fix needs a bounded loop, not one
// lucky call. The owner asked for the correction over the warning; three
// rounds is the budget that costs at most ~15k tokens and still converges
// class by class.
const maxRepairRounds = 3

var callMermaidRepair = func(ctx context.Context, ag *agent.Agent, failures []agent.MermaidFailure) (agent.MermaidRepairResult, error) {
	return ag.RepairMermaid(ctx, failures)
}

func latestConversationMessage(messages []history.Message) *history.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != history.RoleRepair {
			return &messages[i]
		}
	}
	return nil
}

func repairError(why string) error {
	return newBridgeError(codeInvalid, "Não foi possível reparar o diagrama", why,
		"Peça ao agente outra forma de representar o diagrama.")
}

func (a *App) StartMermaidRepair(convID, messageID string, failures []history.MermaidFailure) (MermaidRepairStart, error) {
	if !a.busyMux.TryLock() {
		return MermaidRepairStart{}, newBridgeError(codeBusy, "O agente já está respondendo", "um turno por vez", "Aguarde a resposta atual terminar.")
	}
	defer a.busyMux.Unlock()
	ag := a.currentAgent()
	if ag == nil {
		return MermaidRepairStart{}, a.errNoAgent()
	}
	conv, err := a.store.Get(convID)
	if err != nil {
		return MermaidRepairStart{}, errHistory(err)
	}
	message := latestConversationMessage(conv.Messages)
	if message == nil || message.ID != messageID || message.Role != history.RoleAgent || message.Status != history.StatusDone {
		return MermaidRepairStart{}, repairError("a resposta não é mais a última resposta da conversa")
	}
	for _, msg := range conv.Messages {
		if msg.Role == history.RoleRepair && msg.MermaidRepair != nil && msg.MermaidRepair.MessageID == messageID {
			return MermaidRepairStart{}, repairError("já houve uma tentativa para esta resposta")
		}
	}
	request, err := checkFailures(message.Content, failures)
	if err != nil {
		return MermaidRepairStart{}, err
	}
	attempt := uuid.NewString()
	started := &history.MermaidRepair{AttemptID: attempt, MessageID: messageID, State: history.MermaidRepairStarted, Failures: failures}
	if _, err = a.appendMermaidRepair(convID, started); err != nil {
		return MermaidRepairStart{}, err
	}
	result, callErr := callMermaidRepair(a.ctx, ag, request)
	current, err := a.store.Get(convID)
	if err != nil {
		return MermaidRepairStart{}, errHistory(err)
	}
	latest := latestConversationMessage(current.Messages)
	if latest == nil || latest.ID != messageID || len(current.Messages) == 0 || current.Messages[len(current.Messages)-1].MermaidRepair == nil || current.Messages[len(current.Messages)-1].MermaidRepair.AttemptID != attempt {
		callErr = fmt.Errorf("a conversa avançou durante o reparo")
	}
	var usage *history.TokenUsage
	if result.Usage != nil {
		usage = projectUsage(*result.Usage, contextWindow{})
	}
	replacements := make([]history.MermaidReplacement, 0, len(result.Replacements))
	for _, replacement := range result.Replacements {
		replacements = append(replacements, history.MermaidReplacement{Index: replacement.Index, Source: replacement.Source})
	}
	if callErr == nil {
		_, callErr = replaceMermaidSources(message.Content, replacements)
	}
	if callErr != nil {
		failed := &history.MermaidRepair{AttemptID: attempt, MessageID: messageID, State: history.MermaidRepairFailed, Error: callErr.Error(), Usage: usage}
		if _, err = a.appendMermaidRepair(convID, failed); err != nil {
			return MermaidRepairStart{}, err
		}
		return MermaidRepairStart{}, repairError(callErr.Error())
	}
	candidate := &history.MermaidRepair{AttemptID: attempt, MessageID: messageID, State: history.MermaidRepairCandidate, Replacements: replacements, Usage: usage}
	if _, err = a.appendMermaidRepair(convID, candidate); err != nil {
		return MermaidRepairStart{}, err
	}
	if a.repairAttempts == nil {
		a.repairAttempts = make(map[string]string)
	}
	a.repairAttempts[attempt] = convID
	return MermaidRepairStart{AttemptID: attempt, Replacements: replacements}, nil
}

func (a *App) FinishMermaidRepair(convID, attemptID string, accepted bool) (history.Conversation, error) {
	if !a.busyMux.TryLock() {
		return history.Conversation{}, newBridgeError(codeBusy, "O agente já está respondendo", "um turno por vez", "Aguarde a resposta atual terminar.")
	}
	defer a.busyMux.Unlock()
	if a.repairAttempts[attemptID] != convID {
		return history.Conversation{}, repairError("a tentativa foi interrompida ou pertence a outra conversa")
	}
	conv, err := a.store.Get(convID)
	if err != nil {
		return history.Conversation{}, errHistory(err)
	}
	if len(conv.Messages) == 0 {
		return history.Conversation{}, repairError("tentativa não encontrada")
	}
	last := conv.Messages[len(conv.Messages)-1]
	candidate := last.MermaidRepair
	if last.Role != history.RoleRepair || candidate == nil || candidate.AttemptID != attemptID || candidate.State != history.MermaidRepairCandidate {
		return history.Conversation{}, repairError("a tentativa não está mais pendente")
	}
	original := latestConversationMessage(conv.Messages)
	if original == nil || original.ID != candidate.MessageID || original.Role != history.RoleAgent {
		return history.Conversation{}, repairError("a resposta original ficou desatualizada")
	}
	finished := &history.MermaidRepair{AttemptID: attemptID, MessageID: original.ID, State: history.MermaidRepairRejected, Usage: candidate.Usage}
	if accepted {
		finished.CorrectedContent, err = replaceMermaidSources(original.Content, candidate.Replacements)
		if err != nil {
			return history.Conversation{}, repairError(fmt.Sprintf("não foi possível recompor o candidato: %v", err))
		}
		finished.State = history.MermaidRepairAccepted
	} else {
		finished.Error = "O renderizador rejeitou a correção."
	}
	conv, err = a.appendMermaidRepair(convID, finished)
	if err != nil {
		return history.Conversation{}, err
	}
	delete(a.repairAttempts, attemptID)
	return history.ProjectConversation(conv), nil
}

func (a *App) appendMermaidRepair(convID string, repair *history.MermaidRepair) (history.Conversation, error) {
	conv, err := a.store.Append(convID, history.Message{ID: uuid.NewString(), Role: history.RoleRepair, Time: history.NowStamp(), Status: history.StatusDone, MermaidRepair: repair})
	if err != nil {
		return history.Conversation{}, errHistory(err)
	}
	return conv, nil
}

// checkFailures validates the reported failures against the text they claim to
// come from — the persisted reply or a composed candidate — and returns the
// call payload. Index, source and error are checked against the blocks of that
// text, never trusted from the client.
func checkFailures(text string, failures []history.MermaidFailure) ([]agent.MermaidFailure, error) {
	if len(failures) == 0 {
		return nil, repairError("nenhum bloco válido foi informado")
	}
	blocks, err := mermaidSources(text)
	if err != nil {
		return nil, repairError("nenhum bloco válido foi informado")
	}
	seen := make(map[int]bool, len(failures))
	request := make([]agent.MermaidFailure, 0, len(failures))
	for _, failure := range failures {
		if failure.Index < 1 || failure.Index > len(blocks) || seen[failure.Index] || failure.Source != blocks[failure.Index-1].Source || strings.TrimSpace(failure.Error) == "" {
			return nil, repairError("os blocos ou erros não correspondem à resposta gravada")
		}
		seen[failure.Index] = true
		request = append(request, agent.MermaidFailure{Index: failure.Index, Source: failure.Source, Error: failure.Error})
	}
	return request, nil
}

// mergeReplacements folds a round's replacements into the attempt's set: later
// rounds win per block index, because the model re-freed what still failed.
func mergeReplacements(base, round []history.MermaidReplacement) []history.MermaidReplacement {
	merged := make([]history.MermaidReplacement, 0, len(base)+len(round))
	pos := make(map[int]int, len(base)+len(round))
	for _, r := range base {
		pos[r.Index] = len(merged)
		merged = append(merged, r)
	}
	for _, r := range round {
		if i, ok := pos[r.Index]; ok {
			merged[i] = r
			continue
		}
		pos[r.Index] = len(merged)
		merged = append(merged, r)
	}
	return merged
}

// addTokenUsage accumulates the usage of a repair attempt: every model round
// reports its own, and the terminal event carries the sum — the usage is the
// measure of what the correction cost, not of its last call.
func addTokenUsage(total *history.TokenUsage, round *history.TokenUsage) *history.TokenUsage {
	if round == nil {
		return total
	}
	if total == nil {
		copy := *round
		return &copy
	}
	total.Input += round.Input
	total.Output += round.Output
	total.Total += round.Total
	total.ContextUsed = round.ContextUsed
	total.ContextTotal = round.ContextTotal
	total.ContextSource = round.ContextSource
	return total
}

// RetryMermaidRepair runs another model round inside the SAME attempt, carrying
// the renderer's feedback on the current candidate back to the model. One
// attempt, at most maxRepairRounds model calls: each round replaces earlier
// candidates block by block, and the last candidate is what Finish composes.
// Identity, concurrency and stale checks are the start's: the failures are
// checked against the composed candidate, and a conversation that advanced
// rejects the result and preserves the original.
func (a *App) RetryMermaidRepair(convID, attemptID string, failures []history.MermaidFailure) (MermaidRepairStart, error) {
	if !a.busyMux.TryLock() {
		return MermaidRepairStart{}, newBridgeError(codeBusy, "O agente já está respondendo", "um turno por vez", "Aguarde a resposta atual terminar.")
	}
	defer a.busyMux.Unlock()
	ag := a.currentAgent()
	if ag == nil {
		return MermaidRepairStart{}, a.errNoAgent()
	}
	if a.repairAttempts[attemptID] != convID {
		return MermaidRepairStart{}, repairError("a tentativa foi interrompida ou pertence a outra conversa")
	}
	conv, err := a.store.Get(convID)
	if err != nil {
		return MermaidRepairStart{}, errHistory(err)
	}
	if len(conv.Messages) == 0 {
		return MermaidRepairStart{}, repairError("tentativa não encontrada")
	}
	last := conv.Messages[len(conv.Messages)-1]
	previous := last.MermaidRepair
	if last.Role != history.RoleRepair || previous == nil || previous.AttemptID != attemptID || previous.State != history.MermaidRepairCandidate {
		return MermaidRepairStart{}, repairError("a tentativa não está mais pendente")
	}
	original := latestConversationMessage(conv.Messages)
	if original == nil || original.ID != previous.MessageID || original.Role != history.RoleAgent {
		return MermaidRepairStart{}, repairError("a resposta original ficou desatualizada")
	}
	rounds := 0
	for _, msg := range conv.Messages {
		if msg.Role == history.RoleRepair && msg.MermaidRepair != nil && msg.MermaidRepair.AttemptID == attemptID && msg.MermaidRepair.State == history.MermaidRepairCandidate {
			rounds++
		}
	}
	if rounds >= maxRepairRounds {
		return MermaidRepairStart{}, repairError("a tentativa esgotou as rodadas de correção")
	}
	candidate, err := replaceMermaidSources(original.Content, previous.Replacements)
	if err != nil {
		return MermaidRepairStart{}, repairError("não foi possível recompor o candidato")
	}
	request, err := checkFailures(candidate, failures)
	if err != nil {
		return MermaidRepairStart{}, err
	}
	result, callErr := callMermaidRepair(a.ctx, ag, request)
	current, err := a.store.Get(convID)
	if err != nil {
		return MermaidRepairStart{}, errHistory(err)
	}
	latest := latestConversationMessage(current.Messages)
	if latest == nil || latest.ID != previous.MessageID || len(current.Messages) == 0 || current.Messages[len(current.Messages)-1].MermaidRepair == nil || current.Messages[len(current.Messages)-1].MermaidRepair.AttemptID != attemptID {
		callErr = fmt.Errorf("a conversa avançou durante o reparo")
	}
	usage := previous.Usage
	if result.Usage != nil {
		usage = addTokenUsage(usage, projectUsage(*result.Usage, contextWindow{}))
	}
	round := make([]history.MermaidReplacement, 0, len(result.Replacements))
	for _, replacement := range result.Replacements {
		round = append(round, history.MermaidReplacement{Index: replacement.Index, Source: replacement.Source})
	}
	replacements := mergeReplacements(previous.Replacements, round)
	if callErr == nil {
		_, callErr = replaceMermaidSources(original.Content, replacements)
	}
	if callErr != nil {
		failed := &history.MermaidRepair{AttemptID: attemptID, MessageID: previous.MessageID, State: history.MermaidRepairFailed, Error: callErr.Error(), Usage: usage}
		if _, err = a.appendMermaidRepair(convID, failed); err != nil {
			return MermaidRepairStart{}, err
		}
		return MermaidRepairStart{}, repairError(callErr.Error())
	}
	next := &history.MermaidRepair{AttemptID: attemptID, MessageID: previous.MessageID, State: history.MermaidRepairCandidate, Replacements: replacements, Usage: usage}
	if _, err = a.appendMermaidRepair(convID, next); err != nil {
		return MermaidRepairStart{}, err
	}
	return MermaidRepairStart{AttemptID: attemptID, Replacements: replacements}, nil
}
