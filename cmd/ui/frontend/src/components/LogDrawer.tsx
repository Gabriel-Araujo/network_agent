import { useEffect, useRef, useState } from 'react'
import { X } from '@phosphor-icons/react'
import { Clipboard } from '@wailsio/runtime'
import { App } from '../../bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui'
import { inWails } from '../types'

// A gaveta de Logs: as MESMAS linhas que o CLI mostra no terminal, porque a
// fonte é o sink de console — o Go escreve nos dois destinos no mesmo Handle
// (ticket 06). Daí decorre o que ela não mostra: nada de `source`, valor
// truncado em 512, nada de TRACE. Quem quer isso lê o arquivo, e é para isso
// que serve o "copiar o caminho".

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

// Uma linha, com a cor só no nível: o resto competiria com a mensagem, e é
// assim que o terminal a mostra.
function LogLine({ line }: { line: string }) {
  const level = levelOf(line)
  if (!level) return <div className="logs__line">{line}</div>
  return (
    <div className="logs__line">
      {line.slice(0, LEVEL_AT)}
      <span className={`logs__level logs__level--${level}`}>
        {line.slice(LEVEL_AT, LEVEL_AT + LEVEL_LEN)}
      </span>
      {line.slice(LEVEL_AT + LEVEL_LEN)}
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
  const [copied, setCopied] = useState(false)
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

  if (!open) return null

  // O caminho vai para um grep ou para o anexo de um ticket; abrir no editor
  // seria "um local" que não é do app (ticket 06).
  const copyPath = () => {
    if (!path || !inWails()) return
    Clipboard.SetText(path)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <aside className="logs" aria-label="Logs da execução">
      <header className="logs__head">
        <span className="logs__title">Logs</span>
        <span className="logs__count">{lines.length}</span>
        <span className="logs__spacer" />
        <button
          className="icon-btn icon-btn--xs"
          onClick={onClose}
          aria-label="Fechar os logs"
          title="Fechar"
        >
          <X size={14} weight="bold" aria-hidden="true" />
        </button>
      </header>

      <div
        className="logs__body"
        ref={bodyRef}
        onScroll={(e) => {
          const el = e.currentTarget
          pinned.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24
        }}
      >
        {lines.length === 0 ? (
          <p className="logs__empty">Nenhuma linha nesta execução.</p>
        ) : (
          lines.map((line, i) => <LogLine key={i} line={line} />)
        )}
      </div>

      <footer className="logs__foot">
        <button
          className="link-btn"
          onClick={copyPath}
          disabled={!path}
          title={path || 'Esta execução não abriu arquivo de log'}
        >
          {copied ? 'Caminho copiado' : 'Copiar o caminho do arquivo'}
        </button>
      </footer>
    </aside>
  )
}
