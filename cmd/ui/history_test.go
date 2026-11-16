package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/turn"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/openai/openai-go/v3/packages/respjson"
	"github.com/openai/openai-go/v3/responses"
)

// windowBodyLMStudio é o corpo do GET /api/v1/models (a sondagem da janela de
// contexto) no formato do LM Studio com o modelo carregado — 65024, enquanto o
// teto do arquivo, 262144, e o context_size declarado na configuração, 128000,
// são outros números. É o caso que prova que o medidor passa a mostrar a janela
// real.
const windowBodyLMStudio = `{"models":[
	{"type":"llm","key":"modelo-de-mentira","loaded_instances":[{"id":"modelo-de-mentira","config":{"context_length":65024}}],"max_context_length":262144}]}`

// ---------- provedor de mentira ----------

// fakeProvider é um provedor OpenAI-compatible de mentira: registra as
// requisições recebidas e responde com um texto fixo (e o usage, quando pedido).
// É o que permite provar o turno ponta a ponta sem um provedor real de pé.
type fakeProvider struct {
	mu         sync.Mutex
	requests   []map[string]any
	reply      string
	status     int
	noUsage    bool
	delay      time.Duration
	onRequest  func()
	windowBody string
}

func (f *fakeProvider) setReply(reply string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reply = reply
}

// setWindow liga o endpoint de sondagem da janela de contexto
// (GET /api/v1/models) com o corpo dado.
func (f *fakeProvider) setWindow(body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.windowBody = body
}

func (f *fakeProvider) setStatus(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = status
}

func (f *fakeProvider) setHook(hook func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onRequest = hook
}

// request devolve a n-ésima requisição recebida.
func (f *fakeProvider) request(t *testing.T, n int) map[string]any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if n >= len(f.requests) {
		t.Fatalf("esperava ao menos %d requisições, chegaram %d", n+1, len(f.requests))
	}
	return f.requests[n]
}

func (f *fakeProvider) handler(w http.ResponseWriter, r *http.Request) {
	// A sondagem da janela de contexto é GET; o turno é POST. Caminho de
	// sondagem que o teste não ligou responde 404: é assim que se prova o
	// provedor que não expõe a janela.
	if r.Method == http.MethodGet {
		f.mu.Lock()
		body := f.windowBody
		f.mu.Unlock()
		if body == "" || r.URL.Path != "/api/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	f.requests = append(f.requests, body)
	status, reply, noUsage, delay, hook := f.status, f.reply, f.noUsage, f.delay, f.onRequest
	f.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	if hook != nil {
		hook()
	}
	if status != 0 {
		http.Error(w, "erro simulado do provedor", status)
		return
	}

	resp := map[string]any{
		"id": "resp_1", "object": "response", "created_at": 0,
		"status": "completed", "model": "modelo-de-mentira",
		"output": []any{map[string]any{
			"type": "message", "id": "msg_1", "role": "assistant", "status": "completed",
			"content": []any{map[string]any{"type": "output_text", "text": reply, "annotations": []any{}}},
		}},
		"parallel_tool_calls": false, "tool_choice": "auto", "tools": []any{},
	}
	if !noUsage {
		resp["usage"] = map[string]any{"input_tokens": 11, "output_tokens": 7, "total_tokens": 18}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ---------- coletor de eventos ----------

// eventCollector ocupa o lugar da ponte do Wails nos testes: registra os
// eventos que o AskAgent emitiria (com um contexto que não veio dos hooks do
// app, o runtime real do Wails encerra o processo).
type eventCollector struct {
	mu     sync.Mutex
	events []capturedEvent
}

type capturedEvent struct {
	name string
	data []any
}

func (c *eventCollector) emit(name string, data ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, capturedEvent{name: name, data: data})
}

func (c *eventCollector) names() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.events))
	for _, e := range c.events {
		out = append(out, e.name)
	}
	return out
}

// payloads devolve os mapas de todos os eventos com o nome dado.
func (c *eventCollector) payloads(name string) []map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []map[string]string
	for _, e := range c.events {
		if e.name != name || len(e.data) == 0 {
			continue
		}
		if m, ok := e.data[0].(map[string]string); ok {
			out = append(out, m)
		}
	}
	return out
}

// joined concatena o campo dado dos eventos com o nome dado, na ordem.
func (c *eventCollector) joined(name, field string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sb strings.Builder
	for _, e := range c.events {
		if e.name != name || len(e.data) == 0 {
			continue
		}
		if m, ok := e.data[0].(map[string]string); ok {
			sb.WriteString(m[field])
		}
	}
	return sb.String()
}

// ---------- montagem ----------

