package main

import (
	"cmp"
	"context"
	"sync"

	"github.com/Gabriel-Araujo/network_agent/internal/agent"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/turn"
	"github.com/Gabriel-Araujo/network_agent/internal/types"
	"github.com/wailsapp/wails/v3/pkg/application"
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

	// emit publica um evento para a UI. É um campo, e não o Event.Emit do app
	// direto nos métodos, para que o teste do turno observe os eventos sem a
	// ponte do Wails, que fora do application.New não existe.
	emit func(eventName string, data ...any)

	// conn é o estado da conexão com o provedor, para a UI dizer o que está
	// acontecendo em vez de deduzir da ausência do agente. Anda sob o mesmo
	// agentMux: é o mesmo fato visto de outro lado.
	conn connection

	// connGen numera as tentativas de conexão. A carga do modelo leva dezenas
	// de segundos, então duas gravações seguidas nas Configurações deixam duas
	// tentativas em voo: a que chega atrasada e não é mais a corrente é
	// descartada, para o agente ativo nunca ser o de uma configuração vencida.
	connGen uint64

	// agentMux protege a troca do agente quando a configuração muda.
	agentMux sync.RWMutex

	// busy trava a execução do agente: apenas UMA chamada AskAgent por vez.
	// Garante que cada input do usuário seja processado uma única vez e
	// que o próximo input aguarde o término do atual.
	busy           bool
	busyMux        sync.Mutex
	repairAttempts map[string]string

	// turnState guarda o State da composição do turno por Conversa. A skip
	// rule do Modo RAG compara a classificação com a do turno anterior, então
	// o State precisa sobreviver entre chamadas do AskAgent — e por Conversa,
	// porque é a Conversa que tem turno anterior. Fica em memória de
	// propósito: a economia de retrieval é dentro de uma sessão de trabalho,
	// não entre reinícios do app, e no disco poria um campo opaco no
	// histórico. Zera ao fechar o app.
	turnState    map[string]turn.State
	turnStateMux sync.Mutex
}

// activeModel é o que a última conexão sabe da configuração ativa: o nome do
// provedor e o id do modelo (a string que vai ao provedor).
type activeModel struct {
	Provider string
	Model    string
}

// Os quatro estados da conexão com o provedor. O agente nil tinha três causas
// e uma única mensagem — "cadastre um provedor" —, que o drive do ticket 07
// mediu apontando para o lugar errado justamente quando a configuração estava
// certa e o modelo estava carregando.
const (
	connNone       = "none"       // não há provedor e modelo configurados
	connConnecting = "connecting" // conectando (a carga do modelo leva dezenas de segundos)
	connFailed     = "failed"     // a conexão falhou, e `Cause` diz por quê
	connReady      = "ready"      // há agente ativo
)

