import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import './App.css'
import Sidebar from './components/Sidebar'
import ChatArea from './components/ChatArea'
import InputDock from './components/InputDock'
import SettingsModal from './components/SettingsModal'
import WindowControls from './components/WindowControls'
import { ChatMeta, FALLBACK_CHATS, Message, ToolCall, TokenUsage, AppConfig, activeModel } from './types'

const countTokens = (text: string): number => {
  const t = (text ?? '').trim()
  if (!t) return 0
  return Math.max(1, Math.round(t.length / 4))
}

// Context window simulado (128k). O agente real deve preencher a partir da API;
// o modelo configurado agora vem das Configurações (provedor + modelo).
const CONTEXT_TOTAL = 128_000

// Título gerado pelo agente a partir do primeiro prompt do usuário.
// No agente real, isso pode vir de uma chamada de modelo (ex.: "resuma esta conversa em ≤ 6 palavras").
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
  const [chats, setChats] = useState<ChatMeta[]>(FALLBACK_CHATS)
  const [isWorking, setIsWorking] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [config, setConfig] = useState<AppConfig | null>(null)
  // bloqueia a UI até existir provedor + modelo configurados
  const [configReady, setConfigReady] = useState<boolean | null>(null)
  const [blockedMsg, setBlockedMsg] = useState('')
  const [chatTitle, setChatTitle] = useState('')

  const streamingId = useRef<string | null>(null)
  const pendingTools = useRef<ToolCall[]>([])
  const startedAtRef = useRef<number>(0)
  const lastPromptRef = useRef('')
  const hasTitledRef = useRef(false)
  // ref síncrono de "enviando": bloqueia double-submit no mesmo tick.
  // O isWorking (state) é assíncrono e pode vazar chamadas rápidas no
  // mesmo render; este ref garante que cada input vá UMA única vez.
  const sendingRef = useRef(false)
  const lastFinishedId = useRef<string | null>(null)

  useEffect(() => {
    const w = window as any
    if (w.go?.main?.App?.ListChats) {
      w.go.main.App.ListChats()
        .then(setChats)
        .catch(() => setChats(FALLBACK_CHATS))
    }
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
      // sem bindings (browser puro): mantém o mock utilizável
      setConfigReady(true)
    }
  }, [])

  const appendToken = useCallback((token: string) => {
    const msgId = streamingId.current
    if (!msgId) return
    setMessages((prev) =>
      prev.map((m) => (m.id === msgId ? { ...m, content: m.content + token } : m)),
    )
  }, [])

  const finishStreaming = useCallback(
    (fullText?: string) => {
      // Guarda o id da mensagem que acabou de fechar. O backend emite
      // "agent:done" ANTES de AskAgent retornar, então a chamada seguinte
      // (finishStreaming(full), com o texto íntegro do agente) precisa
      // conseguir reescrever essa mesma mensagem, e não ser descartada.
      const msgId = streamingId.current ?? lastFinishedId.current
      if (!msgId) return
      lastFinishedId.current = msgId
      const elapsedMs = startedAtRef.current ? Date.now() - startedAtRef.current : 0
      setMessages((prev) => {
        const target = prev.find((m) => m.id === msgId)
        if (!target) return prev
        const final = fullText && fullText.trim() ? fullText : target.content
        // Uso de tokens (mock): entrada = prompt do usuário, saída = resposta,
        // contexto = soma aproximada de toda a conversa.
        const userMsg = prev.find((m) => m.role === 'user' && m.id === 'u-' + msgId)
        const inputTokens = userMsg ? countTokens(userMsg.content) : 0
        const outputTokens = countTokens(final)
        let contextUsed = inputTokens + outputTokens
        for (const m of prev) {
          if (m.id === msgId) continue
          contextUsed += m.content ? countTokens(m.content) : 0
        }
        const usage = {
          input: inputTokens,
          output: outputTokens,
          total: inputTokens + outputTokens,
          contextUsed,
          contextTotal: CONTEXT_TOTAL,
        }
        return prev.map((m) => {
          if (m.id === msgId) {
            return { ...m, content: final, status: 'done', toolCalls: pendingTools.current, elapsedMs, tokens: outputTokens }
          }
          if (m.id === 'u-' + msgId) {
            return { ...m, usage }
          }
          return m
        })
      })
      pendingTools.current = []
      streamingId.current = null
      setIsWorking(false)
      sendingRef.current = false

      // Gera o título da conversa ao final da primeira resposta do agente.
      if (!hasTitledRef.current) {
        hasTitledRef.current = true
        setChatTitle(generateTitle(lastPromptRef.current))
      }
    },
    [],
  )

  const failStreaming = useCallback((err: unknown) => {
    const msgId = streamingId.current
    if (!msgId) return
    setMessages((prev) =>
      prev.map((m) =>
        m.id === msgId
          ? {
              ...m,
              status: 'error',
              error: {
                what: 'Não consegui processar sua solicitação',
                why: String((err as Error)?.message ?? err),
                next: 'Tente reformular o pedido ou confira a conexão com o agente.',
              },
            }
          : m,
      ),
    )
    streamingId.current = null
    setIsWorking(false)
    sendingRef.current = false
  }, [])

  // Subscribe to Wails runtime events (agent -> UI streaming).
  useEffect(() => {
    const w = window as any
    if (!w.runtime?.EventsOn) return
    w.runtime.EventsOn('agent:token', (d: any) => appendToken(d?.token ?? ''))
    w.runtime.EventsOn('agent:toolcall', (tc: ToolCall) => {
      pendingTools.current = [...pendingTools.current, tc]
    })
    w.runtime.EventsOn('agent:done', () => finishStreaming())
    return () => {
      w.runtime.EventsOff('agent:token', 'agent:toolcall', 'agent:done')
    }
  }, [appendToken, finishStreaming])

  const mockStream = (prompt: string, id: string) => {
    pendingTools.current = [
      { id: 'tc-1', action: 'search', label: 'Buscando na documentação', result: '3 resultados relevantes encontrados.' },
      { id: 'tc-2', action: 'file', label: `Lendo net_agent/README.md`, result: 'Arquivo processado (1.2 KB).' },
    ]
    const text = `Entendi: "${prompt}". No app desktop o agente real executa o plano com ferramentas. Aqui está a prévia do fluxo funcionando com streaming, tool calls e estados.`
    const words = text.split(' ')
    let i = 0
    const t = window.setInterval(() => {
      if (i >= words.length) {
        clearInterval(t)
        finishStreaming()
        return
      }
      appendToken(words[i] + ' ')
      i++
    }, 22)
  }

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
    sendingRef.current = true
    lastFinishedId.current = null
    const id = 'msg-' + Date.now()
    startedAtRef.current = Date.now()
    lastPromptRef.current = text
    const now = new Date().toLocaleString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
    setMessages((prev) => [
      ...prev,
      { id: 'u-' + id, role: 'user', content: text, status: 'done', time: now },
    ])
    streamingId.current = id
    pendingTools.current = []
    setIsWorking(true)
    setMessages((prev) => [
      ...prev,
      { id, role: 'agent', content: '', status: 'streaming', time: now },
    ])

    const w = window as any
    if (w.go?.main?.App?.AskAgent) {
      try {
        const full: string = await w.go.main.App.AskAgent(text)
        finishStreaming(full)
      } catch (err) {
        failStreaming(err)
      }
    } else {
      mockStream(text, id)
    }
  }

  const newChat = () => {
    setMessages([])
    streamingId.current = null
    pendingTools.current = []
    setIsWorking(false)
    sendingRef.current = false
    hasTitledRef.current = false
    lastPromptRef.current = ''
    setChatTitle('')
  }

  // Identidades estáveis para o modal de configurações. Props inline geravam um
  // `load` novo a cada render do App, o que reexecutava o efeito de foco do modal
  // (roubando o foco dos inputs) e alimentava um loop de GetConfig.
  const closeSettings = useCallback(() => setSettingsOpen(false), [])
  const handleConfigChanged = useCallback((c: AppConfig) => {
    setConfig(c)
    setConfigReady(!!activeModel(c))
  }, [])

  // Provedor + modelo exibidos no topo (vindos das Configurações).
  const active = useMemo(() => activeModel(config), [config])

  // Resumo agregado de tokens de toda a conversa, exibido no rodapé do input.
  const usageSummary = useMemo<TokenUsage>(() => {
    let input = 0
    let output = 0
    let contextUsed = 0
    let contextTotal = CONTEXT_TOTAL
    for (const m of messages) {
      if (!m.usage) continue
      input += m.usage.input
      output += m.usage.output
      contextUsed = m.usage.contextUsed
      contextTotal = m.usage.contextTotal
    }
    return { input, output, total: input + output, contextUsed, contextTotal }
  }, [messages])

  return (
    <div className={`app ${sidebarCollapsed ? 'app--sidebar-collapsed' : ''}`}>
      <div className="app__bg" aria-hidden="true">
        <span className="app__glow app__glow--a" />
        <span className="app__glow app__glow--b" />
        <span className="app__glow app__glow--c" />
      </div>

      <Sidebar
        open={sidebarOpen}
        collapsed={sidebarCollapsed}
        chats={chats}
        onClose={() => setSidebarOpen(false)}
        onToggleCollapse={() => setSidebarCollapsed((c) => !c)}
        onNewChat={newChat}
        onOpenSettings={() => setSettingsOpen(true)}
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
    </div>
  )
}

export default App