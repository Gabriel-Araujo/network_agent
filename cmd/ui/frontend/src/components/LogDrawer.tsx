import { useEffect, useRef, useState } from 'react'
import { ActionButton, Link, ToastQueue, Tooltip, TooltipTrigger } from '@react-spectrum/s2'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Close from '@react-spectrum/s2/icons/Close'
import { App } from '../../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { inWails } from '../types'
import { copyText, mono } from './CodeBlock'

// A gaveta de Logs (docs/FRONTEND.md, Shell e janela): as MESMAS linhas que o CLI
// mostra no terminal, porque a fonte é o sink de console — o Go escreve nos dois
// destinos no mesmo Handle (ticket 06). Daí decorre o que ela não mostra: nada de
// `source`, valor truncado em 512, nada de TRACE. Quem quer isso lê o arquivo, e é
// para isso que serve o "copiar o caminho".

// Transporte: pull, só enquanto aberta. O writer do lado Go não conhece o
// Wails e não emite evento — com isso nada aqui depende do ctx do app, que é
// nil durante as primeiras linhas de todo run.
const POLL_MS = 250

// O nível sai do token `[INFO ]`, não de ANSI (que o writer do Go tira): o
// timestamp do console tem 23 chars (timeLayout) e o rótulo tem largura fixa 5
// (levelWidth), então o token com os colchetes ocupa 7 chars a partir do 24.
// internal/logger/console.go:16,20
const LEVEL_AT = 24
const LEVEL_LEN = 7

function levelOf(line: string): string {
  if (line[LEVEL_AT] !== '[') return ''
  return line.slice(LEVEL_AT + 1, LEVEL_AT + LEVEL_LEN - 1).trim().toLowerCase()
}

// A gaveta fica na largura final da coluna, `--logs-w` (App.tsx), que a coluna
// anima: assim as linhas não refluem enquanto ela abre.
const drawer = style({
  width: '[var(--logs-w)]',
  height: 'full',
  display: 'grid',
  gridTemplateRows: ['auto', 'minmax(0, 1fr)', 'auto'],
  minWidth: 0,
  minHeight: 0,
  backgroundColor: 'layer-1',
})

const head = style({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  paddingStart: 16,
  paddingEnd: 8,
  paddingY: 4,
  borderWidth: 0,
  borderBottomWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
})

const title = style({
  font: 'ui-sm',
  fontWeight: 'bold',
  textTransform: 'uppercase',
  color: 'neutral-subdued',
})

const count = style({
  font: 'ui-sm',
  paddingX: 8,
  borderRadius: 'full',
  backgroundColor: 'accent-subtle',
  color: 'accent',
  fontVariantNumeric: 'tabular-nums',
})

const grow = style({ flexGrow: 1 })
const icon = iconStyle({ size: 'S' })

const body = style({
  overflow: 'auto',
  paddingX: 12,
  paddingY: 8,
  fontSize: 'code-xs',
  lineHeight: 'code',
  color: 'neutral',
})

// pre-wrap: a linha do console é larga (valor truncado em 512) e o diagnóstico
// está no fim dela — quebrar é melhor que esconder atrás de scroll lateral.
const line = style({ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' })
const stamp = style({ color: 'neutral-subdued' })

// As cores do console (console.go:25-30) nos tokens do S2: o nível é a única cor
// da linha, como no terminal.
const levelStyle = style<{ level: 'debug' | 'info' | 'warn' | 'error' }>({
  fontWeight: 'bold',
  color: {
    level: { debug: 'cyan-900', info: 'positive-900', warn: 'notice-900', error: 'negative' },
  },
})

const empty = style({ margin: 0, font: 'ui-sm', color: 'neutral-subdued' })

const foot = style({
  display: 'flex',
  justifyContent: 'center',
  paddingX: 12,
  paddingY: 8,
  borderWidth: 0,
  borderTopWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  font: 'ui-sm',
  color: 'neutral-subdued',
})

function isLevel(s: string): s is 'debug' | 'info' | 'warn' | 'error' {
  return s === 'debug' || s === 'info' || s === 'warn' || s === 'error'
}

// Uma linha. Uma mensagem terminada em \n rende uma linha em branco (medido: 3 das
// 15 do run); o terminal a mostra, então a gaveta também: o espaço dá a altura.
function LogLine({ text }: { text: string }) {
  const level = levelOf(text)
  if (!isLevel(level)) return <div className={line}>{text || ' '}</div>
  return (
    <div className={line}>
      <span className={stamp}>{text.slice(0, LEVEL_AT)}</span>
      <span className={levelStyle({ level })}>{text.slice(LEVEL_AT, LEVEL_AT + LEVEL_LEN)}</span>
      {text.slice(LEVEL_AT + LEVEL_LEN)}
    </div>
  )
}

interface LogDrawerProps {
  open: boolean
  onClose: () => void
}

export default function LogDrawer({ open, onClose }: LogDrawerProps) {
  const [lines, setLines] = useState<string[]>([])
  const [path, setPath] = useState('')
  const bodyRef = useRef<HTMLDivElement>(null)
  // Acompanha o fim só quando o usuário já está no fim: rolar para trás para
  // ler uma linha antiga não é desfeito pelo tick seguinte.
  const pinned = useRef(true)

  useEffect(() => {
    if (!open) return
    if (!inWails()) return

    let alive = true
    const tick = async () => {
      try {
        const next: string[] = await App.GetLogLines()
        if (alive) setLines(next ?? [])
      } catch {
        /* ponte ausente (browser puro): a gaveta fica vazia, o app segue */
      }
    }
    tick()
    const id = window.setInterval(tick, POLL_MS)

    App.GetLogPath()
      .then((p: string) => {
        if (alive) setPath(p ?? '')
      })
      .catch(() => {})

    return () => {
      alive = false
      window.clearInterval(id)
    }
  }, [open])

  useEffect(() => {
    const box = bodyRef.current
    if (!box || !pinned.current) return
    box.scrollTop = box.scrollHeight
  }, [lines])

  // O caminho vai para um grep ou para o anexo de um ticket; abrir no editor
  // seria "um local" que não é do app (ticket 06).
  const copyPath = () =>
    copyText(path).then(
      () => ToastQueue.positive('Caminho copiado.', { timeout: 5000 }),
      () => ToastQueue.negative('Não foi possível copiar.', { timeout: 5000 }),
    )

  return (
    <aside className={drawer} aria-label="Logs da execução">
      <header className={head}>
        <span className={title}>Logs</span>
        <span className={count}>{lines.length}</span>
        <span className={grow} />
        <TooltipTrigger>
          <ActionButton isQuiet size="S" aria-label="Fechar os logs" onPress={onClose}>
            <Close styles={icon} />
          </ActionButton>
          <Tooltip>Fechar</Tooltip>
        </TooltipTrigger>
      </header>

      <div
        className={`${body} ${mono}`}
        ref={bodyRef}
        onScroll={(e) => {
          const el = e.currentTarget
          pinned.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24
        }}
      >
        {lines.length === 0 ? (
          <p className={empty}>Nenhuma linha nesta execução.</p>
        ) : (
          lines.map((text, i) => <LogLine key={i} text={text} />)
        )}
      </div>

      <footer className={foot}>
        {path ? (
          <Link variant="secondary" isStandalone onPress={copyPath}>
            Copiar o caminho do arquivo
          </Link>
        ) : (
          'Esta execução não abriu arquivo de log.'
        )}
      </footer>
    </aside>
  )
}