// novoAppDeTeste monta um App com o histórico num diretório temporário e o
// agente apontado para o provedor de mentira. Nenhum caminho aqui lê ou grava a
// configuração real do home do usuário: o par provedor/modelo chega pelo
// reconnect, como chega na vida real.
func novoAppDeTeste(t *testing.T, url string, contextTotal int64) (*App, *eventCollector) {
	t.Helper()

	a := &App{}
	a.ctx = context.Background()
	a.store = history.NewFileStore(t.TempDir())
	events := &eventCollector{}
	a.emit = events.emit

	a.reconnect(types.Config{
		Providers: []types.Provider{{
			Name:   "Provedor de mentira",
			Url:    url,
			ApiKey: "teste",
			Models: []types.Model{{Id: "modelo-de-mentira", Name: "Modelo", ContextSize: contextTotal}},
		}},
	})
	esperaConectar(t, a)

	// Por padrão o turno é só a chamada do LLM: o que estes testes provam é o
	// que o AskAgent faz com o resultado da composição, e a composição inteira
	// tem os testes dela em internal/turn.
	trocaOTurno(t, turnoSoDoChat)
	return a, events
}

// esperaConectar aguarda a conexão sair de `connecting` e exige que ela tenha
// dado em agente. O reconnect é background desde o ticket 13 (a carga do modelo
// no provedor leva dezenas de segundos e travaria o arranque do app), então
// quem monta um App para o teste tem de esperar pelo que o app só mostra na
// tela.
func esperaConectar(t *testing.T, a *App) {
	t.Helper()
	limite := time.Now().Add(30 * time.Second)
	for a.GetConnection().State == connConnecting {
		if time.Now().After(limite) {
			t.Fatal("a conexão não saiu de connecting em 30s")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if conn := a.GetConnection(); conn.State != connReady {
		t.Fatalf("a conexão terminou em %q (causa: %s), esperava ready", conn.State, conn.Cause)
	}
}

// composicao é o tipo da composição do turno — o runTurn de history.go.
type composicao = func(context.Context, *agent.Agent, []history.Message, string, turn.State) (turn.Outcome, turn.State, error)

// trocaOTurno põe uma composição de mentira no lugar do turn.Run pela duração
// do teste. O turn.Run de verdade classifica duas vezes, busca no banco do RAG
// e chama o LLM três vezes: nada que um teste de unidade tenha.
func trocaOTurno(t *testing.T, fake composicao) {
	t.Helper()
	anterior := runTurn
	runTurn = fake
	t.Cleanup(func() { runTurn = anterior })
}

// turnoSoDoChat é a composição reduzida à chamada do LLM: sem classificação,
// sem retrieval, sem scanner de Mermaid — o provedor de mentira recebe o
// transcript e o prompt como os recebia antes da composição entrar.
func turnoSoDoChat(ctx context.Context, ag *agent.Agent, transcript []history.Message, input string, state turn.State) (turn.Outcome, turn.State, error) {
	resp, _, _, err := ag.Chat(ctx, transcript, input)
	if err != nil {
		return turn.Outcome{Prompt: input}, state, err
	}
	return turn.Outcome{Text: resp.OutputText(), Prompt: input, Usage: resp.Usage}, state, nil
}

// codigoDoErro exige que o erro seja o JSON estruturado da ponte e devolve o
// código, para o front-end decidir além de mostrar o texto.
func codigoDoErro(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("esperava um erro, veio nil")
	}
	var b bridgeError
	if jsonErr := json.Unmarshal([]byte(err.Error()), &b); jsonErr != nil {
		t.Fatalf("o erro não é o JSON estruturado da ponte: %v (erro: %q)", jsonErr, err.Error())
	}
	if b.What == "" || b.Next == "" {
		t.Errorf("erro estruturado incompleto: what=%q next=%q", b.What, b.Next)
	}
	return b.Code
}

// rolesETextos extrai papel e texto dos itens de input de uma requisição.
func rolesETextos(t *testing.T, req map[string]any) ([]string, []string) {
	t.Helper()
	input, ok := req["input"].([]any)
	if !ok {
		t.Fatalf("requisição sem lista de input: %v", req["input"])
	}
	roles := make([]string, 0, len(input))
	textos := make([]string, 0, len(input))
	for _, raw := range input {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("item de input não é objeto: %v", raw)
		}
		role, _ := item["role"].(string)
		roles = append(roles, role)
		textos = append(textos, textoDoConteudo(item["content"]))
	}
	return roles, textos
}

// textoDoConteudo lê o texto de um conteúdo de mensagem, seja string direta ou
// lista de partes.
func textoDoConteudo(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var sb strings.Builder
		for _, part := range c {
			if m, ok := part.(map[string]any); ok {
				if s, ok := m["text"].(string); ok {
					sb.WriteString(s)
				}
			}
		}
		return sb.String()
	}
	return ""
}

// ---------- testes ----------

