import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Clock from '@react-spectrum/s2/icons/Clock'
import { ContextWindow, Conversation, contextOrigin, formatTs } from '../types'

// A barra de status (docs/FRONTEND.md, Início, barra de status e toast): mostra a
// Aba focada. Lê a Conversa como ela está no disco (a entrada da lista), então,
// durante um turno, o horário e os totais seguem os do turno anterior até ele
// gravar. Na Início, `conversation` é null: totais zerados e sem horário.

const bar = style({
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  height: 30,
  paddingX: 16,
  backgroundColor: 'layer-1',
  font: 'ui-sm',
  color: 'neutral-subdued',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  userSelect: 'none',
  '--iconPrimary': { type: 'color', value: '[currentColor]' },
})

const item = style({ display: 'flex', alignItems: 'center', gap: 4 })
const strong = style({ fontWeight: 'bold', color: 'neutral' })
const grow = style({ flexGrow: 1 })
const icon = iconStyle({ size: 'XS' })

const fmtNum = (n: number): string => n.toLocaleString('pt-BR')

interface StatusBarProps {
  conversation: Conversation | null
  /** A janela do modelo ativo agora; sem ela vale a do último turno medido. */
  contextWindow: ContextWindow | null
}

export default function StatusBar({ conversation, contextWindow }: StatusBarProps) {
  let input = 0
  let output = 0
  let contextUsed = 0
  let contextTotal = 0
  let contextSource: string | undefined
  let last: string | undefined
  for (const m of conversation?.messages ?? []) {
    if (m.role === 'agent' && m.time) last = m.time
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
  // Sem janela sondada nem medida, o item de contexto não aparece.
  const origin = contextTotal > 0 ? contextOrigin(contextSource) : null

  return (
    <footer className={bar} aria-label="Barra de status">
      {last && (
        <span className={item} title="Último turno">
          <Clock styles={icon} />
          {formatTs(last)}
        </span>
      )}
      <span className={grow} />
      <span className={item}>
        Total de tokens de entrada: <b className={strong}>{fmtNum(input)}</b>
      </span>
      <span className={item}>
        Total de tokens de saída: <b className={strong}>{fmtNum(output)}</b>
      </span>
      <span className={item}>
        Total de tokens: <b className={strong}>{fmtNum(input + output)}</b>
      </span>
      {origin && (
        <span className={item}>
          Contexto utilizado:{' '}
          <b className={strong}>
            {fmtNum(contextUsed)}/{fmtNum(contextTotal)}
          </b>
          <span title={origin.title}>({origin.label})</span>
        </span>
      )}
    </footer>
  )
}
