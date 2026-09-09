import {
  Check,
  Eye,
  FileText,
  Globe,
  MagnifyingGlass,
  PencilSimple,
} from '@phosphor-icons/react'
import { Message, ToolCall } from '../types'
import Markdown from './Markdown'

const ACTION_ICONS: Record<string, React.ReactNode> = {
  search: <MagnifyingGlass size={14} weight="bold" />,
  file: <FileText size={14} weight="bold" />,
  globe: <Globe size={14} weight="bold" />,
  pencil: <PencilSimple size={14} weight="bold" />,
  eye: <Eye size={14} weight="bold" />,
}

function ToolCallCard({ tc }: { tc: ToolCall }) {
  return (
    <details className="tool-call">
      <summary className="tool-call__summary">
        <span className="tool-call__icon" aria-hidden="true">
          {ACTION_ICONS[tc.action] ?? <PencilSimple size={14} weight="bold" />}
        </span>
        <span className="tool-call__label">{tc.label}</span>
        <span className="tool-call__state" aria-label="Concluída">
          <Check size={14} weight="bold" />
        </span>
      </summary>
      <div className="tool-call__result">{tc.result}</div>
    </details>
  )
}

function ThinkingIndicator() {
  return (
    <span className="thinking">
      <span className="thinking__dot" aria-hidden="true" />
      Trabalhando
    </span>
  )
}

function fmtDuration(ms: number): string {
  if (ms < 1000) return `${ms.toLocaleString('pt-BR')} ms`
  const s = ms / 1000
  if (s < 60) return `${s.toLocaleString('pt-BR', { maximumFractionDigits: 1 })} s`
  const m = Math.floor(s / 60)
  const rs = Math.round(s % 60)
  return `${m} min ${rs} s`
}

const fmtNum = (n: number): string => n.toLocaleString('pt-BR')

function TokenUsageFooter({
  usage,
  own,
}: {
  usage: NonNullable<Message['usage']>
  own?: boolean
}) {
  return (
    <div className={`usage ${own ? 'usage--own' : ''}`}>
      <span className="usage__item">
        Entrada <strong>{fmtNum(usage.input)}</strong>
      </span>
      <span className="usage__sep" aria-hidden="true" />
      <span className="usage__item">
        Saída <strong>{fmtNum(usage.output)}</strong>
      </span>
      <span className="usage__sep" aria-hidden="true" />
      <span className="usage__item">
        Total <strong>{fmtNum(usage.total)}</strong>
      </span>
      <span className="usage__sep" aria-hidden="true" />
      <span className="usage__item">
        Contexto{' '}
        <strong>
          {fmtNum(usage.contextUsed)}/{fmtNum(usage.contextTotal)}
        </strong>
      </span>
    </div>
  )
}

export default function MessageBubble({ message }: { message: Message }) {
  if (message.role === 'user') {
    return (
      <div className="msg msg--user" data-msg-id={message.id}>
        <div className="bubble bubble--user">{message.content}</div>
        {message.time && <span className="msg-ts msg-ts--own">{message.time}</span>}
        {message.usage && <TokenUsageFooter usage={message.usage} own />}
      </div>
    )
  }

  const streaming = message.status === 'streaming'
  const isEmpty = message.content.trim() === ''

  return (
    <div className="msg msg--agent" data-msg-id={message.id}>
      {message.error ? (
        <div className="bubble bubble--agent bubble--error" role="alert">
          <p className="error__title">{message.error.what}</p>
          <p className="error__why">{message.error.why}</p>
          <p className="error__next">{message.error.next}</p>
        </div>
      ) : (
        <div className={`bubble bubble--agent ${streaming ? 'bubble--streaming' : ''}`}>
          {message.toolCalls && message.toolCalls.length > 0 && (
            <div className="bubble__tools">
              {message.toolCalls.map((tc) => (
                <ToolCallCard key={tc.id} tc={tc} />
              ))}
            </div>
          )}

          {isEmpty && streaming ? (
            <ThinkingIndicator />
          ) : (
            <Markdown content={message.content} streaming={streaming} />
          )}

          {message.status === 'done' && message.time && (
            <span className="msg-ts msg-ts--meta">
              {message.time}
              {message.elapsedMs != null && ` · ${fmtDuration(message.elapsedMs)}`}
              {message.tokens != null && ` · ~${message.tokens} tokens`}
            </span>
          )}
        </div>
      )}
    </div>
  )
}