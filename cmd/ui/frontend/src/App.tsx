import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Sidebar from './components/Sidebar'
import ChatArea from './components/ChatArea'
import Home from './components/Home'
import InputDock, { InputDockHandle } from './components/InputDock'
import SettingsModal from './components/SettingsModal'
import LogDrawer from './components/LogDrawer'
import Titlebar from './components/Titlebar'
import TabBar from './components/TabBar'
import { HOME, closeTab } from './tabs'
import StatusBar from './components/StatusBar'
import {
  AppConfig,
  BridgeError,
  Connection,
  ContextWindow,
  Conversation,
  MermaidRepair,
  Message,
  activeModel,
  formatTs,
  parseBridgeError,
  RecentWorkspace,
  sortByUpdatedDesc,
  toUiMessages,
} from './types'
import { runMermaidRepair } from './mermaidRepair'
import { Events } from '@wailsio/runtime'
import { App as GoApp } from '../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { inWails } from './types'
import { AlertDialog, DialogContainer, ToastContainer } from '@react-spectrum/s2'
import { TabPanel, Tabs } from 'react-aria-components'
import { style } from '@react-spectrum/s2/style' with { type: 'macro' }

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

/**
 * A ponte do Wails não existe no primeiro render: o Go injeta
 * `window._wails.environment` depois que a página carregou. O boot espera por
 * ela; `absent` é o browser puro (testes e dev sem Wails), onde o app abre sem
 * histórico e o envio falha visível.
 */
type Bridge = 'pending' | 'ready' | 'absent'

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

/** Uma Aba de Conversa aberta: o transcript na tela (o rascunho e a rolagem moram no painel montado). */
interface OpenTab {
  id: string
  messages: Message[]
}

/** As Abas abertas e a focada (`HOME` ou o id da Conversa), num estado só para fechar com a regra de foco. */
interface TabState {
  tabs: OpenTab[]
  focused: string
}

/** A bolha do turno ao vivo: o id da tela começa com `live-` (ver `send` e `openConversation`). */
const isLive = (m: Message | undefined) => !!m && m.role === 'agent' && m.id.startsWith('live-')

