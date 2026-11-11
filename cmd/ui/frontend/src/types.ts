export type Role = 'user' | 'agent'
export type MsgStatus = 'streaming' | 'done' | 'error'

export interface ToolCall {
  id: string
  action: string
  label: string
  result: string
}

export interface AgentError {
  what: string
  why: string
  next: string
}

export interface TokenUsage {
  input: number
  output: number
  total: number
  contextUsed: number
  contextTotal: number
  /**
   * De onde veio a janela (`contextTotal`): 'provider' quando o endpoint do
   * provedor a informou, 'config' quando é o context_size declarado na
   * configuração. Conversas gravadas antes desta marcação vêm sem o campo — e o
   * número delas era o declarado.
   */
  contextSource?: string
}

/** A janela de contexto do modelo ativo: o total e a origem (cmd/ui/context.go). */
export interface ContextWindow {
  total: number
  source: string
}

/**
 * O estado da conexão com o provedor (cmd/ui/app.go). A conexão é background —
 * a carga do modelo no provedor leva dezenas de segundos —, então a ausência de
 * agente não é mais sinônimo de falta de configuração: `cause` só vem no
 * estado failed, e é o erro do Connect.
 */
export interface Connection {
  state: 'none' | 'connecting' | 'failed' | 'ready'
  cause?: string
}

export interface Message {
  id: string
  role: Role
  content: string
  status: MsgStatus
  time?: string
  elapsedMs?: number
  tokens?: number
  usage?: TokenUsage
  toolCalls?: ToolCall[]
  error?: AgentError
  /**
   * Quem respondeu o turno: o Subagente (`teacher`, `builder`, `modifier`,
   * `illustrator`) ou vazio para o agente genérico. É quem DE FATO respondeu,
   * não quem a classificação nomeou. Conversa gravada antes desta decisão vem
   * sem o campo — e o turno dela foi do genérico, que é o mesmo vazio.
   */
  subagent?: string
  /**
   * Avisos de degradação do turno, na ordem em que saíram: classificação que
   * falhou, retrieval que não veio, Subagente que não carregou. O turno
   * respondeu — degradação nunca mata o turno —, mas com menos do que devia.
   */
  warnings?: string[]
}

/** Conversa como o Go a devolve (`history.Conversation`): transcript junto, updatedAt desc. */
export interface Conversation {
  id: string
  title: string
  createdAt: string
  updatedAt: string
  provider: string
  model: string
  messages: Message[]
}

// ---------- erros da ponte do Wails ----------
// O Wails entrega `error` ao TS como string, então o Go manda o JSON
// {code, what, why, next} dentro dela (cmd/ui/history.go). Isto é o que a
// camada de cima do bridgeError diz: cada código tem um `next` acionável.

export interface BridgeError {
  code: string
  what: string
  why: string
  next: string
}

/** Interpreta o erro que o Wails entrega ao TS (uma string, com o JSON dentro). */
export function parseBridgeError(e: unknown): BridgeError {
  const raw = e instanceof Error ? e.message : String(e ?? '')
  try {
    const p = JSON.parse(raw)
    if (p && typeof p === 'object' && 'what' in p) {
      return {
        code: String(p.code ?? ''),
        what: String(p.what ?? ''),
        why: String(p.why ?? ''),
        next: String(p.next ?? ''),
      }
    }
  } catch {
    // não era o JSON da ponte: cai no genérico abaixo
  }
  return {
    code: '',
    what: 'Falha inesperada',
    why: raw,
    next: 'Tente de novo; se persistir, feche e reabra o app.',
  }
}

// ---------- tempo ----------

/**
 * Rótulo e explicação da origem da janela de contexto exibida pelo medidor.
 *
 * O número da janela vem do endpoint do provedor quando ele o expõe e do
 * context_size declarado quando não — e o medidor diz qual dos dois é, porque
 * número declarado não se passa por medido. Sem marcação (conversa gravada
 * antes disto) o valor era o declarado.
 */
export function contextOrigin(source?: string): { label: string; title: string } {
  if (source === 'provider') {
    return {
      label: 'provedor',
      title: 'Janela de contexto lida agora do endpoint do provedor.',
    }
  }
  return {
    label: 'declarado',
    title:
      'O provedor não expõe a janela de contexto; este é o contexto máximo declarado na configuração.',
  }
}

/**
 * Formata um instante do disco (RFC3339 em UTC) para exibição: "dd/mm/aaaa,
 * hh:mm" no fuso local. É o MESMO formatador para a mensagem ao vivo e a
 * reaberta — mensagem de disco nunca guarda hora já localizada, e sim instante.
 */
