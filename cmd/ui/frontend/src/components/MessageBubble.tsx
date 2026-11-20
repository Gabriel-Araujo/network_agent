import {
  Content,
  Disclosure,
  DisclosurePanel,
  DisclosureTitle,
  Heading,
  InlineAlert,
  ProgressCircle,
} from '@react-spectrum/s2'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import CheckmarkCircle from '@react-spectrum/s2/icons/CheckmarkCircle'
import Edit from '@react-spectrum/s2/icons/Edit'
import FileText from '@react-spectrum/s2/icons/FileText'
import GlobeGrid from '@react-spectrum/s2/icons/GlobeGrid'
import Search from '@react-spectrum/s2/icons/Search'
import Visibility from '@react-spectrum/s2/icons/Visibility'
import { Message, ToolCall, contextOrigin } from '../types'
import Markdown from './Markdown'
import { mono } from './CodeBlock'

// As bolhas da Conversa (docs/FRONTEND.md, Componentes): a do usuário é própria;
// o agente fala sem bolha, na largura da coluna. Tool call e "Mostrar resposta
// original" são `Disclosure`, o erro e os Avisos são `InlineAlert`.

const userCol = style({
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'end',
  gap: 4,
})
const userBubble = style({
  maxWidth: '[80%]',
  paddingX: 16,
  paddingY: 12,
  borderRadius: 'xl',
  backgroundColor: 'accent-subtle',
  font: 'ui',
  color: 'neutral',
  whiteSpace: 'pre-wrap',
  overflowWrap: 'break-word',
})
const agentCol = style({ display: 'flex', flexDirection: 'column', minWidth: 0 })
const author = style({
  font: 'detail-sm',
  color: 'neutral-subdued',
  textTransform: 'capitalize',
  marginBottom: 8,
})
const meta = style({ font: 'ui-sm', color: 'neutral-subdued', marginTop: 12 })
const metaOwn = style({ font: 'ui-sm', color: 'neutral-subdued' })
const usageRow = style({
  display: 'flex',
  flexWrap: 'wrap',
  alignItems: 'center',
  columnGap: 12,
  rowGap: 2,
  font: 'ui-sm',
  color: 'neutral-subdued',
})
const usageOwn = style({ justifyContent: 'end' })
const num = style({ fontWeight: 'bold', color: 'neutral' })
const help = style({ cursor: 'help' })

const tools = style({ display: 'flex', flexDirection: 'column', gap: 4, marginBottom: 12 })
const toolTitle = style({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  '--iconPrimary': { type: 'color', value: '[currentColor]' },
})
const toolLabel = style({ flexGrow: 1 })
const toolDone = style({ display: 'inline-flex' })
const toolResult = style({
  font: 'ui-sm',
  color: 'neutral-subdued',
  whiteSpace: 'pre-wrap',
  overflowWrap: 'anywhere',
})
const icon = iconStyle({ size: 'S' })
const doneIcon = iconStyle({ size: 'S', color: 'positive' })

const thinking = style({
  display: 'inline-flex',
  alignItems: 'center',
  gap: 8,
  font: 'ui',
  color: 'neutral-subdued',
})

const cause = style({ margin: 0, whiteSpace: 'pre-wrap', fontSize: 'code-sm', lineHeight: 'code' })
const alertWide = style({ width: 'full' })
const alertGap = style({ marginTop: 12 })
const warnList = style({ margin: 0, paddingStart: 20 })
const repairText = style({ marginTop: 12, marginBottom: 0, font: 'ui-sm', color: 'neutral-subdued' })

const ACTION_ICONS: Record<string, React.ReactNode> = {
  search: <Search styles={icon} />,
  file: <FileText styles={icon} />,
  globe: <GlobeGrid styles={icon} />,
  pencil: <Edit styles={icon} />,
  eye: <Visibility styles={icon} />,
}

function ToolCallCard({ tc }: { tc: ToolCall }) {
  return (
    <Disclosure isQuiet size="S" density="compact">
      <DisclosureTitle>
        <span className={toolTitle}>
          {ACTION_ICONS[tc.action] ?? <Edit styles={icon} />}
          <span className={toolLabel}>{tc.label}</span>
          <span className={toolDone} role="img" aria-label="Concluída">
            <CheckmarkCircle styles={doneIcon} />
          </span>
        </span>
      </DisclosureTitle>
      <DisclosurePanel>
        <div className={toolResult}>{tc.result}</div>
      </DisclosurePanel>
    </Disclosure>
  )
}

