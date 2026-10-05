package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx   context.Context
	agent *agent.Agent

	// store é o núcleo do histórico em disco (~/.network_agent/history). Nada é
	// criado no disco antes da primeira gravação, então a primeira execução
	// sobe sem diretório de histórico.
	store history.Store

	// active é o que a última conexão sabe da configuração ativa: o provedor e
	// o modelo que a conversa nova grava. Anda junto do agente, sob o mesmo
	// mutex, porque sai da mesma configuração que o construiu. A janela de
	// contexto do medidor não mora aqui: ela é do modelo ativo, e é sondada no
	// provedor a cada pergunta da UI (GetContextWindow).
	active activeModel

	// emit publica um evento para a UI. É um campo, e não runtime.EventsEmit
	// direto nos métodos, para que o teste do turno observe os eventos sem a
	// ponte do Wails — com um contexto que não veio dos hooks do app, o runtime
	// real encerra o processo.
	emit func(eventName string, data ...any)

	// agentMux protege a troca do agente quando a configuração muda.
	agentMux sync.RWMutex

	// busy trava a execução do agente: apenas UMA chamada AskAgent por vez.
	// Garante que cada input do usuário seja processado uma única vez e
	// que o próximo input aguarde o término do atual.
	busy    bool
	busyMux sync.Mutex
}

// activeModel é o que a última conexão sabe da configuração ativa: o nome do
// provedor e o id do modelo (a string que vai ao provedor).
type activeModel struct {
	Provider string
	Model    string
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{}

	// O histórico vive ao lado do config.json, em ~/.network_agent/history.
	a.store = history.NewFileStore(history.DefaultDir())

	// A ponte de eventos de produção; o ctx chega no startup e o closure lê o
	// campo na hora da chamada.
	a.emit = func(eventName string, data ...any) {
		runtime.EventsEmit(a.ctx, eventName, data...)
	}

	// Conecta a partir do arquivo de configuração (~/.network_agent/config.json),
	// o mesmo usado pelo cmd/cli. Diferente de config.Load(), loadConfig() não
	// abre o criador interativo por stdin — numa GUI isso travaria o app. Sem
	// provedor configurado o agente fica nil e a UI orienta a configurar.
	cfg, err := readConfig()
	if err != nil {
		fmt.Printf("[config] falha ao ler a configuração: %v\n", err)
		return a
	}
	config.Normalize(&cfg)
	a.reconnect(cfg)
	return a
}

// reconnect troca o agente ativo conforme a configuração e atualiza o par
// provedor/modelo/contexto que a conversa nova grava. É chamado no startup e a
// cada gravação vinda da tela de Configurações.
func (a *App) reconnect(cfg types.Config) {
	a.agentMux.Lock()
	defer a.agentMux.Unlock()

	provider, model, ok := selectedModel(cfg)
	if !ok {
		a.agent = nil
		a.active = activeModel{}
		return
	}

	newAgent, err := agent.Connect(cfg)
	if err != nil {
		fmt.Printf("[config] falha ao conectar ao provedor %q: %v\n", provider.Name, err)
		a.agent = nil
		a.active = activeModel{}
		return
	}
	a.agent = newAgent
	a.active = activeModel{
		Provider: provider.Name,
		Model:    model.Id,
	}
}

// currentAgent devolve o agente ativo sob lock de leitura.
func (a *App) currentAgent() *agent.Agent {
	a.agentMux.RLock()
	defer a.agentMux.RUnlock()
	return a.agent
}

// activeConfig devolve o snapshot do provedor/modelo ativos sob lock de leitura.
func (a *App) activeConfig() activeModel {
	a.agentMux.RLock()
	defer a.agentMux.RUnlock()
	return a.active
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) emitStatus(label string) {
	a.emit("agent:status", map[string]string{"label": label})
}
