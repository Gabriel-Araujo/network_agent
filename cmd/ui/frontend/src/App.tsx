import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import './App.css'
// O diálogo de confirmação (apagar conversa) usa o visual `.alert`, que mora aqui.
import './Settings.css'
import Sidebar from './components/Sidebar'
import ChatArea from './components/ChatArea'
import InputDock from './components/InputDock'
import SettingsModal from './components/SettingsModal'
import LogDrawer from './components/LogDrawer'
import WindowControls from './components/WindowControls'
import {
  AppConfig,
  BridgeError,
  Connection,
  ContextWindow,
  Conversation,
  MermaidRepair,
  Message,
  TokenUsage,
  activeModel,
  formatTs,
  parseBridgeError,
  sortByUpdatedDesc,
  toUiMessages,
} from './types'
import { runMermaidRepair } from './mermaidRepair'
import { Events } from '@wailsio/runtime'
import { App as GoApp } from '../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { inWails } from './types'
import { ToastContainer } from '@react-spectrum/s2'

/**
 * Por que o envio está bloqueado. O alerta era fixo em "Configuração
 * necessária" com um botão para as Configurações: com a conexão em background
 * (ticket 13) isso passaria a mentir em dois dos três casos — o modelo que
 * está carregando e a conexão que falhou não se resolvem na tela de
 * configuração, que está certa.
 */
interface Blocked {
  title: string
  text: string
  cta?: 'settings' | 'reconnect'
}

// Título da conversa derivado do primeiro prompt, gravado na criação
// preguiçosa (CreateConversation). O núcleo só armazena títulos; a geração
// fica no front-end.
function generateTitle(prompt: string): string {
  const t = (prompt ?? '').trim().replace(/\s+/g, ' ')
  if (!t) return ''
  const words = t.split(' ')
  let out = ''
  for (const w of words) {
    const candidate = (out + ' ' + w).trim()
    if (candidate.length > 48) break
    out = candidate
  }
  if (out.length < t.length) out += '…'
  return out.charAt(0).toUpperCase() + out.slice(1)
}

