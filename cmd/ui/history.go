package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/turn"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v3/responses"
)

// ---------- erros que atravessam a ponte ----------

// bridgeError é o erro que atravessa a ponte do Wails.
//
// O Wails entrega `error` ao TypeScript como string, então uma sentinela como
// history.ErrNotFound chegaria indistinguível de "o disco falhou" e a UI não
// teria como escrever um `next` útil. O JSON carrega código + what/why/next —
// o formato que a UI já renderiza (types.ts) — e o front decide pelo código
// além de mostrar o texto.
type bridgeError struct {
	Code string `json:"code"`
	What string `json:"what"`
	Why  string `json:"why"`
	Next string `json:"next"`
}

// Códigos que a UI usa para decidir o que fazer:
//
//	not_found      — a conversa não está mais no disco (recarregar a lista)
//	invalid        — dados recusados (pedido vazio, título vazio, arquivo corrompido)
//	io             — falha de leitura/escrita no disco do histórico
//	no_agent       — não há provedor e modelo configurados
//	connecting     — a conexão está em andamento (não é erro: aguardar)
//	connect_failed — a conexão falhou, e o `why` traz a causa
//	busy           — já existe um turno em andamento
const (
	codeNotFound      = "not_found"
	codeInvalid       = "invalid"
	codeIO            = "io"
	codeNoAgent       = "no_agent"
	codeConnecting    = "connecting"
	codeConnectFailed = "connect_failed"
	codeBusy          = "busy"
)

func (e bridgeError) Error() string {
	raw, err := json.Marshal(e)
	if err != nil {
		// Não deve acontecer (os campos são todos string), mas o What ainda
		// diz mais à UI do que uma string vazia.
		return e.What
	}
	return string(raw)
}

func newBridgeError(code, what, why, next string) error {
	return bridgeError{Code: code, What: what, Why: why, Next: next}
}

// errNoAgent explica a ausência do agente pelo estado da conexão, em vez de
// pelo palpite de que falta configuração. O mesmo nil tem três causas — nada
// configurado, conexão em andamento, conexão falhada — e mandar o usuário à
// tela de Configurações nas três é a mensagem que o drive do ticket 07 mediu
// apontando para o lugar errado, com a configuração certa no disco.
//
// É um ponto único de propósito: os três pontos que checam o agente (o turno,
// o medidor de janela e a criação de conversa) davam a mesma resposta errada.
func (a *App) errNoAgent() error {
	conn := a.GetConnection()
	switch conn.State {
	case connConnecting:
		return newBridgeError(codeConnecting,
			"Conectando ao provedor",
			"o modelo está sendo carregado no provedor, e o agente só existe depois disso",
			"Aguarde a carga terminar — o estado aparece ao lado do nome do modelo.")
	case connFailed:
		return newBridgeError(codeConnectFailed,
			"A conexão com o provedor falhou",
			conn.Cause,
			"Confira se o provedor está no ar e se o modelo configurado existe; depois use Reconectar.")
	default:
		return newBridgeError(codeNoAgent,
			"Nenhum provedor e modelo configurados",
			"não há provedor e modelo ativos para atender o pedido",
			"Abra Configurações e cadastre um provedor com ao menos um modelo.")
	}
}

// errHistory converte um erro do Store no erro que a UI recebe. O why carrega a
// mensagem crua do Store — é ela que nomeia o arquivo corrompido, decidido no
// ticket 06.
func errHistory(err error) error {
	switch {
	case errors.Is(err, history.ErrNotFound):
		return newBridgeError(codeNotFound,
			"A conversa não está mais no disco",
			err.Error(),
			"Atualize a lista de conversas: ela pode ter sido apagada em outro lugar.")
	case errors.Is(err, history.ErrInvalid):
		return newBridgeError(codeInvalid,
			"Não consegui ler a conversa",
			err.Error(),
			"O arquivo está corrompido em ~/.network_agent/history. Corrija ou remova o arquivo e recarregue a lista.")
	default:
		return newBridgeError(codeIO,
			"Falha ao acessar o histórico em disco",
			err.Error(),
			"Verifique permissões e espaço em ~/.network_agent/history e tente de novo.")
	}
}

// ---------- superfície do histórico ----------

// ListConversations devolve todas as conversas, com o transcript junto, da mais
// recente para a mais antiga (updatedAt desc).
//
// A lista é gorda de propósito (ticket 05): o Store lê e desserializa todos os
// arquivos de todo modo, e uma leitura só mantém sidebar e chat de acordo — sem
// um Get separado. Diretório que ainda não existe é lista vazia, não erro: na
// primeira execução o histórico está legitimamente vazio.
func (a *App) ListConversations() ([]history.Conversation, error) {
	convs, err := a.store.List()
	if err != nil {
		return nil, errHistory(err)
	}
	for i := range convs {
		convs[i] = history.ProjectConversation(convs[i])
	}
	return convs, nil
}

