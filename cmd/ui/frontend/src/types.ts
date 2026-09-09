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