// TestAskAgentGravaOTurnoEApreservaOTextoNoStreaming prova o turno completo:
// a mensagem do usuário entra no disco ANTES da chamada ao provedor, a resposta
// volta gravada com ids, tempo e usage medidos, e o stream transmite o texto
// byte a byte (o markdown depende do whitespace) com o id da conversa e da
// mensagem em cada evento.
func TestAskAgentGravaOTurnoEApreservaOTextoNoStreaming(t *testing.T) {
	// Markdown com quebras de linha e lista: se o stream perder whitespace, a
	// concatenação dos tokens deixa de bater com o texto.
	reply := "# Título\n\nlinha um\n\n- item a\n- item b\n"

	fp := &fakeProvider{reply: reply, delay: 20 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, events := novoAppDeTeste(t, srv.URL, 128000)

	conv, err := a.CreateConversation("Conversa de teste")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if conv.Provider != "Provedor de mentira" || conv.Model != "modelo-de-mentira" {
		t.Errorf("provedor/modelo gravados %q/%q, esperava o par ativo", conv.Provider, conv.Model)
	}
	if len(conv.Messages) != 0 {
		t.Errorf("conversa nova chegou com %d mensagens", len(conv.Messages))
	}
	if _, err := time.Parse(time.RFC3339, conv.CreatedAt); err != nil {
		t.Errorf("CreatedAt não é RFC3339: %q (%v)", conv.CreatedAt, err)
	}

	// No momento da chamada ao provedor, a mensagem do usuário já tem que
	// estar no disco — um turno interrompido não pode sumir sem rastro.
	fp.setHook(func() {
		disk, err := a.store.Get(conv.ID)
		if err != nil {
			t.Errorf("leitura do disco no meio do turno falhou: %v", err)
			return
		}
		if len(disk.Messages) != 1 || disk.Messages[0].Content != "primeira pergunta" {
			t.Errorf("na chamada ao provedor o disco tinha %d mensagens: a do usuário não foi gravada antes", len(disk.Messages))
		}
	})

	got, err := a.AskAgent(conv.ID, "primeira pergunta")
	if err != nil {
		t.Fatalf("AskAgent: %v", err)
	}

	if len(got.Messages) != 2 {
		t.Fatalf("conversa com %d mensagens, esperava 2 (usuário + agente)", len(got.Messages))
	}
	user, agent := got.Messages[0], got.Messages[1]

	if user.Role != history.RoleUser || user.Content != "primeira pergunta" || user.Status != history.StatusDone {
		t.Errorf("mensagem do usuário errada: %+v", user)
	}
	if user.ID == "" || user.Time == "" {
		t.Errorf("mensagem do usuário sem id/tempo: %+v", user)
	}

	if agent.Role != history.RoleAgent || agent.Content != reply || agent.Status != history.StatusDone {
		t.Errorf("mensagem do agente errada: role=%q status=%q content=%q", agent.Role, agent.Status, agent.Content)
	}
	if agent.ID == user.ID {
		t.Error("as duas mensagens nasceram com o mesmo id")
	}
	ts, err := time.Parse(time.RFC3339, agent.Time)
	if err != nil {
		t.Errorf("Time não é RFC3339: %q (%v)", agent.Time, err)
	} else if ts.Location() != time.UTC {
		t.Errorf("Time não está em UTC: %q", agent.Time)
	}
	if agent.ElapsedMs < 10 {
		t.Errorf("ElapsedMs=%d: o tempo de geração não foi medido (provedor demorou 20ms)", agent.ElapsedMs)
	}
	if agent.Usage == nil {
		t.Fatal("a mensagem do agente não guardou o usage reportado pelo provedor")
	}
	if agent.Usage.Input != 11 || agent.Usage.Output != 7 || agent.Usage.Total != 18 {
		t.Errorf("usage %+v, esperava 11/7/18", *agent.Usage)
	}
	if agent.Usage.ContextUsed != 11 || agent.Usage.ContextTotal != 128000 {
		t.Errorf("contexto usado/total %d/%d, esperava 11/128000", agent.Usage.ContextUsed, agent.Usage.ContextTotal)
	}
	if agent.Tokens != 7 {
		t.Errorf("Tokens=%d, esperava a saída medida (7)", agent.Tokens)
	}

	// O disco é a fonte: a conversa devolvida bate com o que foi gravado.
	disk, err := a.store.Get(conv.ID)
	if err != nil {
		t.Fatalf("Get depois do turno: %v", err)
	}
	if len(disk.Messages) != 2 || disk.Messages[1].ID != agent.ID {
		t.Errorf("o disco não tem as duas mensagens do turno: %+v", disk.Messages)
	}

	// O stream transmite o texto inteiro, byte a byte.
	if streamed := events.joined("agent:token", "token"); streamed != reply {
		t.Errorf("a concatenação dos tokens não bate com o texto:\n%q\n%q", streamed, reply)
	}

	tokens := events.payloads("agent:token")
	if len(tokens) == 0 {
		t.Fatal("nenhum agent:token foi emitido")
	}
	for i, ev := range tokens {
		if ev["conversationId"] != conv.ID || ev["messageId"] != agent.ID {
			t.Errorf("token %d sem os ids da conversa/mensagem: %v", i, ev)
			break
		}
	}
	done := events.payloads("agent:done")
	if len(done) != 1 || done[0]["conversationId"] != conv.ID || done[0]["messageId"] != agent.ID {
		t.Errorf("agent:done sem os ids: %v", done)
	}

	// Ordem: status, tokens, done.
	names := events.names()
	if names[0] != "agent:status" || names[len(names)-1] != "agent:done" {
		t.Errorf("ordem dos eventos inesperada: %v", names)
	}

	// A requisição levou system + a mensagem nova (o transcript estava vazio).
	roles, textos := rolesETextos(t, fp.request(t, 0))
	if len(roles) != 2 || roles[0] != "system" || roles[1] != "user" {
		t.Fatalf("input do primeiro turno com papéis %v, esperava [system user]", roles)
	}
	if textos[1] != "primeira pergunta" {
		t.Errorf("a mensagem nova no input foi %q", textos[1])
	}
}

// TestAskAgentRetomaDoDisco prova a retomada: a segunda chamada monta o input a
// partir do transcript gravado — a conversa continua a mesma, e o provedor
// recebe as duas perguntas e a primeira resposta.
func TestAskAgentRetomaDoDisco(t *testing.T) {
	fp := &fakeProvider{reply: "resposta um"}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, _ := novoAppDeTeste(t, srv.URL, 128000)

	conv, err := a.CreateConversation("Conversa retomada")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if _, err := a.AskAgent(conv.ID, "primeira pergunta"); err != nil {
		t.Fatalf("primeiro turno: %v", err)
	}

	fp.setReply("resposta dois")
	got, err := a.AskAgent(conv.ID, "segunda pergunta")
	if err != nil {
		t.Fatalf("segundo turno: %v", err)
	}

	roles, textos := rolesETextos(t, fp.request(t, 1))
	esperados := []string{"system", "user", "assistant", "user"}
	if len(roles) != len(esperados) {
		t.Fatalf("input da retomada com %d itens (%v), esperava %d", len(roles), roles, len(esperados))
	}
	for i, esperado := range esperados {
		if roles[i] != esperado {
			t.Errorf("papel do item %d: %q, esperava %q", i, roles[i], esperado)
		}
	}
	if textos[1] != "primeira pergunta" || textos[2] != "resposta um" || textos[3] != "segunda pergunta" {
		t.Errorf("os textos reenviados não são o transcript do disco: %q", textos)
	}

	if len(got.Messages) != 4 {
		t.Errorf("a conversa retomada tem %d mensagens, esperava 4 (mesma conversa, não uma nova)", len(got.Messages))
	}

	convs, err := a.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 1 || convs[0].ID != conv.ID {
		t.Errorf("a retomada criou outra conversa: %d conversas no histórico", len(convs))
	}
}

// TestAskAgentGravaOTurnoQueFalhaEPulaNoReenvio prova o turno que falha: ele é
// gravado com status error e o AgentError montado no Go, nada é transmitido, a
// conversa volta como está no disco — e o reenvio seguinte não leva a mensagem
// que falhou (um turno que falhou não é contexto).
func TestAskAgentGravaOTurnoQueFalhaEPulaNoReenvio(t *testing.T) {
	// 400 (e não 500): o SDK não repete a requisição, então o teste fica com
	// exatamente uma chamada por turno.
	fp := &fakeProvider{status: http.StatusBadRequest}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, events := novoAppDeTeste(t, srv.URL, 128000)

	conv, err := a.CreateConversation("Conversa que falha")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	got, err := a.AskAgent(conv.ID, "vai falhar")
	if err != nil {
		t.Fatalf("o erro do turno é dado, não exceção — AskAgent devolveu erro: %v", err)
	}

	if len(got.Messages) != 2 {
		t.Fatalf("conversa com %d mensagens, esperava usuário + agente", len(got.Messages))
	}
	agent := got.Messages[1]
	if agent.Status != history.StatusError {
		t.Errorf("a mensagem do turno que falhou ficou com status %q", agent.Status)
	}
	if agent.Error == nil || agent.Error.What == "" || agent.Error.Why == "" || agent.Error.Next == "" {
		t.Fatalf("AgentError incompleto na mensagem: %+v", agent.Error)
	}
	if agent.Content != "" {
		t.Errorf("a mensagem que falhou guardou conteúdo: %q", agent.Content)
	}

	disk, err := a.store.Get(conv.ID)
	if err != nil {
		t.Fatalf("Get depois da falha: %v", err)
	}
	if len(disk.Messages) != 2 || disk.Messages[1].ID != agent.ID || disk.Messages[1].Status != history.StatusError {
		t.Errorf("o turno que falhou não está no disco como a conversa devolvida: %+v", disk.Messages)
	}

	if n := len(events.payloads("agent:token")); n != 0 {
		t.Errorf("%d tokens transmitidos de um turno que falhou", n)
	}
	if n := len(events.payloads("agent:done")); n != 0 {
		t.Errorf("agent:done emitido para um turno que falhou")
	}

	// O reenvio pula a mensagem de erro (mas mantém a pergunta do usuário).
	fp.setStatus(0)
	fp.setReply("agora vai")
	got2, err := a.AskAgent(conv.ID, "de novo")
	if err != nil {
		t.Fatalf("segundo turno: %v", err)
	}

	roles, textos := rolesETextos(t, fp.request(t, 1))
	if len(roles) != 3 || roles[0] != "system" || roles[1] != "user" || roles[2] != "user" {
		t.Fatalf("papéis do reenvio %v, esperava [system user user] (a mensagem de erro fica de fora)", roles)
	}
	if textos[1] != "vai falhar" || textos[2] != "de novo" {
		t.Errorf("textos do reenvio %q", textos)
	}
	if len(got2.Messages) != 4 {
		t.Errorf("a conversa depois do reenvio tem %d mensagens, esperava 4", len(got2.Messages))
	}
	if got2.Messages[3].Status != history.StatusDone {
		t.Errorf("o reenvio não terminou como done: %q", got2.Messages[3].Status)
	}
}

// TestErrosEstruturadosAtravessamAPonte fixa o formato e os códigos dos erros
// que a UI recebe: JSON com what/why/next, e código que distingue "a conversa
// sumiu" de "o pedido é inválido" e de "já tem turno rodando".
func TestErrosEstruturadosAtravessamAPonte(t *testing.T) {
	fp := &fakeProvider{reply: "ok"}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, _ := novoAppDeTeste(t, srv.URL, 128000)

	conv, err := a.CreateConversation("Conversa")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if _, err := a.AskAgent("", "oi"); codigoDoErro(t, err) != codeInvalid {
		t.Errorf("AskAgent sem id de conversa devolveu %v", err)
	}
	if _, err := a.AskAgent(conv.ID, "   "); codigoDoErro(t, err) != codeInvalid {
		t.Errorf("AskAgent com prompt vazio devolveu %v", err)
	}
	if _, err := a.AskAgent("nao-existe", "oi"); codigoDoErro(t, err) != codeNotFound {
		t.Errorf("AskAgent em conversa inexistente devolveu %v", err)
	}
	if _, err := a.RenameConversation("nao-existe", "título"); codigoDoErro(t, err) != codeNotFound {
		t.Errorf("RenameConversation em conversa inexistente devolveu %v", err)
	}
	if _, err := a.RenameConversation(conv.ID, "  "); codigoDoErro(t, err) != codeInvalid {
		t.Errorf("RenameConversation com título vazio devolveu %v", err)
	}
	if err := a.DeleteConversation("nao-existe"); codigoDoErro(t, err) != codeNotFound {
		t.Errorf("DeleteConversation em conversa inexistente devolveu %v", err)
	}
	if _, err := a.CreateConversation("  "); codigoDoErro(t, err) != codeInvalid {
		t.Errorf("CreateConversation com título vazio devolveu %v", err)
	}

	// Turno em andamento: a trava do busyMux recusa o segundo.
	a.busyMux.Lock()
	_, err = a.AskAgent(conv.ID, "oi")
	a.busyMux.Unlock()
	if codigoDoErro(t, err) != codeBusy {
		t.Errorf("AskAgent com turno em andamento devolveu %v", err)
	}

	// Sem agente conectado (primeira execução, sem configuração): erro claro,
	// nunca panic.
	sem := &App{ctx: context.Background(), store: history.NewFileStore(t.TempDir()), emit: (&eventCollector{}).emit}
	if _, err := sem.AskAgent("qualquer", "oi"); codigoDoErro(t, err) != codeNoAgent {
		t.Errorf("AskAgent sem agente devolveu %v", err)
	}
	if _, err := sem.CreateConversation("título"); codigoDoErro(t, err) != codeNoAgent {
		t.Errorf("CreateConversation sem agente devolveu %v", err)
	}
}

// TestACausaDaAusenciaDoAgenteChegaAoFront prova o conserto da mensagem do
// ticket 13: o mesmo agente nil tem três causas, e o drive do ticket 07 mediu
// as três chegando como "cadastre um provedor" — com o provedor cadastrado,
// certo, e o modelo apenas carregando. Cada estado da conexão tem seu código,
// nos três pontos que checam o agente, e a causa crua da falha viaja com ele.
func TestACausaDaAusenciaDoAgenteChegaAoFront(t *testing.T) {
	casos := []struct {
		nome   string
		conn   connection
		codigo string
	}{
		{"nada configurado", connection{}, codeNoAgent},
		{"conectando", connection{State: connConnecting}, codeConnecting},
		{"falhou", connection{State: connFailed, Cause: `Post "http://localhost:1234/api/v1/models/load": dial tcp: connection refused`}, codeConnectFailed},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			a := &App{
				ctx:   context.Background(),
				store: history.NewFileStore(t.TempDir()),
				emit:  (&eventCollector{}).emit,
				conn:  c.conn,
			}

			_, err := a.AskAgent("qualquer", "oi")
			if codigoDoErro(t, err) != c.codigo {
				t.Errorf("AskAgent devolveu %v, esperava o código %q", err, c.codigo)
			}
			if _, errCriar := a.CreateConversation("título"); codigoDoErro(t, errCriar) != c.codigo {
				t.Errorf("CreateConversation devolveu %v, esperava o código %q", errCriar, c.codigo)
			}
			if _, errJanela := a.GetContextWindow(); codigoDoErro(t, errJanela) != c.codigo {
				t.Errorf("GetContextWindow devolveu %v, esperava o código %q", errJanela, c.codigo)
			}

			// A causa é o que o usuário não tem como adivinhar: sem ela, a
			// falha de conexão e a falta de configuração são a mesma tela.
			if c.conn.Cause != "" {
				var b bridgeError
				if jsonErr := json.Unmarshal([]byte(err.Error()), &b); jsonErr != nil {
					t.Fatalf("o erro não é o JSON da ponte: %v", jsonErr)
				}
				if b.Why != c.conn.Cause {
					t.Errorf("a causa que chegou ao front foi %q, esperava %q", b.Why, c.conn.Cause)
				}
			}
		})
	}
}

