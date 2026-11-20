import {
  ActionButton,
  ActionButtonGroup,
  Content,
  Heading,
  InlineAlert,
  ToggleButton,
  Tooltip,
  TooltipTrigger,
} from '@react-spectrum/s2'
import { css, iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Code from '@react-spectrum/s2/icons/Code'
import Download from '@react-spectrum/s2/icons/Download'
import Refresh from '@react-spectrum/s2/icons/Refresh'
import ZoomFitToScreen from '@react-spectrum/s2/icons/ZoomFitToScreen'
import ZoomIn from '@react-spectrum/s2/icons/ZoomIn'
import ZoomOut from '@react-spectrum/s2/icons/ZoomOut'
import { type PointerEvent, type ReactNode, useEffect, useRef, useState } from 'react'
import { render } from 'pele'
import { peleErrorMessage } from '../mermaidBlocks'
import CodeBlock, { mono } from './CodeBlock'

type DiagramPosition = { x: number; y: number }

const MIN_SCALE = 0.25
const MAX_SCALE = 3
const ZOOM_STEP = 1.2

// O tema do diagrama (docs/FRONTEND.md, Tema → Diagrama): o pele lê `--pele-*`, e
// o macro as define aqui com os tokens do S2. Elas saem como `light-dark(...)`,
// então o SVG na tela troca de tema sozinho, sem renderizar de novo.
const canvas = style({
  marginY: 12,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  borderRadius: 'xl',
  backgroundColor: 'gray-25',
  overflow: 'hidden',
  '--pele-bg': { type: 'backgroundColor', value: 'gray-25' },
  '--pele-fg': { type: 'color', value: 'neutral' },
  '--pele-muted': { type: 'color', value: 'neutral-subdued' },
  '--pele-line': { type: 'color', value: 'gray-600' },
  '--pele-surface': { type: 'backgroundColor', value: 'gray-100' },
  '--pele-surface-alt': { type: 'backgroundColor', value: 'gray-200' },
  '--pele-border': { type: 'borderColor', value: 'gray-400' },
  '--pele-accent': { type: 'color', value: 'accent' },
  '--pele-font': { type: 'fontFamily', value: 'sans' },
})
// A JetBrains Mono não é token do macro (ver CodeBlock): entra por `css()`.
const peleMono = css(`--pele-font-mono: 'JetBrains Mono', ui-monospace, Consolas, monospace;`)

const bar = style({
  display: 'flex',
  alignItems: 'center',
  gap: 4,
  paddingX: 8,
  paddingY: 4,
  borderWidth: 0,
  borderBottomWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  backgroundColor: 'layer-1',
})
const zoomValue = style({ minWidth: 48, textAlign: 'center', font: 'ui-sm', color: 'neutral-subdued' })
const grow = style({ flexGrow: 1 })
const viewportBox = style({
  position: 'relative',
  overflow: 'hidden',
  maxHeight: 520,
  padding: 16,
  textAlign: 'center',
  touchAction: 'none',
})
const canvasInner = style({ display: 'inline-block', transformOrigin: 'center', willChange: 'transform' })
// O SVG do pele traz o tamanho próprio: aqui ele cabe na largura da coluna.
const fitSvg = css(`& > svg { display: block; max-width: 100%; height: auto; }`)
const source = style({ margin: 0, padding: 16, overflowX: 'auto', textAlign: 'start', fontSize: 'code-sm', lineHeight: 'code', color: 'neutral' })
const failed = style({ marginY: 12 })
const wide = style({ width: 'full' })
// A causa crua pode vir multilinha (o parse error do pele traz a linha e uma
// linha de caret): o pre-wrap mantém o caret alinhado.
const cause = style({ whiteSpace: 'pre-wrap', fontSize: 'code-sm', lineHeight: 'code' })
const icon = iconStyle({ size: 'S' })

const THEME_COLORS = ['--pele-bg', '--pele-fg', '--pele-muted', '--pele-line', '--pele-surface', '--pele-surface-alt', '--pele-border', '--pele-accent']
const THEME_FONTS = ['--pele-font', '--pele-font-mono']

/**
 * Injeta no SVG o tema do app para que o download seja legível sozinho. O fundo
 * também vai: o SVG não pinta o dele, e um tema escuro sobre a página branca de
 * quem o abrir ficaria ilegível.
 */
export function standaloneSvg(svg: string, theme: Record<string, string>): string {
  const declarations = Object.entries(theme)
    .filter(([, value]) => value)
    .map(([name, value]) => `${name}:${value}`)
    .join(';')
  return svg.replace(/<svg\b([^>]*)>/, `<svg$1><style>svg{${declarations};background:var(--pele-bg)}</style>`)
}

/**
 * As `--pele-*` com o valor que o tema em uso dá a elas. Uma custom property
 * guarda o `light-dark(...)` cru; um elemento de prova que a usa em `color` ou
 * `font-family` devolve o rgb ou a família já resolvidos, que é o que o arquivo
 * baixado precisa (fora do app, um `light-dark` não resolveria para o tema certo).
 */
function resolvedTheme(host: HTMLElement): Record<string, string> {
  const probe = document.createElement('span')
  host.append(probe)
  const read = (name: string, property: 'color' | 'fontFamily') => {
    probe.style[property] = `var(${name})`
    return getComputedStyle(probe)[property]
  }
  const theme = {
    ...Object.fromEntries(THEME_COLORS.map((name) => [name, read(name, 'color')])),
    ...Object.fromEntries(THEME_FONTS.map((name) => [name, read(name, 'fontFamily')])),
  }
  probe.remove()
  return theme
}

function Tool({ label, onPress, children }: { label: string; onPress: () => void; children: ReactNode }) {
  return (
    <TooltipTrigger>
      <ActionButton isQuiet size="S" aria-label={label} onPress={onPress}>
        {children}
      </ActionButton>
      <Tooltip>{label}</Tooltip>
    </TooltipTrigger>
  )
}

// Renderiza um Bloco de diagrama (sintaxe Mermaid) como SVG inline, pelo pele.
//
// O render do pele é síncrono, não toca no DOM e não emite `id` no SVG, então o
// SVG existe já no primeiro render, logo o elemento nunca aparece vazio — era isso
// que o render assíncrono do mermaid (mais o chunk do tipo de diagrama carregado
// por demanda) deixava em branco por segundos.
//
// Segurança: o fonte é escrito pelo modelo. O pele escapa o texto de rótulo e só
// deixa passar link de esquema conhecido (`linkSchemes` default: http, https,
// mailto, tel), com `rel="noopener"` — um `click ... href "javascript:..."` não
// vira link. O tema sai das variáveis `--pele-*` do `canvas`, acima.
// Tudo medido em `.agent/ui/assets/03-fallback-diagrama/probe-pele.txt`.
export default function Mermaid({ code }: { code: string }) {
  const [showSource, setShowSource] = useState(false)
  const [scale, setScale] = useState(1)
  const [position, setPosition] = useState<DiagramPosition>({ x: 0, y: 0 })
  const [dragging, setDragging] = useState(false)
  const dragStart = useRef<{ pointerId: number; clientX: number; clientY: number } | null>(null)
  const root = useRef<HTMLDivElement>(null)
  const viewport = useRef<HTMLDivElement>(null)
  const diagram = useRef<HTMLDivElement>(null)
  let svg = ''
  let failure: unknown = null
  try {
    svg = render(code).svg
  } catch (e) {
    failure = e
  }

  const setZoom = (nextScale: number) => setScale(Math.min(MAX_SCALE, Math.max(MIN_SCALE, nextScale)))

  const resetViewport = () => {
    setScale(1)
    setPosition({ x: 0, y: 0 })
  }

  // Cabe o diagrama, no tamanho de layout (a transformação não o muda), na caixa
  // do viewport, que tem altura máxima: um diagrama alto encolhe até caber. O 16
  // é o padding do viewport.
  const fit = () => {
    const box = viewport.current
    const el = diagram.current
    if (!box || !el) return
    const room = { w: box.clientWidth - 32, h: box.clientHeight - 32 }
    setZoom(Math.min(room.w / el.offsetWidth, room.h / el.offsetHeight))
    // A transformação gira em torno do centro da caixa de layout, que num diagrama
    // alto fica abaixo da janela do viewport: desce o centro dele para o do viewport.
    setPosition({ x: 0, y: box.clientHeight / 2 - (16 + el.offsetHeight / 2) })
  }

  const beginDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return
    dragStart.current = { pointerId: event.pointerId, clientX: event.clientX, clientY: event.clientY }
    event.currentTarget.setPointerCapture(event.pointerId)
    setDragging(true)
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
    setDragging(false)
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
  }, [showSource])

  const download = () => {
    if (!root.current) return
    const file = new Blob([standaloneSvg(svg, resolvedTheme(root.current))], { type: 'image/svg+xml;charset=utf-8' })
    const url = URL.createObjectURL(file)
    const link = document.createElement('a')
    link.href = url
    link.download = 'diagrama.svg'
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  }

  if (failure) {
    // o objeto inteiro fica no devtools do `wails3 dev`; só a mensagem vai para a
    // tela, que é onde o usuário precisa dela.
    console.error('diagrama: render falhou', failure)
    // Bloco não renderizado: o aviso usa o mesmo triplo o-quê/por-quê/próximo
    // dos outros erros do app, e o fonte vai como bloco de código — em JSX, para
    // o React escapar, e não por innerHTML.
    return (
      <div className={failed}>
        <InlineAlert variant="negative" styles={wide}>
          <Heading>O diagrama não renderizou</Heading>
          <Content>
            <p className={`${cause} ${mono}`}>{peleErrorMessage(failure)}</p>
            <p>O fonte está abaixo; peça o diagrama de novo se quiser tentar outra forma.</p>
          </Content>
        </InlineAlert>
        <CodeBlock language="mermaid" code={code} />
      </div>
    )
  }

  return (
    <div ref={root} className={`${canvas} ${peleMono}`}>
      <div className={bar}>
        {!showSource && (
          <>
            <ActionButtonGroup size="S" isQuiet aria-label="Controle do diagrama">
              <Tool label="Menos zoom" onPress={() => setZoom(scale / ZOOM_STEP)}>
                <ZoomOut styles={icon} />
              </Tool>
              <Tool label="Mais zoom" onPress={() => setZoom(scale * ZOOM_STEP)}>
                <ZoomIn styles={icon} />
              </Tool>
              <Tool label="Redefinir zoom e posição" onPress={resetViewport}>
                <Refresh styles={icon} />
              </Tool>
              <Tool label="Ajustar à tela" onPress={fit}>
                <ZoomFitToScreen styles={icon} />
              </Tool>
            </ActionButtonGroup>
            <output className={zoomValue} aria-live="polite">
              {Math.round(scale * 100)}%
            </output>
          </>
        )}
        <span className={grow} />
        <TooltipTrigger>
          <ToggleButton isQuiet size="S" aria-label="Ver código" isSelected={showSource} onChange={setShowSource}>
            <Code styles={icon} />
          </ToggleButton>
          <Tooltip>{showSource ? 'Ver diagrama' : 'Ver código'}</Tooltip>
        </TooltipTrigger>
        <ActionButtonGroup size="S" isQuiet aria-label="Arquivo do diagrama">
          <Tool label="Baixar SVG" onPress={download}>
            <Download styles={icon} />
          </Tool>
        </ActionButtonGroup>
      </div>
      {showSource ? (
        <pre className={`${source} ${mono}`}>{code}</pre>
      ) : (
        <div
          ref={viewport}
          className={viewportBox}
          style={{ cursor: dragging ? 'grabbing' : 'grab' }}
          onPointerDown={beginDrag}
          onPointerMove={drag}
          onPointerUp={endDrag}
          onPointerCancel={endDrag}
          aria-label="Use a roda do mouse para zoom e arraste para mover o diagrama"
        >
          <div className={canvasInner} style={{ transform: `translate(${position.x}px, ${position.y}px) scale(${scale})` }}>
            <div ref={diagram} className={fitSvg} data-diagram dangerouslySetInnerHTML={{ __html: svg }} />
          </div>
        </div>
      )}
    </div>
  )
}
