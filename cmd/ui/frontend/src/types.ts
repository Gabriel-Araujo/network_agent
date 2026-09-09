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
}

export interface ChatMeta {
  id: string
  title: string
  group: string
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

export const FALLBACK_CHATS: ChatMeta[] = [
  { id: 'c1', title: 'Interface do agente de IA', group: 'Hoje' },
  { id: 'c2', title: 'Stack Wails + React', group: 'Hoje' },
  { id: 'c3', title: 'Paleta glass azulado', group: 'Ontem' },
  { id: 'c4', title: 'Prompt /impeccable', group: 'Ontem' },
  { id: 'c5', title: 'Arquitetura IPC-first', group: '7 dias atrás' },
]

// Tiny inline markdown renderer: bold, inline code, line breaks.
export function renderInline(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
}

export function toParagraphs(text: string): string[] {
  return text.split(/\n+/).filter((p) => p.trim().length > 0)
}