function App() {
  // Histórico real, lido do disco. Sem mock ao lado: a lista vazia é a verdade
  // da primeira execução e a falha de leitura vira o card de erro do sidebar.
  const [conversations, setConversations] = useState<Conversation[]>([])
  // As Abas: a Início fixa (fora da lista) e uma por Conversa aberta. Nada persiste.
  const [tabState, setTabState] = useState<TabState>({ tabs: [], focused: HOME })
  // Conversa dona do turno em voo (um por vez no app): o `dot` da aba e o envio
  // bloqueado nas outras. Null também enquanto a Conversa nasce pela Início.
  const [turnConvId, setTurnConvId] = useState<string | null>(null)
  const [sidebarError, setSidebarError] = useState<BridgeError | null>(null)
  const [isWorking, setIsWorking] = useState(false)
  // Sidebar expandida ou faixa recolhida: sempre abre expandida e não persiste.
  const [sidebarExpanded, setSidebarExpanded] = useState(true)
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
  // Janela de contexto do modelo ativo, sondada pelo Go (provedor quando ele a
  // expõe, senão o declarado — sempre com a origem). O medidor precisa dela
  // mesmo em conversa vazia: o `usage` do disco só existe depois do turno.
  const [contextWindow, setContextWindow] = useState<ContextWindow | null>(null)
  // O `inWails()` pode ser falso no primeiro render (o Go injeta o environment
  // depois); o boot espera a ponte em vez de abrir sem config/histórico.
  const [bridge, setBridge] = useState<Bridge>(() => (inWails() ? 'ready' : 'pending'))

  // O composer da Início, para a Nova conversa focar o campo e a sugestão
  // preencher o rascunho sem enviar.
  const homeDock = useRef<InputDockHandle>(null)
  // timer do pull do estado da conexão (só existe enquanto carregando).
  const connectionTimer = useRef<number | undefined>(undefined)
  // ref síncrono de "enviando": bloqueia double-submit no mesmo tick.
  // O isWorking (state) é assíncrono e pode vazar chamadas rápidas no
  // mesmo render; este ref garante que cada input vá UMA única vez.
  const sendingRef = useRef(false)

  // Troca as mensagens da aba da Conversa `id`, se ela estiver aberta. Conversa
  // sem aba descarta: o turno segue no Go e é gravado no disco, e reabrir mostra
  // o texto inteiro, vindo de lá.
  const patchTab = useCallback((id: string, fn: (msgs: Message[]) => Message[]) => {
    setTabState((s) => ({
      ...s,
      tabs: s.tabs.map((t) => (t.id === id ? { ...t, messages: fn(t.messages) } : t)),
    }))
  }, [])

  // Aplica `fn` à bolha do turno ao vivo da Conversa, a última da aba.
  const patchTurn = useCallback(
    (id: string, fn: (m: Message) => Message) =>
      patchTab(id, (msgs) => (isLive(msgs.at(-1)) ? [...msgs.slice(0, -1), fn(msgs.at(-1)!)] : msgs)),
    [patchTab],
  )

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

  // Espera a ponte do Wails. O Go injeta o environment depois da navegação, e
  // o efeito de boot do React pode rodar antes; sem esta espera a tela abria
  // sem config nem histórico (e sem os eventos do stream).
  useEffect(() => {
    if (bridge !== 'pending') return
    const start = Date.now()
    let timer: number | undefined
    const tick = () => {
      if (inWails()) {
        setBridge('ready')
        return
      }
      if (Date.now() - start > 3000) {
        setBridge('absent')
        return
      }
      timer = window.setTimeout(tick, 50)
    }
    tick()
    return () => window.clearTimeout(timer)
  }, [bridge])

  useEffect(() => {
    if (bridge === 'pending') return
    loadConversations()
    refreshContextWindow()
    pollConnection()
    // Carrega a configuração no boot. Na primeira execução (sem provedor ou
    // sem modelo) abre a tela de Configurações e bloqueia o input.
    if (bridge === 'ready') {
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
  }, [bridge, loadConversations, refreshContextWindow, pollConnection])

  // Token do turno em voo, na aba da Conversa dele, focada ou não.
  const appendToken = useCallback(
    (payload: any) => {
      const token = payload?.token ?? ''
      if (!token || !payload?.conversationId) return
      patchTurn(payload.conversationId, (m) =>
        m.status === 'streaming' ? { ...m, content: m.content + token } : m,
      )
    },
    [patchTurn],
  )

  // Fim do stream. O texto autoritativo chega em seguida, na conversa que o
  // AskAgent devolve (a mesma que está no disco); aqui só o balão sai de
  // "streaming" — nenhuma medida é calculada no front.
  const onAgentDone = useCallback(
    (payload: any) => {
      if (!payload?.conversationId) return
      patchTurn(payload.conversationId, (m) => (m.status === 'streaming' ? { ...m, status: 'done' } : m))
    },
    [patchTurn],
  )

  // Aplica a conversa como ela ficou no disco — a MESMA projeção que reabre
  // uma conversa antiga, então o turno ao vivo e a conversa reaberta são
  // indistinguíveis. A entrada da lista só é trocada se ainda existir: uma
  // conversa apagada durante o turno não ressuscita na lista.
  const applyConversation = useCallback(
    (conv: Conversation) => {
      setConversations((prev) => {
        if (!prev.some((c) => c.id === conv.id)) return prev
        return sortByUpdatedDesc(prev.map((c) => (c.id === conv.id ? conv : c)))
      })
      patchTab(conv.id, (prev) => {
        const disk = toUiMessages(conv.messages)
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
    },
    [patchTab],
  )

  // O turno não pôde existir (disco, conversa sumida, sem agente, ocupado): o
  // erro estruturado do Go vira a bolha do turno, na aba dele. Sem Conversa (a
  // criação pela Início falhou), não há bolha: o erro é da lista, e cai no card
  // da sidebar.
  const failTurn = useCallback(
    (convId: string | null, be: BridgeError) => {
      if (!convId) {
        setSidebarError(be)
        return
      }
      patchTurn(convId, (m) => ({
        ...m,
        status: 'error',
        error: { what: be.what, why: be.why, next: be.next },
      }))
    },
    [patchTurn],
  )

  // Estado do reparo de diagrama na mensagem dona da resposta. O alvo é o id
  // persistido (o id da tela é chave de render e muda no turno ao vivo); o
  // estado só existe na conversa aberta — a verdade persistida vem depois, na
  // conversa que a operação de conclusão devolve.
  const patchRepairState = useCallback(
    (convID: string, messageID: string, repair: MermaidRepair) =>
      patchTab(convID, (msgs) =>
        msgs.map((m) => ((m.persistentId ?? m.id) === messageID ? { ...m, mermaidRepair: repair } : m)),
      ),
    [patchTab],
  )

  // Streaming do agente (Go -> UI). Os eventos levam a conversa e a mensagem;
  // só estes dois têm produtor no Go — o `agent:toolcall` escutado aqui nunca
  // teve produtor e saiu (tool calls da UI vêm do transcript do disco).
  useEffect(() => {
    if (bridge !== 'ready') return
    const offToken = Events.On('agent:token', (e) => appendToken(e.data))
    const offDone = Events.On('agent:done', (e) => onAgentDone(e.data))
    return () => {
      offToken()
      offDone()
    }
  }, [bridge, appendToken, onAgentDone])

  // Envia pela aba `key` (a Início ou uma Conversa). Um turno por vez no app:
  // enquanto ele roda, o envio fica bloqueado em todas as abas.
  const send = async (key: string, raw: string) => {
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
    const bubbles: Message[] = [
      { id: 'u-' + turnId, role: 'user', content: text, status: 'done', time: now },
      { id: turnId, role: 'agent', content: '', status: 'streaming', time: now },
    ]
    sendingRef.current = true
    setIsWorking(true)

    if (!inWails()) {
      // Sem a ponte do Wails não existe turno real nem Conversa: a ausência da
      // ponte falha visível, no card da sidebar.
      failTurn(null, {
        code: '',
        what: 'Sem conexão com o aplicativo',
        why: 'esta página não está rodando dentro do app do Wails (a ponte do Wails está ausente)',
        next: 'Abra o app pela janela do Wails (wails3 dev) em vez de um navegador.',
      })
      setIsWorking(false)
      sendingRef.current = false
      return
    }

    let convId: string | null = key === HOME ? null : key
    try {
      if (!convId) {
        // Conversa preguiçosa: nasce no primeiro envio pela Início, com o título
        // derivado do primeiro prompt (o núcleo só armazena títulos), e abre numa
        // aba nova, focada. A Início volta vazia.
        const created: Conversation = await GoApp.CreateConversation(generateTitle(text))
        convId = created.id
        setConversations((prev) => sortByUpdatedDesc([created, ...prev]))
        setSidebarError(null)
        setTabState((s) => ({ tabs: [...s.tabs, { id: created.id, messages: bubbles }], focused: created.id }))
      } else {
        const id = convId
        patchTab(id, (msgs) => [...msgs, ...bubbles])
      }
      setTurnConvId(convId)

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
      failTurn(convId, be)
    } finally {
      setTurnConvId(null)
      setIsWorking(false)
      sendingRef.current = false
      // O turno acaba de sondar o modelo ativo (e pode tê-lo recarregado no
      // provedor): o medidor do rodapé relê a janela.
      refreshContextWindow()
    }
  }

  // Nova conversa: foca a Início e põe o cursor no composer dela, sem apagar o
  // rascunho. O campo só fica focável depois que o painel aparece.
  const newChat = () => {
    setTabState((s) => ({ ...s, focused: HOME }))
    requestAnimationFrame(() => homeDock.current?.focus())
  }

  // Abrir uma Conversa foca a aba dela, ou abre uma no fim da barra, focada. A
  // aba nova lê o que a lista gorda já entregou: sem Get separado e sem ida
  // extra ao disco. Se esta é a Conversa do turno em voo, um balão de streaming
  // volta à tela para os tokens que ainda chegarem — o disco é a fonte no fim.
  const openConversation = (c: Conversation) => {
    setTabState((s) => {
      if (s.tabs.some((t) => t.id === c.id)) return { ...s, focused: c.id }
      const ui = toUiMessages(c.messages)
      if (turnConvId === c.id) {
        ui.push({ id: 'live-' + c.id, role: 'agent', content: '', status: 'streaming' })
      }
      return { tabs: [...s.tabs, { id: c.id, messages: ui }], focused: c.id }
    })
  }

  // Ctrl+W fecha a aba focada; na Início, não faz nada.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!e.ctrlKey || e.altKey || e.shiftKey || e.key.toLowerCase() !== 'w') return
      e.preventDefault()
      setTabState((s) => (s.focused === HOME ? s : closeTab(s, s.focused)))
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

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
        return true
      } catch (e) {
        setSidebarError(parseBridgeError(e))
        return false
      }
    },
    [],
  )

  // A confirmação é o AlertDialog da Sidebar: aqui o disco já pode ser tocado.
  const deleteConversation = async (target: Conversation) => {
    if (!inWails()) return
    try {
      await GoApp.DeleteConversation(target.id)
      setConversations((prev) => prev.filter((c) => c.id !== target.id))
      setSidebarError(null)
      // Apagar fecha a aba, com a mesma regra de foco de fechar.
      setTabState((s) => closeTab(s, target.id))
    } catch (e) {
      setSidebarError(parseBridgeError(e))
    }
  }

  // O chip do Workspace: o mesmo PickWorkspace das Configurações, que já grava.
  // Cancelar devolve a configuração como está. A falha cai no card da sidebar,
  // o único lugar de erro fora de um dialog.
  const pickWorkspace = async () => {
    if (!inWails()) return
    try {
      setConfig(await GoApp.PickWorkspace())
      pollConnection()
    } catch (e) {
      setSidebarError(parseBridgeError(e))
    }
  }

  // Os recentes do menu do chip: o Go confere cada pasta no disco, a cada abertura.
  const [recents, setRecents] = useState<RecentWorkspace[]>([])
  const refreshRecents = useCallback(async () => {
    if (!inWails()) return []
    try {
      const list = await GoApp.ListRecentWorkspaces()
      setRecents(list)
      return list
    } catch {
      return []
    }
  }, [])
  useEffect(() => {
    refreshRecents()
  }, [refreshRecents, config?.workspace])

  // Trocar por um recente. Se a pasta sumiu entre abrir o menu e clicar, o Go
  // recusa e nada muda: devolve false, o menu reabre com o item desabilitado e
  // não há toast nem card. Qualquer outra falha cai no card da sidebar.
  const pickRecentWorkspace = async (path: string) => {
    if (!inWails()) return true
    try {
      setConfig(await GoApp.PickRecentWorkspace(path))
      pollConnection()
      return true
    } catch (e) {
      const list = await refreshRecents()
      if (list.some((r) => r.path === path && !r.exists)) return false
      setSidebarError(parseBridgeError(e))
      return true
    }
  }

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

  const titleOf = (id: string) => conversations.find((c) => c.id === id)?.title ?? ''
  const focused = tabState.focused
  // O título da aba do turno, para a linha "Newton está respondendo em …" nas outras abas.
  const busyIn = (key: string) => (turnConvId && key !== turnConvId ? titleOf(turnConvId) : undefined)

  // O conteúdo de uma aba: a Início (título, sugestões e o composer) ou o
  // transcript da Conversa, com o mesmo composer embaixo.
  const panel = (key: string, messages: Message[]) => (
    <TabPanel key={key} id={key} shouldForceMount className={({ isInert }) => tabPanel({ isInert })}>
      {key === HOME ? (
        <Home onSuggestion={(t) => homeDock.current?.fill(t)} />
      ) : (
        <ChatArea messages={messages} isWorking={isWorking} />
      )}
      <InputDock
        ref={key === HOME ? homeDock : undefined}
        onSend={(t) => send(key, t)}
        isWorking={isWorking}
        busyIn={busyIn(key)}
        configReady={configReady !== false}
        onBlocked={() => setBlocked(sendBlock)}
      />
    </TabPanel>
  )

  return (
    <div className={shell}>
      <Titlebar
        sidebarExpanded={sidebarExpanded}
        onToggleSidebar={() => setSidebarExpanded((v) => !v)}
        onOpenLogs={() => setLogsOpen((v) => !v)}
        onOpenSettings={() => setSettingsOpen(true)}
        workspace={config?.workspace ?? ''}
        onPickWorkspace={pickWorkspace}
        recents={recents}
        onOpenRecents={refreshRecents}
        onPickRecent={pickRecentWorkspace}
        workspaceLocked={isWorking}
        active={active ? { provider: active.provider.name, model: active.model.name } : null}
        connection={connection}
        onReconnect={reconnectNow}
      />

      <div className={columns}>
        <Sidebar
          expanded={sidebarExpanded}
          conversations={conversations}
          activeId={focused === HOME ? null : focused}
          busy={isWorking}
          error={sidebarError}
          onNewChat={newChat}
          onOpen={openConversation}
          onRename={renameConversation}
          onDelete={deleteConversation}
          onReload={loadConversations}
          onDismissError={() => setSidebarError(null)}
        />

        {/* As Abas: a barra no topo do conteúdo e um painel por aba, todos montados
            (o rascunho e a rolagem ficam neles); só o da aba focada aparece. */}
        <Tabs
          className={content}
          selectedKey={focused}
          onSelectionChange={(k) => setTabState((s) => ({ ...s, focused: String(k) }))}
        >
          <TabBar
            selected={focused}
            tabs={tabState.tabs.map((t) => ({ id: t.id, title: titleOf(t.id), busy: t.id === turnConvId }))}
            onClose={(id) => setTabState((s) => closeTab(s, id))}
          />
          <div className={panels}>
            {panel(HOME, [])}
            {tabState.tabs.map((t) => panel(t.id, t.messages))}
          </div>
        </Tabs>

        {/* A gaveta fica montada para animar a largura ao abrir e ao fechar; fechada, é inerte. */}
        <div className={logsColumn({ isOpen: logsOpen })} inert={!logsOpen}>
          <LogDrawer open={logsOpen} onClose={() => setLogsOpen(false)} />
        </div>
      </div>

      <StatusBar
        conversation={conversations.find((c) => c.id === focused) ?? null}
        contextWindow={contextWindow}
      />

      <SettingsModal
        open={settingsOpen}
        onClose={closeSettings}
        onConfigChanged={handleConfigChanged}
        dismissible={configReady !== false}
        workspaceLocked={isWorking}
      />

      {/* O envio bloqueado por configuração/conexão tem lugar próprio, não toast. */}
      <DialogContainer onDismiss={() => setBlocked(null)}>
        {blocked && (
          <AlertDialog
            title={blocked.title}
            cancelLabel={blocked.cta ? 'Fechar' : undefined}
            primaryActionLabel={
              blocked.cta === 'settings'
                ? 'Abrir Configurações'
                : blocked.cta === 'reconnect'
                  ? 'Reconectar'
                  : 'Fechar'
            }
            onCancel={() => setBlocked(null)}
            onPrimaryAction={() => {
              setBlocked(null)
              if (blocked.cta === 'settings') setSettingsOpen(true)
              else if (blocked.cta === 'reconnect') reconnectNow()
            }}
          >
            {blocked.text.split('\n\n').map((p, i) => (
              <p key={i}>{p}</p>
            ))}
          </AlertDialog>
        )}
      </DialogContainer>

      {/* Só as cópias e as falhas delas disparam toast (docs/FRONTEND.md, Início, barra de status e toast). */}
      <ToastContainer placement="bottom end" />
    </div>
  )
}