// TestListConversationsVaziaNaPrimeiraExecucao prova o primeiro uso: sem
// diretório de histórico, a listagem é vazia e sem erro — e listar não cria
// diretório nenhum (o histórico nasce na primeira gravação).
func TestListConversationsVaziaNaPrimeiraExecucao(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "history")

	a := &App{ctx: context.Background(), store: history.NewFileStore(dir), emit: (&eventCollector{}).emit}

	convs, err := a.ListConversations()
	if err != nil {
		t.Fatalf("ListConversations no primeiro uso: %v", err)
	}
	if convs == nil {
		t.Error("lista veio nil: no JSON do front-end isso é null, não []")
	}
	if len(convs) != 0 {
		t.Errorf("histórico recém-nascido com %d conversas", len(convs))
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Error("listar criou o diretório do histórico")
	}
}

// TestGetContextWindowSondaOProvedorEMarcaODeclarado prova a superfície que o
// medidor da UI consome: a janela vem do endpoint do provedor quando ele a
// expõe e do context_size declarado quando não — sempre com a origem, para o
// número declarado não se passar por medido.
func TestGetContextWindowSondaOProvedorEMarcaODeclarado(t *testing.T) {
	t.Run("provedor que expõe a janela", func(t *testing.T) {
		fp := &fakeProvider{reply: "ok"}
		fp.setWindow(windowBodyLMStudio)
		srv := httptest.NewServer(http.HandlerFunc(fp.handler))
		defer srv.Close()

		a, _ := novoAppDeTeste(t, srv.URL, 128000)

		win, err := a.GetContextWindow()
		if err != nil {
			t.Fatalf("GetContextWindow: %v", err)
		}
		if win.Total != 65024 {
			t.Errorf("janela %d, esperava 65024 — a instância carregada, não o declarado (128000) nem o teto do arquivo (262144)", win.Total)
		}
		if win.Source != agent.WindowSourceProvider {
			t.Errorf("origem %q, esperava %q", win.Source, agent.WindowSourceProvider)
		}
	})

	t.Run("provedor que não expõe a janela", func(t *testing.T) {
		fp := &fakeProvider{reply: "ok"}
		srv := httptest.NewServer(http.HandlerFunc(fp.handler))
		defer srv.Close()

		a, _ := novoAppDeTeste(t, srv.URL, 128000)

		win, err := a.GetContextWindow()
		if err != nil {
			t.Fatalf("GetContextWindow: %v", err)
		}
		if win.Total != 128000 || win.Source != agent.WindowSourceConfig {
			t.Errorf("janela %d/%q, esperava o declarado 128000/%q", win.Total, win.Source, agent.WindowSourceConfig)
		}
	})

	t.Run("sem agente", func(t *testing.T) {
		sem := &App{ctx: context.Background(), store: history.NewFileStore(t.TempDir()), emit: (&eventCollector{}).emit}
		if _, err := sem.GetContextWindow(); codigoDoErro(t, err) != codeNoAgent {
			t.Errorf("GetContextWindow sem agente devolveu %v", err)
		}
	})
}