function App() {
  const [messages, setMessages] = useState<Message[]>([])
  // Histórico real, lido do disco. Sem mock ao lado: a lista vazia é a verdade
  // da primeira execução e a falha de leitura vira o card de erro do sidebar.
  const [conversations, setConversations] = useState<Conversation[]>([])
  // Conversa aberta no chat (o React é dono do id ativo, ticket 05).
  const [activeId, setActiveId] = useState<string | null>(null)
  const [sidebarError, setSidebarError] = useState<BridgeError | null>(null)
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null)
  const [isWorking, setIsWorking] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  // A gaveta de Logs ocupa uma terceira coluna do grid, sem scrim: a conversa
  // fica usável ao lado, que é o ponto de ser ao vivo (ticket 06).
  const [logsOpen, setLogsOpen] = useState(false)
  const [config, setConfig] = useState<AppConfig | null>(null)
  // bloqueia a UI até existir provedor + modelo configurados
  const [configReady, setConfigReady] = useState<boolean | null>(null)
  // Estado da conexão com o provedor, por pull (ticket 13). `null` é "não
  // consultado" — e não "none" —, porque sem a ponte do Wails (browser puro)
  // não há conexão nenhuma para consultar, e aí o envio deve falhar visível em
  // vez de ser bloqueado por um estado que ninguém mediu.
  const [connection, setConnection] = useState<Connection | null>(null)
  const [blocked, setBlocked] = useState<Blocked | null>(null)
  const [chatTitle, setChatTitle] = useState('')
  // Janela de contexto do modelo ativo, sondada pelo Go (provedor quando ele a
  // expõe, senão o declarado — sempre com a origem). O medidor precisa dela
  // mesmo em conversa vazia: o `usage` do disco só existe depois do turno.
  const [contextWindow, setContextWindow] = useState<ContextWindow | null>(null)

  // id ativo em ref: os handlers de evento do Wails decidem na hora do
  // disparo, e um state chegaria um render atrasado.
  const activeIdRef = useRef<string | null>(null)
  // balão que está recebendo tokens na tela (some ao trocar de conversa).
  const streamingId = useRef<string | null>(null)
  // conversa dona do turno em voo: é o que descarta token de conversa fechada
  // e religa o stream quando a conversa do turno é reaberta.
  const streamingConvId = useRef<string | null>(null)
  // timer do pull do estado da conexão (só existe enquanto carregando).
  const connectionTimer = useRef<number | undefined>(undefined)
  // ref síncrono de "enviando": bloqueia double-submit no mesmo tick.
  // O isWorking (state) é assíncrono e pode vazar chamadas rápidas no
  // mesmo render; este ref garante que cada input vá UMA única vez.
  const sendingRef = useRef(false)

  // Troca a conversa ativa. Ref e state andam juntos, por um ponto único —
  // os handlers de evento leem o ref, nunca o state.
  const selectConversation = useCallback((id: string | null) => {
    activeIdRef.current = id
    setActiveId(id)
  }, [])

  // A lista é gorda (transcript junto) e vem do Store ordenada por updatedAt
  // desc. Falha de leitura — arquivo corrompido derruba a listagem inteira, por
  // decisão do ticket 06 — vira o card de erro do sidebar, com o arquivo
  // nomeado no `why`. No primeiro uso o histórico está legitimamente vazio:
  // lista vazia não é erro.
  const loadConversations = useCallback(async () => {
    if (!inWails()) return
    try {
      const list: Conversation[] = await GoApp.ListConversations()
      setConversations(list ?? [])
      setSidebarError(null)
    } catch (e) {
      setSidebarError(parseBridgeError(e))
    }
  }, [])

  // A janela de contexto vem do Go a cada pergunta: no boot, a cada gravação
  // das Configurações (outro modelo, outra URL, outro provedor) e ao fim de
  // cada turno — o provedor pode ter recarregado o modelo, e em LM Studio e
  // Ollama a janela é definida na carga. Sem sondagem (sem provedor ativo), o
  // medidor fica sem o número em vez de inventar um.
  const refreshContextWindow = useCallback(async () => {
    if (!inWails()) return
    try {
      const win: ContextWindow = await GoApp.GetContextWindow()
      setContextWindow(win && win.total > 0 ? win : null)
    } catch {
      setContextWindow(null)
    }
  }, [])

  // Estado da conexão, por pull. É pull e não evento porque a conexão termina
  // numa goroutine que pode chegar antes de a janela existir, e o runtime do
  // Wails com ctx nil derruba o app (medido no ticket 06). Repete a 500ms só
  // enquanto estiver carregando o modelo, e para sozinho em ready/failed.
  const pollConnection = useCallback(async () => {
    if (!inWails()) return
    window.clearTimeout(connectionTimer.current)
    try {
      const c = (await GoApp.GetConnection()) as Connection
      setConnection(c)
      if (c.state === 'connecting') {
        connectionTimer.current = window.setTimeout(pollConnection, 500)
      } else if (c.state === 'ready') {
        // O modelo acabou de carregar: a janela de contexto é dele.
        refreshContextWindow()
      }
    } catch {
      setConnection(null)
    }
  }, [refreshContextWindow])

  // Tentar de novo, explicitamente. Sem isto o único caminho de volta de uma
  // falha de conexão era salvar as Configurações outra vez.
  const reconnectNow = useCallback(async () => {
    if (!inWails()) return
    setConnection((await GoApp.Reconnect()) as Connection)
    pollConnection()
  }, [pollConnection])

  useEffect(() => {
    loadConversations()
    refreshContextWindow()
    pollConnection()
    // Carrega a configuração no boot. Na primeira execução (sem provedor ou
    // sem modelo) abre a tela de Configurações e bloqueia o input.
    if (inWails()) {
      GoApp.GetConfig()
        .then((c: AppConfig) => {
          setConfig(c)
          const ok = !!activeModel(c)
          setConfigReady(ok)
          if (!ok) setSettingsOpen(true)
        })
        .catch(() => {
          setConfig(null)
          setConfigReady(false)
          setSettingsOpen(true)
        })
    } else {
      // sem bindings (browser puro): não há histórico nem turno real —
      // o envio falha visível, dizendo que a ponte do Wails está ausente
      setConfigReady(true)
    }
    return () => window.clearTimeout(connectionTimer.current)
  }, [loadConversations, refreshContextWindow, pollConnection])

  // Token do turno em voo. Evento de conversa que não está mais na tela é
  // descartado: o turno segue no Go e é gravado no disco dela — reabrir a
  // conversa mostra o texto inteiro, vindo do disco.
  const appendToken = useCallback((payload: any) => {
    if (payload?.conversationId !== activeIdRef.current) return
    const msgId = streamingId.current
    if (!msgId) return
    const token = payload?.token ?? ''
    if (!token) return
    setMessages((prev) =>
      prev.map((m) => (m.id === msgId ? { ...m, content: m.content + token } : m)),
    )
  }, [])

  // Fim do stream. O texto autoritativo chega em seguida, na conversa que o
  // AskAgent devolve (a mesma que está no disco); aqui só o balão sai de
  // "streaming" — nenhuma medida é calculada no front.
  const onAgentDone = useCallback((payload: any) => {
    if (payload?.conversationId !== activeIdRef.current) return
    const msgId = streamingId.current
    if (!msgId) return
    setMessages((prev) =>
      prev.map((m) => (m.id === msgId ? { ...m, status: 'done' } : m)),
    )
  }, [])

  // Aplica a conversa como ela ficou no disco — a MESMA projeção que reabre
  // uma conversa antiga, então o turno ao vivo e a conversa reaberta são
  // indistinguíveis. A entrada da lista só é trocada se ainda existir: uma
  // conversa apagada durante o turno não ressuscita na lista.
  const applyConversation = useCallback((conv: Conversation) => {
    setConversations((prev) => {
      if (!prev.some((c) => c.id === conv.id)) return prev
      return sortByUpdatedDesc(prev.map((c) => (c.id === conv.id ? conv : c)))
    })
    if (activeIdRef.current === conv.id) {
      const disk = toUiMessages(conv.messages)
      setMessages((prev) => {
        // Balão do turno ao vivo mantém o id que já está na tela quando
        // role+conteúdo batem: trocar a chave remontaria o balão e a animação
        // de entrada rodaria de novo sobre texto que já estava visível. O id
        // persistido segue no `persistentId` — é ele que o reparo de diagrama
        // usa para falar com a ponte (o id da tela é só uma chave de render).
        for (let i = 0; i < prev.length && i < disk.length; i++) {
          if (prev[i].role === disk[i].role && prev[i].content === disk[i].content) {
            disk[i] = { ...disk[i], id: prev[i].id, persistentId: disk[i].id }
          }
        }
        return disk
      })
      setChatTitle(conv.title)
    }
  }, [])

  // O turno não pôde existir (disco, conversa sumida, sem agente, ocupado):
  // o erro estruturado do Go vira o balão do turno — ou o card do sidebar,
  // se a conversa do turno não é mais a que está na tela.
  const failTurn = useCallback((be: BridgeError) => {
    const msgId = streamingId.current
    if (!msgId) {
      setSidebarError(be)
      return
    }
    setMessages((prev) =>
      prev.map((m) =>
        m.id === msgId
          ? {
              ...m,
              status: 'error',
              error: { what: be.what, why: be.why, next: be.next },
            }
          : m,
      ),
    )
  }, [])

  // Estado do reparo de diagrama na mensagem dona da resposta. O alvo é o id
  // persistido (o id da tela é chave de render e muda no turno ao vivo); o
  // estado só existe na conversa aberta — a verdade persistida vem depois, na
  // conversa que a operação de conclusão devolve.
  const patchRepairState = useCallback((convID: string, messageID: string, repair: MermaidRepair) => {
    if (activeIdRef.current !== convID) return
    setMessages((prev) =>
      prev.map((m) =>
        (m.persistentId ?? m.id) === messageID ? { ...m, mermaidRepair: repair } : m,
      ),
    )
  }, [])

  // Streaming do agente (Go -> UI). Os eventos levam a conversa e a mensagem;
  // só estes dois têm produtor no Go — o `agent:toolcall` escutado aqui nunca
  // teve produtor e saiu (tool calls da UI vêm do transcript do disco).
  useEffect(() => {
    if (!inWails()) return
    const offToken = Events.On('agent:token', (e) => appendToken(e.data))
    const offDone = Events.On('agent:done', (e) => onAgentDone(e.data))
    return () => {
      offToken()
      offDone()
    }
  }, [appendToken, onAgentDone])

  const send = async (raw: string) => {
    const text = raw.trim()
    if (!text || isWorking || sendingRef.current) return
    // Sem agente não há para onde enviar: bloqueia e explica QUAL das causas
    // é — falta de configuração, modelo carregando ou conexão falhada —, em
    // vez de mandar o usuário a uma tela de configuração que já está certa.
    if (sendBlock) {
      setBlocked(sendBlock)
      return
    }

    // O balão do usuário e o do agente nascem na hora, com hora pelo mesmo
    // formatador da mensagem reaberta (types.ts). Os ids definitivos e o uso
    // medido chegam do Go quando o turno volta, na conversa do disco.
    const turnId = 'live-' + Date.now()
    const now = formatTs(new Date().toISOString())
    setMessages((prev) => [
      ...prev,
      { id: 'u-' + turnId, role: 'user', content: text, status: 'done', time: now },
      { id: turnId, role: 'agent', content: '', status: 'streaming', time: now },
    ])
    streamingId.current = turnId
    sendingRef.current = true
    setIsWorking(true)

    if (!inWails()) {
      // Sem a ponte do Wails não existe turno real. O stream simulado que
      // morava aqui mostrava número que ninguém mediu e texto que ninguém
      // gravou; agora a ausência da ponte falha visível.
      failTurn({
        code: '',
        what: 'Sem conexão com o aplicativo',
        why: 'esta página não está rodando dentro do app do Wails (a ponte do Wails está ausente)',
        next: 'Abra o app pela janela do Wails (wails3 dev) em vez de um navegador.',
      })
      streamingId.current = null
      setIsWorking(false)
      sendingRef.current = false
      return
    }

    try {
      let convId = activeIdRef.current
      if (!convId) {
        // Conversa preguiçosa: nasce no primeiro envio, com o título derivado
        // do primeiro prompt (o núcleo só armazena títulos — e a criação
        // silenciosa de conversa pelo Go foi descartada no ticket 05).
        const created: Conversation = await GoApp.CreateConversation(generateTitle(text))
        convId = created.id
        selectConversation(created.id)
        setChatTitle(created.title)
        setConversations((prev) => sortByUpdatedDesc([created, ...prev]))
        setSidebarError(null)
      }
      streamingConvId.current = convId

      // O Go lê o transcript do disco, grava o turno e devolve a conversa
      // como ela ficou; o stream (agent:token/agent:done) chega no meio.
      const conv: Conversation = await GoApp.AskAgent(convId, text)
      applyConversation(conv)

      // Reparo automático de diagrama (UMA tentativa): só para a resposta que
      // este turno acabou de concluir — nunca ao reabrir histórico, durante o
      // streaming, nem para resposta de outro reparo —, e sem mensagem de
      // usuário fictícia. Um falha do reparo NUNCA derruba o turno: a resposta
      // original e o fonte do bloco continuam na tela.
      const fresh = [...conv.messages].reverse().find((m) => m.role === 'agent')
      if (fresh && GoApp.StartMermaidRepair && GoApp.FinishMermaidRepair) {
        const messageID = fresh.id
        try {
          const updated = await runMermaidRepair(
            conv.id,
            messageID,
            fresh.content,
            {
              startRepair: (mid, failures) => GoApp.StartMermaidRepair(conv.id, mid, failures),
              retryRepair: (attemptID, failures) =>
                GoApp.RetryMermaidRepair(conv.id, attemptID, failures),
              finishRepair: (attemptID, accepted) =>
                GoApp.FinishMermaidRepair(conv.id, attemptID, accepted),
            },
            (repair) => patchRepairState(conv.id, messageID, repair),
          )
          if (updated) applyConversation(updated)
        } catch (e) {
          patchRepairState(conv.id, messageID, {
            attemptId: '',
            messageId: messageID,
            state: 'failed',
            error: parseBridgeError(e).why,
          })
        }
      }
    } catch (e) {
      const be = parseBridgeError(e)
      // A conversa sumiu do disco no meio do turno: além do erro, atualiza a
      // lista — ela pode ter sido apagada em outro lugar.
      if (be.code === 'not_found') loadConversations()
      failTurn(be)
    } finally {
      streamingId.current = null
      streamingConvId.current = null
      setIsWorking(false)
      sendingRef.current = false
      // O turno acaba de sondar o modelo ativo (e pode tê-lo recarregado no
      // provedor): o medidor do rodapé relê a janela.
      refreshContextWindow()
    }
  }

  const newChat = () => {
    setMessages([])
    selectConversation(null)
    streamingId.current = null
    // `streamingConvId` fica de pé: um turno em voo segue no Go e é gravado na
    // conversa dele — reabri-la religa o stream na tela. `isWorking` também
    // fica: um turno por vez, enquanto ele não terminar.
    setChatTitle('')
  }

  // Abrir uma conversa é ler o que a lista gorda já entregou (ticket 05): sem
  // Get separado e sem ida extra ao disco. O transcript na tela é o do disco, e
  // o tempo de cada mensagem passa pelo mesmo formatador da mensagem ao vivo.
  // Se esta é a conversa dona do turno em voo, um balão de streaming volta à
  // tela para os tokens que ainda chegarem — o disco é a fonte no fim.
  const openConversation = (c: Conversation) => {
    selectConversation(c.id)
    const ui = toUiMessages(c.messages)
    streamingId.current = null
    if (streamingConvId.current === c.id) {
      const liveId = 'live-' + c.id
      ui.push({ id: liveId, role: 'agent', content: '', status: 'streaming' })
      streamingId.current = liveId
    }
    setMessages(ui)
    setChatTitle(c.title)
  }

  const renameConversation = useCallback(
    async (id: string, title: string): Promise<boolean> => {
      if (!inWails()) return false
      try {
        const updated: Conversation = await GoApp.RenameConversation(id, title)
        // Renomear toca o updatedAt no Store: a lista reordena e a conversa sobe.
        setConversations((prev) =>
          sortByUpdatedDesc(prev.map((c) => (c.id === id ? updated : c))),
        )
        setSidebarError(null)
        if (activeId === id) setChatTitle(updated.title)
        return true
      } catch (e) {
        setSidebarError(parseBridgeError(e))
        return false
      }
    },
    [activeId],
  )

  // Apagar passa pela confirmação (pendingDelete) antes de tocar no disco.
  const confirmDelete = async () => {
    const target = pendingDelete
    setPendingDelete(null)
    if (!target) return
    if (!inWails()) return
    try {
      await GoApp.DeleteConversation(target.id)
      setConversations((prev) => prev.filter((c) => c.id !== target.id))
      setSidebarError(null)
      // Apagar a conversa aberta devolve a tela ao estado vazio.
      if (activeId === target.id) newChat()
    } catch (e) {
      setSidebarError(parseBridgeError(e))
    }
  }

  // Esc fecha a confirmação — mesmo comportamento do modal de configuração.
  useEffect(() => {
    if (!pendingDelete) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setPendingDelete(null)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [pendingDelete])

  // Identidades estáveis para o modal de configurações. Props inline geravam um
  // `load` novo a cada render do App, o que reexecutava o efeito de foco do modal
  // (roubando o foco dos inputs) e alimentava um loop de GetConfig.
  const closeSettings = useCallback(() => setSettingsOpen(false), [])
  const handleConfigChanged = useCallback(
    (c: AppConfig) => {
      setConfig(c)
      setConfigReady(!!activeModel(c))
      // A configuração mudou: outro provedor/modelo tem outra janela de
      // contexto — e outra conexão, que o Go já começou a abrir e que leva o
      // tempo da carga do modelo novo.
      refreshContextWindow()
      pollConnection()
    },
    [refreshContextWindow, pollConnection],
  )

  // Provedor + modelo exibidos no topo (vindos das Configurações).
  const active = useMemo(() => activeModel(config), [config])

  // Por que o envio está bloqueado, ou null quando não está. Fonte única: o
  // dock e o alerta discordarem é a família do defeito do ticket 13 — duas
  // superfícies contando histórias diferentes sobre o mesmo agente ausente.
  const sendBlock = useMemo<Blocked | null>(() => {
    if (configReady === false || connection?.state === 'none') {
      return {
        title: 'Configuração necessária',
        text:
          !config || config.providers.length === 0
            ? 'Nenhum provedor configurado. Cadastre um provedor com ao menos um modelo para conversar com o agente.'
            : 'O provedor selecionado não tem nenhum modelo cadastrado. Adicione um modelo para conversar com o agente.',
        cta: 'settings',
      }
    }
    if (connection?.state === 'connecting') {
      return {
        title: 'Conectando ao provedor',
        text:
          'O modelo está sendo carregado no provedor, e o agente só existe depois disso — numa carga a frio isso leva dezenas de segundos. O envio volta assim que a conexão terminar.',
      }
    }
    if (connection?.state === 'failed') {
      return {
        title: 'A conexão com o provedor falhou',
        text:
          (connection.cause || 'O provedor não respondeu.') +
          '\n\nConfira se o provedor está no ar e se o modelo configurado existe.',
        cta: 'reconnect',
      }
    }
    return null
  }, [config, configReady, connection])

  // Resumo agregado de tokens de toda a conversa, exibido no rodapé do input.
  // A janela do medidor é a de AGORA — a sondagem do modelo ativo —, não a do
  // último turno: é com ela que o próximo turno vai comparar, e é o único total
  // que existe enquanto a conversa está vazia. Sem janela sondada, fica a do
  // último turno medido no disco.
  const usageSummary = useMemo<TokenUsage>(() => {
    let input = 0
    let output = 0
    let contextUsed = 0
    let contextTotal = 0
    let contextSource: string | undefined
    for (const m of messages) {
      if (!m.usage) continue
      input += m.usage.input
      output += m.usage.output
      contextUsed = m.usage.contextUsed
      contextTotal = m.usage.contextTotal
      contextSource = m.usage.contextSource
    }
    if (contextWindow) {
      contextTotal = contextWindow.total
      contextSource = contextWindow.source
    }
    return { input, output, total: input + output, contextUsed, contextTotal, contextSource }
  }, [messages, contextWindow])

  return (
    <div
      className={`app ${sidebarCollapsed ? 'app--sidebar-collapsed' : ''} ${
        logsOpen ? 'app--logs-open' : ''
      }`}
    >
      <div className="app__bg" aria-hidden="true">
        <span className="app__glow app__glow--a" />
        <span className="app__glow app__glow--b" />
        <span className="app__glow app__glow--c" />
      </div>

      <Sidebar
        open={sidebarOpen}
        collapsed={sidebarCollapsed}
        conversations={conversations}
        activeId={activeId}
        busy={isWorking}
        error={sidebarError}
        onClose={() => setSidebarOpen(false)}
        onToggleCollapse={() => setSidebarCollapsed((c) => !c)}
        onNewChat={newChat}
        onOpenSettings={() => setSettingsOpen(true)}
        onOpenLogs={() => setLogsOpen((v) => !v)}
        onOpen={openConversation}
        onRename={renameConversation}
        onDelete={setPendingDelete}
        onReload={loadConversations}
        onDismissError={() => setSidebarError(null)}
      />

      <SettingsModal
        open={settingsOpen}
        onClose={closeSettings}
        onConfigChanged={handleConfigChanged}
        dismissible={configReady !== false}
      />

      {blocked && (
        <div className="alert-scrim" role="presentation" onMouseDown={() => setBlocked(null)}>
          <div
            className="alert"
            role="alertdialog"
            aria-modal="true"
            aria-label={blocked.title}
            onMouseDown={(e) => e.stopPropagation()}
          >
            <h3 className="alert__title">{blocked.title}</h3>
            <p className="alert__text">{blocked.text}</p>
            <div className="alert__actions">
              <button className="btn-ghost" onClick={() => setBlocked(null)}>Fechar</button>
              {blocked.cta === 'settings' && (
                <button
                  className="btn-primary"
                  onClick={() => {
                    setBlocked(null)
                    setSettingsOpen(true)
                  }}
                >
                  Abrir Configurações
                </button>
              )}
              {blocked.cta === 'reconnect' && (
                <button
                  className="btn-primary"
                  onClick={() => {
                    setBlocked(null)
                    reconnectNow()
                  }}
                >
                  Reconectar
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Confirmação de apagar conversa: mesmo visual do alerta de configuração. */}
      {pendingDelete && (
        <div className="alert-scrim" role="presentation" onMouseDown={() => setPendingDelete(null)}>
          <div
            className="alert"
            role="alertdialog"
            aria-modal="true"
            aria-label="Apagar conversa"
            onMouseDown={(e) => e.stopPropagation()}
          >
            <h3 className="alert__title">Apagar conversa</h3>
            <p className="alert__text">
              Apagar <strong>{pendingDelete.title}</strong> e o transcript dela do disco? Não dá
              para desfazer.
            </p>
            <div className="alert__actions">
              <button className="btn-ghost" autoFocus onClick={() => setPendingDelete(null)}>
                Cancelar
              </button>
              <button className="btn-danger" onClick={confirmDelete}>
                Apagar
              </button>
            </div>
          </div>
        </div>
      )}

      <main className="main">
        <header className="topbar" style={{ "--wails-draggable": "drag" } as any}>
          <button className="icon-btn topbar__menu" onClick={() => setSidebarOpen(true)} aria-label="Abrir menu" style={{ '--wails-draggable': 'no-drag' } as any}>
            <svg width="18" height="18" viewBox="0 0 256 256" fill="none" aria-hidden="true">
              <path d="M40 64h176M40 128h176M40 192h176" stroke="currentColor" strokeWidth="20" strokeLinecap="round" />
            </svg>
          </button>
          {chatTitle && (
            <div className="topbar__title">
              <span className="topbar__dot" aria-hidden="true" />
              <span className="topbar__title-text">{chatTitle}</span>
            </div>
          )}
          <div className="topbar__spacer" />
          {/* O Workspace ativo, fora de Configurações: o texto do modelo é
              onde o usuário descobre que a escrita foi recusada, e a tela é
              onde ele resolve (ticket 05, §3). Clicar abre a tela que grava. */}
          <button
            className="model-pill model-pill--action"
            style={{ '--wails-draggable': 'no-drag' } as any}
            onClick={() => setSettingsOpen(true)}
            title={config?.workspace || 'Nenhuma pasta escolhida — o agente não grava arquivos.'}
          >
            <span className="model-pill__provider">Workspace</span>
            <span className="model-pill__sep" aria-hidden="true">·</span>
            <span className="model-pill__model">{config?.workspace || 'nenhum'}</span>
          </button>
          <span className="model-pill" style={{ '--wails-draggable': 'no-drag' } as any}>
            {active ? (
              <>
                <span className="model-pill__provider">{active.provider.name}</span>
                <span className="model-pill__sep" aria-hidden="true">·</span>
                <span className="model-pill__model">{active.model.name}</span>
                {/* O estado da conexão fica onde o modelo já está: é dele que
                    se está falando, e é aqui que o usuário olha para saber se
                    pode conversar. */}
                {connection?.state === 'connecting' && (
                  <span className="model-pill__state">carregando modelo…</span>
                )}
                {connection?.state === 'failed' && (
                  <button
                    className="model-pill__state model-pill__state--failed"
                    onClick={reconnectNow}
                    title={connection.cause}
                  >
                    sem conexão · Reconectar
                  </button>
                )}
              </>
            ) : (
              'Nenhum modelo configurado'
            )}
          </span>
          <WindowControls />
        </header>

        <ChatArea messages={messages} onSuggestion={(t) => send(t)} onNewChat={newChat} isWorking={isWorking} />
        <InputDock
          onSend={send}
          isWorking={isWorking}
          usageSummary={usageSummary}
          configReady={configReady !== false}
          onBlocked={() => setBlocked(sendBlock)}
        />
      </main>

      <LogDrawer open={logsOpen} onClose={() => setLogsOpen(false)} />
      {/* Só as cópias e as falhas delas disparam toast (docs/FRONTEND.md, Início, barra de status e toast). */}
      <ToastContainer placement="bottom end" />
    </div>
  )
}

export default App