export function formatTs(stamp: string): string {
  const d = new Date(stamp)
  if (Number.isNaN(d.getTime())) return stamp
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Projeta o transcript do disco na tela: formata o tempo e devolve o `usage` do
 * turno ao balão do usuário.
 *
 * No disco o `usage` mora na mensagem do agente (o Store é append-only e a
 * mensagem do usuário já está gravada quando o uso chega — ticket 07), mas a UI
 * o exibe sob o balão do usuário, lendo o turno. Mover na projeção deixa a
 * conversa reaberta idêntica à que estava na tela e mantém o resumo do rodapé
 * sem contagem dobrada.
 */
export function toUiMessages(msgs: Message[] | null | undefined): Message[] {
  const out = (msgs ?? []).map((m) => ({ ...m, time: m.time ? formatTs(m.time) : undefined }))
  for (let i = 0; i < out.length; i++) {
    if (out[i].role !== 'user') continue
    // a resposta do turno é o primeiro agente depois desta pergunta
    for (let j = i + 1; j < out.length && out[j].role !== 'user'; j++) {
      if (out[j].usage) {
        out[i] = { ...out[i], usage: out[j].usage }
        out[j] = { ...out[j], usage: undefined }
        break
      }
    }
  }
  return out
}

// ---------- agrupamento da lista por data ----------

const DAY_MS = 86_400_000
const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()

/** Rótulo do grupo de data de uma conversa, derivado do `updatedAt`. */
export function groupLabel(stamp: string): string {
  const d = new Date(stamp)
  if (Number.isNaN(d.getTime())) return 'Mais antigas'
  const days = Math.round((startOfDay(new Date()) - startOfDay(d)) / DAY_MS)
  if (days <= 0) return 'Hoje' // <=0: hoje (ou um relógio adiantado, que não inventa grupo novo)
  if (days === 1) return 'Ontem'
  if (days <= 7) return 'Últimos 7 dias'
  return 'Mais antigas'
}

const GROUP_ORDER = ['Hoje', 'Ontem', 'Últimos 7 dias', 'Mais antigas']

/**
 * Agrupa para o sidebar: cabeçalhos na ordem fixa acima, itens na ordem
 * recebida (o Store já devolve updatedAt desc). Grupo sem conversa não aparece.
 */
export function groupConversations(
  convs: Conversation[],
): { label: string; items: Conversation[] }[] {
  const buckets = new Map<string, Conversation[]>()
  for (const c of convs) {
    const label = groupLabel(c.updatedAt)
    const list = buckets.get(label)
    if (list) list.push(c)
    else buckets.set(label, [c])
  }
  return GROUP_ORDER.filter((l) => buckets.has(l)).map((label) => ({
    label,
    items: buckets.get(label) ?? [],
  }))
}

/**
 * Reordena por updatedAt desc — o mesmo critério do Store, em que renomear
 * toca o updatedAt e sobe a conversa para o topo. Comparação de string basta:
 * o carimbo é RFC3339 UTC de largura fixa, então ordem lexicográfica é a
 * cronológica.
 */
export function sortByUpdatedDesc(list: Conversation[]): Conversation[] {
  return [...list].sort((a, b) =>
    a.updatedAt < b.updatedAt ? 1 : a.updatedAt > b.updatedAt ? -1 : 0,
  )
}

// ---------- configurações: provedores e modelos ----------
// Espelha internal/types (types.Config/Provider/Model). Os registros são
// identificados por ÍNDICE, como no Go — não há campo id de provedor.

export interface ModelConfig {
  id: string
  name: string
  context_size: number
}

export interface ProviderConfig {
  name: string
  url: string
  api_key: string
  default_model: number
  models: ModelConfig[]
}

export interface AppConfig {
  selected: number
  providers: ProviderConfig[]
}

export const EMPTY_PROVIDER: ProviderConfig = {
  name: '',
  url: '',
  api_key: '',
  default_model: 0,
  models: [],
}

export const EMPTY_MODEL: ModelConfig = {
  id: '',
  name: '',
  context_size: 128000,
}

/** Provedor/modelo em uso: o selecionado na config e seu modelo padrão. */
export function activeModel(
  c: AppConfig | null,
): { provider: ProviderConfig; model: ModelConfig } | null {
  if (!c || !c.providers?.length) return null
  const provider = c.providers[c.selected] ?? c.providers[0]
  if (!provider || !provider.models?.length) return null
  const model = provider.models[provider.default_model] ?? provider.models[0]
  return { provider, model }
}