// TestAskAgentGravaAJanelaComOrigem prova que o turno grava a janela sondada,
// com a origem, junto do uso: é daí que o medidor da UI lê o número quando a
// conversa é reaberta.
func TestAskAgentGravaAJanelaComOrigem(t *testing.T) {
	casos := []struct {
		nome       string
		windowBody string
		total      int
		source     string
	}{
		{"provedor expõe a janela", windowBodyLMStudio, 65024, agent.WindowSourceProvider},
		{"provedor não expõe a janela", "", 128000, agent.WindowSourceConfig},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			fp := &fakeProvider{reply: "resposta"}
			if c.windowBody != "" {
				fp.setWindow(c.windowBody)
			}
			srv := httptest.NewServer(http.HandlerFunc(fp.handler))
			defer srv.Close()

			a, _ := novoAppDeTeste(t, srv.URL, 128000)

			conv, err := a.CreateConversation("Conversa com janela")
			if err != nil {
				t.Fatalf("CreateConversation: %v", err)
			}
			got, err := a.AskAgent(conv.ID, "pergunta")
			if err != nil {
				t.Fatalf("AskAgent: %v", err)
			}

			usage := got.Messages[1].Usage
			if usage == nil {
				t.Fatal("a mensagem do agente não tem usage")
			}
			if usage.ContextTotal != c.total || usage.ContextSource != c.source {
				t.Errorf("usage gravou a janela %d/%q, esperava %d/%q", usage.ContextTotal, usage.ContextSource, c.total, c.source)
			}
			if usage.ContextUsed != 11 {
				t.Errorf("o contexto usado %d não é o input medido (11)", usage.ContextUsed)
			}

			disk, err := a.store.Get(conv.ID)
			if err != nil {
				t.Fatalf("Get depois do turno: %v", err)
			}
			if disk.Messages[1].Usage == nil ||
				disk.Messages[1].Usage.ContextTotal != c.total ||
				disk.Messages[1].Usage.ContextSource != c.source {
				t.Errorf("o disco não guardou a janela com a origem: %+v", disk.Messages[1].Usage)
			}
		})
	}
}

