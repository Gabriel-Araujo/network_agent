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
  ContextWindow,
  Conversation,
  Message,
  TokenUsage,
  activeModel,
  formatTs,
  parseBridgeError,
  sortByUpdatedDesc,
  toUiMessages,
} from './types'

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
  const [blockedMsg, setBlockedMsg] = useState('')
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
    const a = (window as any).go?.main?.App
    if (!a?.ListConversations) return
    try {
      const list: Conversation[] = await a.ListConversations()
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
    const a = (window as any).go?.main?.App
    if (!a?.GetContextWindow) return
    try {
      const win: ContextWindow = await a.GetContextWindow()
      setContextWindow(win && win.total > 0 ? win : null)
    } catch {
      setContextWindow(null)
    }
  }, [])

  useEffect(() => {
    loadConversations()
    refreshContextWindow()
    const w = window as any
    // Carrega a configuração no boot. Na primeira execução (sem provedor ou
    // sem modelo) abre a tela de Configurações e bloqueia o input.
    if (w.go?.main?.App?.GetConfig) {
      w.go.main.App.GetConfig()
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
  }, [loadConversations, refreshContextWindow])

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
        // de entrada rodaria de novo sobre texto que já estava visível.
        for (let i = 0; i < prev.length && i < disk.length; i++) {
          if (prev[i].role === disk[i].role && prev[i].content === disk[i].content) {
            disk[i] = { ...disk[i], id: prev[i].id }
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

  // Streaming do agente (Go -> UI). Os eventos levam a conversa e a mensagem;
  // só estes dois têm produtor no Go — o `agent:toolcall` escutado aqui nunca
  // teve produtor e saiu (tool calls da UI vêm do transcript do disco).
  useEffect(() => {
    const w = window as any
    if (!w.runtime?.EventsOn) return
    w.runtime.EventsOn('agent:token', (d: any) => appendToken(d))
    w.runtime.EventsOn('agent:done', (d: any) => onAgentDone(d))
    return () => {
      w.runtime.EventsOff('agent:token', 'agent:done')
    }
  }, [appendToken, onAgentDone])

  const send = async (raw: string) => {
    const text = raw.trim()
    if (!text || isWorking || sendingRef.current) return
    // Sem provedor/modelo configurado não há para onde enviar: bloqueia e
    // explica, em vez de deixar o backend falhar com erro genérico.
    if (configReady === false) {
      setBlockedMsg(
        !config || config.providers.length === 0
          ? 'Nenhum provedor configurado. Cadastre um provedor com ao menos um modelo para conversar com o agente.'
          : 'O provedor selecionado não tem nenhum modelo cadastrado. Adicione um modelo para conversar com o agente.',
      )
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

    const app = (window as any).go?.main?.App
    if (!app?.AskAgent) {
      // Sem a ponte do Wails não existe turno real. O stream simulado que
      // morava aqui mostrava número que ninguém mediu e texto que ninguém
      // gravou; agora a ausência da ponte falha visível.
      failTurn({
        code: '',
        what: 'Sem conexão com o aplicativo',
        why: 'esta página não está rodando dentro do app do Wails (window.go ausente)',
        next: 'Abra o app pela janela do Wails (wails dev) em vez de um navegador.',
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
        const created: Conversation = await app.CreateConversation(generateTitle(text))
        convId = created.id
        selectConversation(created.id)
        setChatTitle(created.title)
        setConversations((prev) => sortByUpdatedDesc([created, ...prev]))
        setSidebarError(null)
      }
      streamingConvId.current = convId

      // O Go lê o transcript do disco, grava o turno e devolve a conversa
      // como ela ficou; o stream (agent:token/agent:done) chega no meio.
      const conv: Conversation = await app.AskAgent(convId, text)
      applyConversation(conv)
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
      const a = (window as any).go?.main?.App
      if (!a?.RenameConversation) return false
      try {
        const updated: Conversation = await a.RenameConversation(id, title)
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
    const a = (window as any).go?.main?.App
    if (!a?.DeleteConversation) return
    try {
      await a.DeleteConversation(target.id)
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
      // A configuração mudou: outro provedor/modelo tem outra janela de contexto.
      refreshContextWindow()
    },
    [refreshContextWindow],
  )

  // Provedor + modelo exibidos no topo (vindos das Configurações).
  const active = useMemo(() => activeModel(config), [config])

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

      {blockedMsg && (
        <div className="alert-scrim" role="presentation" onMouseDown={() => setBlockedMsg('')}>
          <div
            className="alert"
            role="alertdialog"
            aria-modal="true"
            aria-label="Configuração necessária"
            onMouseDown={(e) => e.stopPropagation()}
          >
            <h3 className="alert__title">Configuração necessária</h3>
            <p className="alert__text">{blockedMsg}</p>
            <div className="alert__actions">
              <button className="btn-ghost" onClick={() => setBlockedMsg('')}>Fechar</button>
              <button
                className="btn-primary"
                onClick={() => {
                  setBlockedMsg('')
                  setSettingsOpen(true)
                }}
              >
                Abrir Configurações
              </button>
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
          <span className="model-pill" style={{ '--wails-draggable': 'no-drag' } as any}>
            {active ? (
              <>
                <span className="model-pill__provider">{active.provider.name}</span>
                <span className="model-pill__sep" aria-hidden="true">·</span>
                <span className="model-pill__model">{active.model.name}</span>
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
          onBlocked={() =>
            setBlockedMsg(
              !config || config.providers.length === 0
                ? 'Nenhum provedor configurado. Cadastre um provedor com ao menos um modelo para conversar com o agente.'
                : 'O provedor selecionado não tem nenhum modelo cadastrado. Adicione um modelo para conversar com o agente.',
            )
          }
        />
      </main>

      <LogDrawer open={logsOpen} onClose={() => setLogsOpen(false)} />
    </div>
  )
}

export default App