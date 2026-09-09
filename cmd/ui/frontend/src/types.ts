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

export interface ModelConfig {
  id: string
  name: string
  maxContext: number
  maxTokens: number
  temperature: number
  topP: number
  default: boolean
}

export interface ProviderConfig {
  id: string
  name: string
  baseURL: string
  apiKey: string
  enabled: boolean
  models: ModelConfig[]
}

export interface Settings {
  providers: ProviderConfig[]
}

export const EMPTY_PROVIDER: Omit<ProviderConfig, 'id'> = {
  name: '',
  baseURL: '',
  apiKey: '',
  enabled: true,
  models: [],
}

export const EMPTY_MODEL: Omit<ModelConfig, 'id'> = {
  name: '',
  maxContext: 128000,
  maxTokens: 4096,
  temperature: 0.7,
  topP: 1,
  default: false,
}

/** Provedor/modelo em uso: o primeiro provedor ativo que tenha modelos,
 *  preferindo o modelo marcado como padrão. */
export function activeModel(
  s: Settings | null,
): { provider: ProviderConfig; model: ModelConfig } | null {
  if (!s) return null
  for (const p of s.providers) {
    if (!p.enabled || p.models.length === 0) continue
    const model = p.models.find((m) => m.default) ?? p.models[0]
    return { provider: p, model }
  }
  return null
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