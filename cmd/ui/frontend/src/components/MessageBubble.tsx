import {
  Check,
  Eye,
  FileText,
  Globe,
  MagnifyingGlass,
  PencilSimple,
} from '@phosphor-icons/react'
import { Message, ToolCall, contextOrigin } from '../types'
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
  // A janela e a origem dela acompanham o uso medido do turno. Sem total não
  // há o que comparar — o item fica de fora em vez de exibir "1.234/0" — e a
  // origem (provedor ou declarado) é dita ao lado do número, porque declarado
  // não se passa por medido.
  const origin = usage.contextTotal > 0 ? contextOrigin(usage.contextSource) : null

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
      {origin && (
        <>
          <span className="usage__sep" aria-hidden="true" />
          <span className="usage__item">
            Contexto{' '}
            <strong>
              {fmtNum(usage.contextUsed)}/{fmtNum(usage.contextTotal)}
            </strong>
            <span className="usage__origin" title={origin.title}>
              ({origin.label})
            </span>
          </span>
        </>
      )}
    </div>
  )
}

/**
 * Os avisos de degradação do turno. Ficam na bolha do turno em que
 * aconteceram: é a resposta degradada que eles explicam, e um aviso que só
 * chega ao arquivo de log é um aviso silencioso.
 */
function WarningList({ warnings }: { warnings: string[] }) {
  return (
    <ul className="warnings" role="status">
      {warnings.map((w, i) => (
        <li key={i} className="warnings__item">
          {w}
        </li>
      ))}
    </ul>
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
  const warnings = message.warnings?.length ? <WarningList warnings={message.warnings} /> : null

  return (
    <div className="msg msg--agent" data-msg-id={message.id}>
      {message.error ? (
        <div className="bubble bubble--agent bubble--error" role="alert">
          <p className="error__title">{message.error.what}</p>
          <p className="error__why">{message.error.why}</p>
          <p className="error__next">{message.error.next}</p>
          {warnings}
        </div>
      ) : (
        <div className={`bubble bubble--agent ${streaming ? 'bubble--streaming' : ''}`}>
          {/* Quem respondeu. Só depois do stream: o autor chega com a conversa
              gravada, e enquanto os tokens correm a mensagem na tela ainda não
              o tem — dizer "Agente" para depois trocar por "Teacher" seria
              mostrar um autor que não é o do turno. Vazio É o agente genérico,
              e aparece como tal: nada escrito não distinguiria o genérico de
              uma conversa gravada antes de o autor existir. */}
          {!streaming && <span className="author">{message.subagent || 'Agente'}</span>}

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

          {warnings}

          {message.status === 'done' && message.time && (
            <span className="msg-ts msg-ts--meta">
              {message.time}
              {message.elapsedMs != null && ` · ${fmtDuration(message.elapsedMs)}`}
              {/* tokens medidos: vêm do usage do provedor, gravado no disco */}
              {message.tokens != null && ` · ${message.tokens} tokens`}
            </span>
          )}
        </div>
      )}
    </div>
  )
}