function ThinkingIndicator() {
  return (
    <span className={thinking}>
      <ProgressCircle size="S" isIndeterminate aria-label="Trabalhando" />
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
    <div className={own ? `${usageRow} ${usageOwn}` : usageRow}>
      <span>
        Entrada <b className={num}>{fmtNum(usage.input)}</b>
      </span>
      <span>
        Saída <b className={num}>{fmtNum(usage.output)}</b>
      </span>
      <span>
        Total <b className={num}>{fmtNum(usage.total)}</b>
      </span>
      {origin && (
        <span>
          Contexto{' '}
          <b className={num}>
            {fmtNum(usage.contextUsed)}/{fmtNum(usage.contextTotal)}
          </b>{' '}
          <span className={help} title={origin.title}>
            ({origin.label})
          </span>
        </span>
      )}
    </div>
  )
}

/**
 * Os avisos de degradação do turno. Ficam na bolha do turno em que
 * aconteceram: é a resposta degradada que eles explicam, e um aviso que só
 * chega ao arquivo de log é um aviso silencioso. Um `InlineAlert` por mensagem,
 * com a lista dentro.
 */
function WarningList({ warnings }: { warnings: string[] }) {
  return (
    <div className={alertGap}>
      <InlineAlert variant="notice" styles={alertWide}>
        <Heading>{warnings.length > 1 ? 'Avisos' : 'Aviso'}</Heading>
        <Content>
          <ul className={warnList}>
            {warnings.map((w, i) => (
              <li key={i}>{w}</li>
            ))}
          </ul>
        </Content>
      </InlineAlert>
    </div>
  )
}

export default function MessageBubble({ message }: { message: Message }) {
  if (message.role === 'user') {
    return (
      <div className={userCol} data-msg-id={message.id}>
        <div className={userBubble}>{message.content}</div>
        {message.time && <span className={metaOwn}>{message.time}</span>}
        {message.usage && <TokenUsageFooter usage={message.usage} own />}
      </div>
    )
  }

  const streaming = message.status === 'streaming'
  const repair = message.mermaidRepair
  const content = repair?.state === 'accepted' && repair.correctedContent != null
    ? repair.correctedContent : message.content
  const isEmpty = content.trim() === ''
  const warnings = message.warnings?.length ? <WarningList warnings={message.warnings} /> : null

  return (
    <div className={agentCol} data-msg-id={message.id}>
      {message.error ? (
        <>
          <InlineAlert variant="negative" styles={alertWide}>
            <Heading>{message.error.what}</Heading>
            <Content>
              {/* A causa crua pode vir multilinha (o parse error do mermaid traz a
                  linha e uma linha de caret): o pre-wrap mantém o caret alinhado. */}
              <p className={`${cause} ${mono}`}>{message.error.why}</p>
              <p>{message.error.next}</p>
            </Content>
          </InlineAlert>
          {warnings}
        </>
      ) : (
        <>
          {/* Quem respondeu. Só depois do stream: o autor chega com a conversa
              gravada, e enquanto os tokens correm a mensagem na tela ainda não
              o tem — dizer "Agente" para depois trocar por "Teacher" seria
              mostrar um autor que não é o do turno. Vazio É o agente genérico,
              e aparece como tal: nada escrito não distinguiria o genérico de
              uma conversa gravada antes de o autor existir. */}
          {!streaming && <span className={author}>{message.subagent || 'Agente'}</span>}

          {message.toolCalls && message.toolCalls.length > 0 && (
            <div className={tools}>
              {message.toolCalls.map((tc) => (
                <ToolCallCard key={tc.id} tc={tc} />
              ))}
            </div>
          )}

          {isEmpty && streaming ? (
            <ThinkingIndicator />
          ) : (
            <Markdown content={content} streaming={streaming} />
          )}

          {repair?.state === 'repairing' && <p className={repairText} role="status">Tentando corrigir o diagrama automaticamente…</p>}
          {(repair?.state === 'started' || repair?.state === 'candidate') && <p className={repairText} role="status">A tentativa de correção foi interrompida. Peça outra forma ao agente.</p>}
          {(repair?.state === 'failed' || repair?.state === 'rejected') && <p className={repairText} role="status">A correção automática não foi concluída. {repair.error} Peça outra forma ao agente.</p>}
          {repair?.state === 'accepted' && (
            <div className={repairText} role="status">
              Diagrama corrigido automaticamente.
              <Disclosure isQuiet size="S" density="compact">
                <DisclosureTitle>Mostrar resposta original</DisclosureTitle>
                <DisclosurePanel>
                  <Markdown content={message.content} />
                </DisclosurePanel>
              </Disclosure>
            </div>
          )}

          {warnings}

          {message.status === 'done' && message.time && (
            <span className={meta}>
              {message.time}
              {message.elapsedMs != null && ` · ${fmtDuration(message.elapsedMs)}`}
              {/* tokens medidos: vêm do usage do provedor, gravado no disco */}
              {message.tokens != null && ` · ${message.tokens} tokens`}
            </span>
          )}
        </>
      )}
    </div>
  )
}