// A grade do shell (docs/FRONTEND.md, Shell e janela): a titlebar em cima, as
// três colunas no meio e a barra de status embaixo, na largura toda.
const shell = style({
  display: 'grid',
  gridTemplateRows: [48, 'minmax(0, 1fr)', 30],
  height: 'screen',
  overflow: 'hidden',
})

const columns = style({
  display: 'grid',
  gridTemplateColumns: ['auto', 'minmax(0, 1fr)', 'auto'],
  minHeight: 0,
})

// A coluna do meio: a barra de abas em cima e os painéis embaixo.
const content = style({ display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0 })

// Os painéis empilham na mesma célula. O de aba fora de foco é `inert` (RAC) e
// some com `visibility`, que, ao contrário de `display: none`, guarda a rolagem.
// O macro não emite `gridArea`; `position: absolute` + `inset: 0` põe todos os
// painéis na mesma caixa e deixa o painel com a altura da coluna.
// ponytail: toda aba aberta fica montada e com layout; virtualizar se muitas
// abas pesarem.
const panels = style({ position: 'relative', flexGrow: 1, minHeight: 0 })

const tabPanel = style({
  position: 'absolute',
  inset: 0,
  display: 'grid',
  gridTemplateRows: ['minmax(0, 1fr)', 'auto'],
  minWidth: 0,
  minHeight: 0,
  overflow: 'hidden',
  outlineStyle: 'none',
  visibility: { default: 'visible', isInert: 'hidden' },
})

// 360 px, que caem para 300 px abaixo de 1080 px; anima como o SideNavPanel. A
// gaveta lê a mesma `--logs-w`, para não refluir enquanto a coluna anima.
const logsColumn = style({
  '--logs-w': { type: 'width', value: { default: 360, '@media (max-width: 1079px)': 300 } },
  width: { default: 0, isOpen: '[var(--logs-w)]' },
  transition: '[width]',
  transitionDuration: 300,
  transitionTimingFunction: 'default',
  overflow: 'hidden',
  display: 'grid',
  minHeight: 0,
})

export default App
