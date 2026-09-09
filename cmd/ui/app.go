package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx   context.Context
	agent *agent.Agent

	// busy trava a execução do agente: apenas UMA chamada AskAgent por vez.
	// Garante que cada input do usuário seja processado uma única vez e
	// que o próximo input aguarde o término do atual.
	busy    bool
	busyMux sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		// Valores temporários: carrega o agente de teste (cliente OpenAI + tools + system prompt).
		// A config real (provider, modelo, contexto, tools) virá da conexão do network_agent.
		agent: agent.LoadTestAgent(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AskAgent chama o método Chat do agente em loop na conversa.
//
// O mesmo *agent.Agent é reutilizado entre chamadas (o Agent guarda
// previousResponseID), então o contexto da conversa é mantido.
// A resposta final é devolvida ao frontend como string; os tokens são
// emitidos via "agent:token" para a UI renderizar em streaming.
func (a *App) AskAgent(prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt vazio")
	}

	// Trava contra chamadas concorrentes: se um AskAgent já estiver em
	// execução, recusa a nova chamada (a UI já desabilita o input, mas isso
	// é uma salvaguarda contra double-submit/loop).
	if !a.busyMux.TryLock() {
		return "", fmt.Errorf("agente já está processando uma resposta; aguarde")
	}
	defer a.busyMux.Unlock()

	fmt.Printf("[AskAgent] CHAMADO: prompt=%q\n", prompt)

	// Emite um status inicial para a UI (temporário).
	a.emitStatus("Processando")

	response, err := a.agent.Chat(a.ctx, prompt)
	if err != nil {
		return "", err
	}

	// Streaming simulado dos tokens da resposta final, para a UI renderizar
	// em tempo real (o método Chat retorna a string inteira de uma vez).
	for _, chunk := range streamChunks(response) {
		runtime.EventsEmit(a.ctx, "agent:token", map[string]string{
			"token": chunk,
			"id":    "msg-" + fmt.Sprint(time.Now().UnixMilli()),
		})
		time.Sleep(18 * time.Millisecond)
	}
	runtime.EventsEmit(a.ctx, "agent:done", map[string]string{
		"id": "msg-" + fmt.Sprint(time.Now().UnixMilli()),
	})

	return response, nil
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

func (a *App) emitStatus(label string) {
	runtime.EventsEmit(a.ctx, "agent:status", map[string]string{"label": label})
}

// ListChats returns mock conversation history for the sidebar.
func (a *App) ListChats() []map[string]string {
	return []map[string]string{
		{"id": "c1", "title": "Interface do agente de IA", "group": "Hoje"},
		{"id": "c2", "title": "Stack Wails + React", "group": "Hoje"},
		{"id": "c3", "title": "Paleta glass azulado", "group": "Ontem"},
		{"id": "c4", "title": "Prompt /impeccable", "group": "Ontem"},
		{"id": "c5", "title": "Arquitetura IPC-first", "group": "7 dias atrás"},
	}
}