// CreateConversation cria a conversa vazia com o título do front-end. Provedor
// e modelo saem da configuração ativa — o front-end mandando o par criaria uma
// segunda fonte para o mesmo fato (ticket 05) — e são os da última conexão,
// gravados junto com o agente em reconnect.
func (a *App) CreateConversation(title string) (history.Conversation, error) {
	if strings.TrimSpace(title) == "" {
		return history.Conversation{}, newBridgeError(codeInvalid,
			"Título vazio",
			"a conversa precisa de um título para aparecer na lista",
			"Escreva um título para a conversa.")
	}

	act := a.activeConfig()
	if act.Provider == "" {
		return history.Conversation{}, a.errNoAgent()
	}

	conv, err := a.store.Create(history.NewConversation{
		Title:    title,
		Provider: act.Provider,
		Model:    act.Model,
	})
	if err != nil {
		return history.Conversation{}, errHistory(err)
	}
	return conv, nil
}

// RenameConversation troca o título e devolve a conversa atualizada.
func (a *App) RenameConversation(id, title string) (history.Conversation, error) {
	if strings.TrimSpace(title) == "" {
		return history.Conversation{}, newBridgeError(codeInvalid,
			"Título vazio",
			"a conversa precisa de um título para aparecer na lista",
			"Escreva um título para a conversa.")
	}

	conv, err := a.store.Rename(id, title)
	if err != nil {
		return history.Conversation{}, errHistory(err)
	}
	return history.ProjectConversation(conv), nil
}

// DeleteConversation apaga a conversa do disco.
func (a *App) DeleteConversation(id string) error {
	if err := a.store.Delete(id); err != nil {
		return errHistory(err)
	}
	return nil
}

// ---------- o turno ----------

// runTurn é a composição do turno (internal/turn). É variável de pacote pelo
// mesmo motivo do seam de teste do próprio internal/turn (`var retrieval =
// retrieve`): a composição de verdade classifica duas vezes, busca no banco do
// RAG e chama o LLM, e nada disso um teste de unidade tem. A produção nunca a
// troca.
var runTurn = turn.Run

// logTurn registra o metadado do turno que NÃO vai à tela: o Subagente que
// respondeu, os filtros de busca, o tamanho do contexto recuperado e o
// resultado do scanner de Mermaid. Só o autor e os avisos ficam na mensagem (o
// usuário não infere nenhum dos dois do texto); o resto é observabilidade, e
// aqui ele existe em vez de desaparecer.
//
// Sem a chave `classificacao`: a classificação de roteamento carrega uma
// dimensão só, e o valor dela é o `subagente` desta mesma linha.
//
// Os avisos também saem no log, em Warn: a bolha os mostra no turno em que
// aconteceram, e o arquivo é onde o turno de ontem ainda pode ser lido.
func logTurn(convID string, out turn.Outcome) {
	log.Debug("turno composto",
		"conv", convID,
		"subagente", out.Subagent,
		"filtros", out.Filters,
		"contexto_chars", len(out.RecoveredContext),
		"mermaid_checado", out.MermaidChecked,
		"mermaid_blocos", len(out.MermaidBlocks),
		"mermaid_repetido", out.MermaidRetried,
	)
	for _, warning := range out.Warnings {
		log.Warn(warning, "conv", convID)
	}
}