// TestProjectUsageDistingueNaoReportadoDeZero prova que "o provedor não
// reportou uso" não vira uma medida de zero no disco.
func TestProjectUsageDistingueNaoReportadoDeZero(t *testing.T) {
	win := contextWindow{Total: 128000, Source: agent.WindowSourceProvider}

	if u := projectUsage(responses.ResponseUsage{}, win); u != nil {
		t.Errorf("usage ausente virou medida: %+v", u)
	}

	var reported responses.ResponseUsage
	reported.InputTokens = 11
	reported.OutputTokens = 7
	reported.TotalTokens = 18
	reported.JSON.InputTokens = respjson.NewField("11")
	reported.JSON.OutputTokens = respjson.NewField("7")
	reported.JSON.TotalTokens = respjson.NewField("18")

	u := projectUsage(reported, win)
	if u == nil {
		t.Fatal("usage reportado virou nil")
	}
	if u.Input != 11 || u.Output != 7 || u.Total != 18 || u.ContextUsed != 11 || u.ContextTotal != 128000 {
		t.Errorf("projeção errada: %+v", *u)
	}
	if u.ContextSource != agent.WindowSourceProvider {
		t.Errorf("origem %q, esperava %q", u.ContextSource, agent.WindowSourceProvider)
	}
}