// connection é o estado da conexão como o front-end o lê. `Cause` só existe no
// estado failed: é o erro do Connect, cru, porque a causa que o usuário precisa
// ver (servidor fora do ar, modelo inexistente, janela recusada) vem de dentro
// dele.
type connection struct {
	State string `json:"state"`
	Cause string `json:"cause,omitempty"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{}

	// O histórico vive ao lado do config.json, em ~/.network_agent/history.
	a.store = history.NewFileStore(history.DefaultDir())

	// A ponte de eventos de produção. O app do Wails nasce depois do NewApp,
	// então o closure o busca na hora da chamada.
	a.emit = func(eventName string, data ...any) {
		application.Get().Event.Emit(eventName, data...)
	}

	// Conecta a partir do arquivo de configuração (~/.network_agent/config.json),
	// o mesmo usado pelo cmd/cli — e não espera a conexão terminar: ela roda em
	// background, porque NewApp acontece ANTES do Run do Wails e uma carga de
	// modelo de dezenas de segundos aqui seria um duplo-clique sem janela
	// nenhuma na tela (ticket 13).
	a.Reconnect()
	return a
}

// connect monta o agente da configuração. É variável de pacote pelo mesmo
// motivo do runTurn: o Connect de verdade carrega o modelo no provedor, e o
// teste que prova que o Workspace chega ao Agent não pode depender de um LM
// Studio no ar.
var connect = agent.Connect

// reconnect troca o agente ativo conforme a configuração e atualiza o par
// provedor/modelo que a conversa nova grava. É chamado no startup e a cada
// gravação vinda da tela de Configurações.
//
// Não bloqueia: o Connect alarga a janela do modelo no provedor, o que em LM
// Studio é descarregar e carregar de novo — 19s medidos no menor dos modelos
// locais. A conexão corre numa goroutine e só o *resultado* entra sob o lock;
// conectar com o agentMux na mão travaria por esses mesmos segundos todo
// currentAgent(), que é o caminho do turno e do medidor.
func (a *App) reconnect(cfg types.Config) {
	provider, model, ok := selectedModel(cfg)
	if !ok {
		a.setConnection(connection{State: connNone}, nil, activeModel{})
		return
	}

	a.agentMux.Lock()
	a.connGen++
	gen := a.connGen
	// O agente da configuração anterior sai de cena agora: enquanto o novo
	// carrega, responder pelo antigo seria responder com o modelo que o
	// usuário acabou de trocar.
	a.agent, a.active, a.conn = nil, activeModel{}, connection{State: connConnecting}
	a.agentMux.Unlock()

	go func() {
		// O Workspace vem da configuração: é a pasta que o usuário escolheu em
		// Configurações, e vazia significa "nenhuma escolhida" — estado que as
		// file tools recusam com texto próprio (ticket 05, §3), não falha
		// silenciosa.
		newAgent, err := connect(cfg, cfg.Workspace)

		a.agentMux.Lock()
		defer a.agentMux.Unlock()
		if gen != a.connGen {
			log.Info("conexão descartada: a configuração mudou enquanto o modelo carregava",
				"provider", provider.Name, "model", model.Id)
			return
		}
		if err != nil {
			log.Error("falha ao conectar ao provedor", "provider", provider.Name, "err", err)
			a.agent, a.active, a.conn = nil, activeModel{}, connection{State: connFailed, Cause: err.Error()}
			return
		}
		a.agent = newAgent
		a.active = activeModel{Provider: provider.Name, Model: model.Id}
		a.conn = connection{State: connReady}
	}()
}

// setConnection grava o estado da conexão junto do agente e do par ativo, que
// são o mesmo fato: não existe agente nos estados que não são `ready`.
func (a *App) setConnection(conn connection, ag *agent.Agent, active activeModel) {
	a.agentMux.Lock()
	defer a.agentMux.Unlock()
	a.connGen++
	a.agent, a.active, a.conn = ag, active, conn
}

// Reconnect refaz a conexão a partir da configuração do disco e devolve o
// estado em que ela ficou — `connecting`, no caminho normal, porque a conexão
// é background. É o startup e também a ação explícita do aviso de falha: sem
// ela, o único jeito de tentar de novo era salvar as Configurações outra vez,
// que é o que a mensagem errada vinha induzindo o usuário a fazer.
//
// Retentativa automática ficou de fora de propósito: ela esconderia a causa,
// e com o provedor fora do ar seria um laço de carga de modelo disparado sem
// ninguém pedir.
func (a *App) Reconnect() connection {
	cfg, err := readConfig()
	if err != nil {
		log.Error("falha ao ler a configuração", "path", config.Path(), "err", err)
		a.setConnection(connection{State: connFailed, Cause: err.Error()}, nil, activeModel{})
		return a.GetConnection()
	}
	config.Normalize(&cfg)
	a.reconnect(cfg)
	return a.GetConnection()
}

// GetConnection devolve o estado da conexão para o front-end, que o busca no
// boot e repete enquanto for `connecting`.
//
// É pull, e não um evento: a goroutine da conexão pode terminar antes de o
// startup gravar o a.ctx, e o runtime do Wails com ctx nil chama log.Fatalf —
// o push mataria o app exatamente no arranque que este ticket conserta
// (medido no ticket 06).
func (a *App) GetConnection() connection {
	a.agentMux.RLock()
	defer a.agentMux.RUnlock()
	return connection{State: cmp.Or(a.conn.State, connNone), Cause: a.conn.Cause}
}

// currentAgent devolve o agente ativo sob lock de leitura.
func (a *App) currentAgent() *agent.Agent {
	a.agentMux.RLock()
	defer a.agentMux.RUnlock()
	return a.agent
}

// loadTurnState devolve o State da composição do turno da Conversa. O zero
// value é o primeiro turno dela, que é o que uma Conversa desconhecida é.
func (a *App) loadTurnState(convID string) turn.State {
	a.turnStateMux.Lock()
	defer a.turnStateMux.Unlock()
	return a.turnState[convID]
}

// saveTurnState guarda o State que a composição devolveu. É chamado também no
// turno que falhou: o turn.Run devolve o State atualizado nas duas saídas, e
// perdê-lo faria o turno seguinte re-disparar o retrieval sem motivo.
func (a *App) saveTurnState(convID string, state turn.State) {
	a.turnStateMux.Lock()
	defer a.turnStateMux.Unlock()
	if a.turnState == nil {
		a.turnState = map[string]turn.State{}
	}
	a.turnState[convID] = state
}

// activeConfig devolve o snapshot do provedor/modelo ativos sob lock de leitura.
func (a *App) activeConfig() activeModel {
	a.agentMux.RLock()
	defer a.agentMux.RUnlock()
	return a.active
}

// ServiceStartup é o hook de serviço do Wails v3: o ctx vale enquanto o app
// roda e é o dos turnos, da janela de contexto e do reparo de diagrama. Não
// vira binding (o v3 exclui os métodos de ciclo de vida).
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	return nil
}

func (a *App) emitStatus(label string) {
	a.emit("agent:status", map[string]string{"label": label})
}