// AskAgent executa um turno na conversa convID e devolve a conversa como ela
// ficou no disco.
//
// O front-end manda o id, o Go grava (ticket 05): o transcript que vai ao
// provedor é o que está no disco — nunca o que está na tela —, os ids e os
// tempos das mensagens são cunhados aqui, e o `usage` vem do que o provedor
// reportou. A ordem do turno é: lê o transcript, grava a mensagem do usuário
// ANTES de chamar o provedor (um turno interrompido deixa o prompt no disco),
// roda a composição (turn.Run), grava a resposta e só então transmite os
// tokens — texto que a UI mostra já está persistido.
//
// O que fica gravado na mensagem do usuário é o que ele digitou, NÃO o
// out.Prompt da composição (o prompt com o bloco <recovered_context> junto),
// que é o que o cmd/cli grava. São duas razões, e as duas são da UI:
//   - o Content da mensagem é também o que o balão renderiza, então o
//     out.Prompt despejaria o JSON do contexto recuperado dentro do balão do
//     usuário;
//   - a mensagem vai ao disco antes da chamada, e o Store é append-only: o
//     out.Prompt só existe depois, e gravá-lo exigiria trocar a ordem e perder
//     o prompt do turno interrompido.
//
// O custo é que o transcript reenviado não leva os blocos de contexto dos
// turnos passados — que é justamente o que o aceite do Roteamento mediu como
// problema (entrada de 49.731 tokens em 6 turnos, modelo repetindo a resposta
// anterior) e virou o esforço de política de transcript. O contexto do turno
// corrente nunca se perde: a skip rule reinjeta o último recuperado a cada
// turno em que a classificação não muda.
//
// O turno que falha também é gravado, com status error e o AgentError montado
// aqui (é quem tem o erro do provedor e quem grava); o reenvio descarta essa
// mensagem (ticket 04), mas ela aparece ao reabrir a conversa. O erro do turno
// chega ao front-end pela mensagem, não por exceção — a exceção fica para o que
// impediu o turno de existir: disco, conversa ausente, sem agente, ocupado.
func (a *App) AskAgent(convID, prompt string) (history.Conversation, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return history.Conversation{}, newBridgeError(codeInvalid,
			"Mensagem vazia",
			"o turno precisa de um prompt",
			"Escreva uma mensagem antes de enviar.")
	}
	if strings.TrimSpace(convID) == "" {
		return history.Conversation{}, newBridgeError(codeInvalid,
			"Nenhuma conversa informada",
			"o AskAgent grava na conversa do id recebido; criação silenciosa de conversa foi descartada (ticket 05)",
			"Crie ou abra uma conversa antes de enviar.")
	}

	// Trava contra chamadas concorrentes: se um AskAgent já estiver em
	// execução, recusa a nova chamada (a UI já desabilita o input, mas isso
	// é uma salvaguarda contra double-submit/loop).
	if !a.busyMux.TryLock() {
		return history.Conversation{}, newBridgeError(codeBusy,
			"O agente já está respondendo",
			"um turno por vez",
			"Aguarde a resposta atual terminar.")
	}
	defer a.busyMux.Unlock()

	ag := a.currentAgent()
	if ag == nil {
		return history.Conversation{}, a.errNoAgent()
	}

	// O transcript vem do disco, nunca do que está na tela.
	conv, err := a.store.Get(convID)
	if err != nil {
		return history.Conversation{}, errHistory(err)
	}

	// A mensagem do usuário é gravada antes da chamada ao provedor: um turno
	// interrompido (queda, janela fechada) deixa o prompt no disco.
	if _, err := a.store.Append(convID, history.Message{
		ID:      uuid.NewString(),
		Role:    history.RoleUser,
		Content: prompt,
		Time:    history.NowStamp(),
		Status:  history.StatusDone,
	}); err != nil {
		return history.Conversation{}, errHistory(err)
	}

	log.Debug("turno iniciado", "conv", convID, "prompt", prompt)
	a.emitStatus("Processando")

	started := time.Now()

	// A janela de contexto é sondada JUNTO do turno, não antes dele: o provedor
	// que não responde à sondagem não atrasa a resposta, e o número gravado é o
	// do modelo como ele está agora — o provedor pode ter recarregado o modelo
	// desde o turno anterior (ticket 11). O canal tem buffer para a goroutine
	// nunca ficar presa, mesmo no turno que falha.
	window := make(chan contextWindow, 1)
	go func() {
		total, source := ag.ContextWindow(a.ctx)
		window <- contextWindow{Total: total, Source: source}
	}()

	// O turno inteiro é a composição (classificação genérica + roteamento,
	// Modo RAG com skip rule, retrieval, <recovered_context>, scanner de
	// Mermaid, chamada ao LLM). Daqui para baixo este método só persiste e
	// transmite o que ela devolveu.
	out, state, err := runTurn(a.ctx, ag, history.ProjectTranscript(conv.Messages), prompt, a.loadTurnState(convID))

	// O State é guardado ANTES de olhar o erro: ele vale nas duas saídas do
	// turn.Run, e perdê-lo num turno que falhou faria o seguinte re-disparar
	// o retrieval sem motivo.
	a.saveTurnState(convID, state)
	logTurn(convID, out)

	if err != nil {
		return a.recordFailedTurn(convID, err, out.Warnings)
	}
	elapsed := time.Since(started)
	win := <-window

	// Usage fica na mensagem do agente: o Store é append-only e a mensagem do
	// usuário já foi gravada antes da chamada, quando o uso ainda não existia.
	agentMsg := history.Message{
		ID:        uuid.NewString(),
		Role:      history.RoleAgent,
		Content:   out.Text,
		Time:      history.NowStamp(),
		Status:    history.StatusDone,
		ElapsedMs: int(elapsed.Milliseconds()),
		Usage:     projectUsage(out.Usage, win),
		Subagent:  out.Subagent,
		Warnings:  out.Warnings,
	}
	if agentMsg.Usage != nil {
		agentMsg.Tokens = agentMsg.Usage.Output
	}

	// Grava antes de transmitir: se o disco recusar a escrita, o turno falha
	// com erro claro em vez de a UI mostrar texto que não foi persistido.
	conv, err = a.store.Append(convID, agentMsg)
	if err != nil {
		return history.Conversation{}, errHistory(err)
	}

	// Streaming simulado dos tokens da resposta final (o Chat devolve a
	// resposta inteira de uma vez). O evento leva a conversa e a mensagem — o
	// id cunhado acima, o mesmo que foi para o disco — para a UI descartar
	// token de uma conversa que não está mais aberta; o turno segue até o fim
	// de qualquer jeito, e o que ele mostra já está gravado.
	for _, chunk := range streamChunks(agentMsg.Content) {
		a.emit("agent:token", map[string]string{
			"token":          chunk,
			"conversationId": convID,
			"messageId":      agentMsg.ID,
		})
		time.Sleep(18 * time.Millisecond)
	}
	a.emit("agent:done", map[string]string{
		"conversationId": convID,
		"messageId":      agentMsg.ID,
	})

	return history.ProjectConversation(conv), nil
}