// TestAskAgentGravaOAutorEOsAvisosDoTurno prova o que a composição acrescenta à
// mensagem gravada: quem respondeu (o Subagente; "" é o agente genérico) e os
// avisos de degradação do turno. Os dois vão ao disco junto da resposta porque o
// Store é append-only — aviso não gravado na hora se perde — e porque é a
// resposta degradada que eles explicam.
//
// Prova também o prompt: o que fica na mensagem do usuário é o que ele digitou,
// não o out.Prompt da composição. Na UI o Content é também o que o balão
// renderiza, e o bloco <recovered_context> não é texto do usuário.
func TestAskAgentGravaOAutorEOsAvisosDoTurno(t *testing.T) {
	avisos := []string{
		"RAG classification failed — continuing with empty filters: sem provedor",
		"retrieval failed, continuing without RAG context: sem banco",
	}

	fp := &fakeProvider{reply: "ignorado"}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, _ := novoAppDeTeste(t, srv.URL, 128000)
	conv, err := a.CreateConversation("Conversa roteada")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	trocaOTurno(t, func(_ context.Context, _ *agent.Agent, _ []history.Message, input string, state turn.State) (turn.Outcome, turn.State, error) {
		return turn.Outcome{
			Text:             "explicação do teacher",
			Prompt:           input + "\n\n<recovered_context>\n[{\"doc\":1}]\n</recovered_context>",
			Subagent:         "teacher",
			Warnings:         avisos,
			RecoveredContext: "[{\"doc\":1}]",
		}, state, nil
	})

	got, err := a.AskAgent(conv.ID, "o que é route reflector?")
	if err != nil {
		t.Fatalf("AskAgent: %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("conversa com %d mensagens, esperava 2", len(got.Messages))
	}
	user, resposta := got.Messages[0], got.Messages[1]

	if user.Content != "o que é route reflector?" {
		t.Errorf("a mensagem do usuário guardou o prompt composto: %q", user.Content)
	}
	if user.Subagent != "" {
		t.Errorf("a mensagem do usuário ganhou autor: %q", user.Subagent)
	}
	if resposta.Subagent != "teacher" {
		t.Errorf("autor gravado %q, esperava o Subagente que respondeu", resposta.Subagent)
	}
	if !slices.Equal(resposta.Warnings, avisos) {
		t.Errorf("avisos gravados %q, esperava %q", resposta.Warnings, avisos)
	}

	// O disco é a fonte: a conversa devolvida é a que foi gravada.
	disk, err := a.store.Get(conv.ID)
	if err != nil {
		t.Fatalf("Get depois do turno: %v", err)
	}
	if disk.Messages[1].Subagent != "teacher" || !slices.Equal(disk.Messages[1].Warnings, avisos) {
		t.Errorf("autor/avisos não chegaram ao disco: %+v", disk.Messages[1])
	}

	// Turno do agente genérico: autor vazio e nenhum aviso — é o contrato que
	// o balão lê para escrever "Agente", o mesmo de uma conversa gravada antes
	// desta decisão.
	trocaOTurno(t, func(_ context.Context, _ *agent.Agent, _ []history.Message, input string, state turn.State) (turn.Outcome, turn.State, error) {
		return turn.Outcome{Text: "resposta do genérico", Prompt: input}, state, nil
	})
	got, err = a.AskAgent(conv.ID, "e agora?")
	if err != nil {
		t.Fatalf("segundo turno: %v", err)
	}
	if generico := got.Messages[3]; generico.Subagent != "" || len(generico.Warnings) != 0 {
		t.Errorf("turno limpo do genérico gravou autor/avisos: %+v", generico)
	}
}

// TestAskAgentGravaOsAvisosDoTurnoQueFalhou prova que a degradação sobrevive ao
// turno que morre: a composição pode ter avisado (classificação que falhou,
// retrieval que não veio) antes de a chamada ao LLM matar o turno, e esses
// avisos são o que o usuário tem para saber o que o turno tentou.
func TestAskAgentGravaOsAvisosDoTurnoQueFalhou(t *testing.T) {
	avisos := []string{"generic classification failed — answering with the generic agent: sem provedor"}

	fp := &fakeProvider{reply: "ignorado"}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, _ := novoAppDeTeste(t, srv.URL, 128000)
	conv, err := a.CreateConversation("Conversa que falha degradada")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	trocaOTurno(t, func(_ context.Context, _ *agent.Agent, _ []history.Message, input string, state turn.State) (turn.Outcome, turn.State, error) {
		return turn.Outcome{Prompt: input, Warnings: avisos}, state, errors.New("o provedor caiu")
	})

	got, err := a.AskAgent(conv.ID, "vai falhar degradado")
	if err != nil {
		t.Fatalf("o erro do turno é dado, não exceção: %v", err)
	}
	falha := got.Messages[1]
	if falha.Status != history.StatusError || falha.Error == nil {
		t.Fatalf("o turno que falhou não foi gravado como erro: %+v", falha)
	}
	if !slices.Equal(falha.Warnings, avisos) {
		t.Errorf("avisos do turno que falhou %q, esperava %q", falha.Warnings, avisos)
	}
}

// TestAskAgentGuardaOStateDoTurnoPorConversa prova o State da composição entre
// turnos: é guardado por Conversa (a skip rule compara com o turno anterior
// DELA) e guardado também quando o turno falha — o turn.Run devolve o State
// atualizado nas duas saídas, e perdê-lo faria o turno seguinte re-disparar o
// retrieval sem motivo.
//
// O teste olha o mapa, não o valor: turn.State é opaco fora do internal/turn,
// então de fora não há como construir dois States distinguíveis.
func TestAskAgentGuardaOStateDoTurnoPorConversa(t *testing.T) {
	fp := &fakeProvider{reply: "ok"}
	srv := httptest.NewServer(http.HandlerFunc(fp.handler))
	defer srv.Close()

	a, _ := novoAppDeTeste(t, srv.URL, 128000)
	primeira, err := a.CreateConversation("Primeira")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	segunda, err := a.CreateConversation("Segunda")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if _, err := a.AskAgent(primeira.ID, "turno que dá certo"); err != nil {
		t.Fatalf("turno da primeira conversa: %v", err)
	}

	// A segunda conversa falha no LLM: o State dela tem de ficar guardado do
	// mesmo jeito.
	trocaOTurno(t, func(_ context.Context, _ *agent.Agent, _ []history.Message, input string, state turn.State) (turn.Outcome, turn.State, error) {
		return turn.Outcome{Prompt: input}, state, errors.New("o provedor caiu")
	})
	if _, err := a.AskAgent(segunda.ID, "turno que falha"); err != nil {
		t.Fatalf("turno da segunda conversa: %v", err)
	}

	if len(a.turnState) != 2 {
		t.Fatalf("o app guardou %d States, esperava um por Conversa: %v", len(a.turnState), a.turnState)
	}
	if _, ok := a.turnState[primeira.ID]; !ok {
		t.Error("o State do turno que deu certo não foi guardado")
	}
	if _, ok := a.turnState[segunda.ID]; !ok {
		t.Error("o State do turno que falhou não foi guardado — o próximo turno re-buscaria sem motivo")
	}
}
