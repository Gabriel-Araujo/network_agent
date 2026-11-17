import { ArrowCounterClockwise, Code, DownloadSimple, Graph, Minus, Plus } from '@phosphor-icons/react'
import { type PointerEvent, useCallback, useEffect, useRef, useState } from 'react'
import { render } from 'pele'
import { peleErrorMessage } from '../mermaidBlocks'

type DiagramView = 'diagram' | 'source'

type DiagramPosition = { x: number; y: number }

const MIN_SCALE = 0.5
const MAX_SCALE = 3
const ZOOM_STEP = 1.2

const peleThemeVariables = [
  ['--pele-bg', '--bg-base'],
  ['--pele-fg', '--text'],
  ['--pele-muted', '--text-dim'],
  ['--pele-line', '--text-dim'],
  ['--pele-surface', '--bg-elev-2'],
  ['--pele-surface-alt', '--bg-elev'],
  ['--pele-border', '--border-strong'],
  ['--pele-accent', '--accent'],
  ['--pele-font', '--font'],
  ['--pele-font-mono', '--font-mono', 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace'],
] as const

/** Injeta no SVG o tema do app para que o download seja leg?vel sozinho. */
export function standaloneSvg(svg: string, theme: Record<string, string>): string {
  const declarations = Object.entries(theme)
    .filter(([, value]) => value)
    .map(([name, value]) => `${name}:${value}`)
    .join(';')
  return svg.replace(/<svg\b([^>]*)>/, `<svg$1><style>svg{${declarations}}</style>`)
}


// Renderiza um Bloco de diagrama (sintaxe Mermaid) como SVG inline, pelo pele.
//
// O render do pele é síncrono, não toca no DOM e não emite `id` no SVG, então o
// componente é função pura: sem estado, sem efeito, sem contador de id. O SVG
// existe já no primeiro render, logo o elemento nunca aparece vazio — era isso
// que o render assíncrono do mermaid (mais o chunk do tipo de diagrama carregado
// por demanda) deixava em branco por segundos.
//
// Segurança: o fonte é escrito pelo modelo. O pele escapa o texto de rótulo e só
// deixa passar link de esquema conhecido (`linkSchemes` default: http, https,
// mailto, tel), com `rel="noopener"` — um `click ... href "javascript:..."` não
// vira link. O tema sai das variáveis `--pele-*` definidas no App.css.
// Tudo medido em `.agent/ui/assets/03-fallback-diagrama/probe-pele.txt`.
export default function Mermaid({ code }: { code: string }) {
  const [view, setView] = useState<DiagramView>('diagram')
  const [scale, setScale] = useState(1)
  const [position, setPosition] = useState<DiagramPosition>({ x: 0, y: 0 })
  const dragStart = useRef<{ pointerId: number; clientX: number; clientY: number } | null>(null)
  const viewport = useRef<HTMLDivElement>(null)
  let svg: string
  try {
    svg = render(code).svg
  } catch (e) {
    // o objeto inteiro fica no devtools do `wails dev`; só a mensagem vai para a
    // tela, que é onde o usuário precisa dela.
    console.error('diagrama: render falhou', e)
    // Bloco não renderizado: o aviso usa o mesmo triplo o-quê/por-quê/próximo
    // dos outros erros do app, e o fonte vai como texto — em JSX, para o React
    // escapar, e não por innerHTML.
    return (
      <div className="mermaid mermaid--failed" role="alert">
        <p className="error__title">O diagrama não renderizou</p>
        <p className="error__why">{peleErrorMessage(e)}</p>
        <p className="error__next">
          O fonte está abaixo; peça o diagrama de novo se quiser tentar outra forma.
        </p>
        <pre className="mermaid-source">{code}</pre>
      </div>
    )
  }

  const setZoom = useCallback((nextScale: number) => {
    setScale(Math.min(MAX_SCALE, Math.max(MIN_SCALE, nextScale)))
  }, [])

  const resetViewport = () => {
    setScale(1)
    setPosition({ x: 0, y: 0 })
  }

  const beginDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return
    dragStart.current = { pointerId: event.pointerId, clientX: event.clientX, clientY: event.clientY }
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  const drag = (event: PointerEvent<HTMLDivElement>) => {
    const start = dragStart.current
    if (!start || start.pointerId !== event.pointerId) return
    setPosition((current) => ({
      x: current.x + event.clientX - start.clientX,
      y: current.y + event.clientY - start.clientY,
    }))
    dragStart.current = { ...start, clientX: event.clientX, clientY: event.clientY }
  }

  const endDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (dragStart.current?.pointerId !== event.pointerId) return
    dragStart.current = null
    event.currentTarget.releasePointerCapture(event.pointerId)
  }

  useEffect(() => {
    const element = viewport.current
    if (!element) return

    const zoomWithWheel = (event: WheelEvent) => {
      // O listener nativo nao-passivo bloqueia a rolagem da conversa no Wails.
      event.preventDefault()
      // Uma volta comum da roda (~100) altera cerca de 10%; trackpads mantem
      // a granularidade dos seus deltas menores.
      setScale((current) => Math.min(MAX_SCALE, Math.max(MIN_SCALE, current * Math.exp(-event.deltaY * 0.001))))
    }

    element.addEventListener('wheel', zoomWithWheel, { passive: false })
    return () => element.removeEventListener('wheel', zoomWithWheel)
  }, [])

  const download = () => {
    const styles = getComputedStyle(document.documentElement)
    const theme = Object.fromEntries(
      peleThemeVariables.map(([peleVariable, appVariable, fallback = '']) => [
        peleVariable,
        styles.getPropertyValue(appVariable).trim() || fallback,
      ]),
    )
    const file = new Blob([standaloneSvg(svg, theme)], { type: 'image/svg+xml;charset=utf-8' })
    const url = URL.createObjectURL(file)
    const link = document.createElement('a')
    link.href = url
    link.download = 'diagrama.svg'
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  }

  return (
    <div className="mermaid">
      <div className="mermaid__toolbar">
        {view === 'diagram' && (
          <div className="mermaid__viewport-controls" role="group" aria-label="Controle do diagrama">
            <button type="button" className="mermaid__viewport-button" onClick={() => setZoom(scale / ZOOM_STEP)} aria-label="Menos zoom" title="Menos zoom">
              <Minus size={16} aria-hidden="true" />
            </button>
            <output className="mermaid__zoom" aria-live="polite">{Math.round(scale * 100)}%</output>
            <button type="button" className="mermaid__viewport-button" onClick={() => setZoom(scale * ZOOM_STEP)} aria-label="Mais zoom" title="Mais zoom">
              <Plus size={16} aria-hidden="true" />
            </button>
            <button type="button" className="mermaid__viewport-button" onClick={resetViewport} aria-label="Redefinir zoom e posicao" title="Redefinir zoom e posicao">
              <ArrowCounterClockwise size={16} aria-hidden="true" />
            </button>
          </div>
        )}
        <div className="mermaid__view-controls" role="group" aria-label="Visualizacao do diagrama">
          <button type="button" className={`mermaid__view ${view === 'diagram' ? 'mermaid__view--active' : ''}`} aria-label="Visualizar diagrama" title="Visualizar diagrama" aria-pressed={view === 'diagram'} onClick={() => setView('diagram')}>
            <Graph size={16} aria-hidden="true" />
          </button>
          <button type="button" className={`mermaid__view ${view === 'source' ? 'mermaid__view--active' : ''}`} aria-label="Visualizar codigo" title="Visualizar codigo" aria-pressed={view === 'source'} onClick={() => setView('source')}>
            <Code size={16} aria-hidden="true" />
          </button>
          <button type="button" className="mermaid__download" onClick={download} aria-label="Baixar SVG" title="Baixar SVG">
            <DownloadSimple size={16} aria-hidden="true" />
          </button>
        </div>
      </div>
      {view === 'diagram' ? (
        <>
          <div
            ref={viewport}
            className="mermaid__viewport"
            onPointerDown={beginDrag}
            onPointerMove={drag}
            onPointerUp={endDrag}
            onPointerCancel={endDrag}
            aria-label="Use a roda do mouse para zoom e arraste para mover o diagrama"
          >
            <div className="mermaid__canvas" style={{ transform: `translate(${position.x}px, ${position.y}px) scale(${scale})` }}>
              <div className="mermaid__diagram" dangerouslySetInnerHTML={{ __html: svg }} />
            </div>
          </div>
        </>
      ) : <pre className="mermaid-source">{code}</pre>}
    </div>
  )
}