// recordFailedTurn grava o turno que falhou: a resposta do agente vira uma
// mensagem de status error com o AgentError montado no Go, e a conversa volta
// como ficou no disco — o front-end mostra o erro pela mensagem. Se nem isso o
// disco aceitar, o erro de disco é o que sobe.
//
// Os avisos de degradação do turno vão junto: a composição já pode ter
// degradado (classificação que falhou, retrieval que não veio) antes de a
// chamada ao LLM matar o turno, e esses avisos explicam o que o turno tentou.
func (a *App) recordFailedTurn(convID string, cause error, warnings []string) (history.Conversation, error) {
	conv, err := a.store.Append(convID, history.Message{
		ID:       uuid.NewString(),
		Role:     history.RoleAgent,
		Time:     history.NowStamp(),
		Status:   history.StatusError,
		Warnings: warnings,
		Error: &history.AgentError{
			What: "O provedor não completou o turno",
			Why:  cause.Error(),
			Next: "Confira se o provedor está no ar e se o modelo configurado existe; depois reenvie a mensagem.",
		},
	})
	if err != nil {
		log.Error("falha ao gravar o turno que falhou", "conv", convID, "err", err, "cause", cause)
		return history.Conversation{}, errHistory(err)
	}
	return history.ProjectConversation(conv), nil
}

// projectUsage projeta o uso que o provedor reportou no TokenUsage persistido.
// "Não reportou" e zero não são a mesma coisa: o SDK marca campo presente em
// JSON.*.Valid, e sem nenhum uso reportado o valor é nil — o disco não recebe
// uma medida que ninguém mediu. O contexto usado é o que foi enviado ao
// provedor no payload, o número que o medidor da UI compara com a janela; a
// janela e a origem dela vêm da sondagem do modelo ativo (ContextWindow), para
// o medidor não exibir um número declarado como se fosse medido.
func projectUsage(u responses.ResponseUsage, win contextWindow) *history.TokenUsage {
	in, out, total := u.JSON.InputTokens, u.JSON.OutputTokens, u.JSON.TotalTokens
	if !in.Valid() && !out.Valid() && !total.Valid() {
		return nil
	}
	return &history.TokenUsage{
		Input:         int(u.InputTokens),
		Output:        int(u.OutputTokens),
		Total:         int(u.TotalTokens),
		ContextUsed:   int(u.InputTokens),
		ContextTotal:  int(win.Total),
		ContextSource: win.Source,
	}
}

// streamChunks divide a resposta em pedaços preservando TODO o whitespace
// (quebras de linha, linhas em branco e indentação). Isso é essencial para o
// markdown: strings.Fields descartaria os "\n", transformando parágrafos,
// listas e tabelas em uma única linha corrida.
//
// Cada pedaço é uma palavra junto com o whitespace que a segue, de modo que
// a concatenação de todos os pedaços reproduz a string original byte a byte.
func streamChunks(s string) []string {
	var chunks []string
	i := 0
	for i < len(s) {
		start := i
		// avança pelo texto visível
		for i < len(s) && !isSpace(s[i]) {
			i++
		}
		// avança pelo whitespace que o segue (inclui \n e \n\n)
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		chunks = append(chunks, s[start:i])
	}
	return chunks
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// selectedModel devolve o provedor e o modelo ativos (o provedor selecionado e
// o modelo padrão dele). Espera uma configuração normalizada — Normalize mantém
// Selected e DefaultModel dentro dos limites — e devolve ok=false quando não há
// provedor, ou o provedor não tem modelo.
func selectedModel(cfg types.Config) (types.Provider, types.Model, bool) {
	if len(cfg.Providers) == 0 {
		return types.Provider{}, types.Model{}, false
	}
	provider := cfg.Providers[cfg.Selected]
	if len(provider.Models) == 0 {
		return types.Provider{}, types.Model{}, false
	}
	return provider, provider.Models[provider.DefaultModel], true
}
