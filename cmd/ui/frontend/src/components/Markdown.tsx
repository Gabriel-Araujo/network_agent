import { ComponentPropsWithoutRef, ElementType, useMemo } from 'react'
import ReactMarkdown, { Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { css, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import CodeBlock, { mono } from './CodeBlock'
import Mermaid from './Mermaid'
import { extractMermaidBlocks, normalizeBlockContent } from '../mermaidBlocks'

const root = style({
  font: 'ui',
  lineHeight: 'body',
  color: 'neutral',
  wordBreak: 'break-word',
})
// A margem não vaza para fora da bolha: o primeiro filho sobe e o último desce até a borda.
const edges = css(`& > :first-child { margin-top: 0; } & > :last-child { margin-bottom: 0; }`)
const p = style({ marginY: 0, marginBottom: 12 })
const strong = style({ fontWeight: 'bold', color: 'neutral' })
const list = style({ marginY: 0, marginBottom: 12, paddingStart: 24 })
const li = style({ marginY: 4, color: 'neutral' })
const a = style({
  color: 'accent',
  textDecoration: 'underline',
  outlineStyle: { default: 'none', ':focus-visible': 'solid' },
  outlineColor: 'focus-ring',
})
const quote = style({
  marginY: 12,
  marginX: 0,
  paddingX: 16,
  borderWidth: 0,
  borderStartWidth: 4,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  color: 'neutral-subdued',
})
const hr = style({ marginY: 16, borderWidth: 0, borderTopWidth: 1, borderStyle: 'solid', borderColor: 'gray-300' })
const h1 = style({ font: 'heading-sm', marginTop: 24, marginBottom: 12 })
const h2 = style({ font: 'heading-xs', marginTop: 24, marginBottom: 12 })
const h3 = style({ font: 'heading-2xs', marginTop: 20, marginBottom: 8 })
const h4 = style({ font: 'heading-2xs', marginTop: 16, marginBottom: 8 })
const inlineCode = style({
  fontSize: 'code-sm',
  paddingX: 4,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  borderRadius: 'sm',
  backgroundColor: 'layer-1',
  color: 'neutral',
})
const tableWrap = style({
  marginY: 12,
  overflowX: 'auto',
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  borderRadius: 'xl',
})
const table = style({ width: 'full', borderCollapse: 'collapse', font: 'ui' })
const th = style({
  paddingX: 12,
  paddingY: 8,
  textAlign: 'start',
  fontWeight: 'bold',
  color: 'neutral',
  backgroundColor: 'layer-1',
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
})
const td = style({
  paddingX: 12,
  paddingY: 8,
  textAlign: 'start',
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
})
const task = style({ marginEnd: 8, verticalAlign: 'middle' })

type Tag = ElementType
// Uma classe por elemento: `el('p', p)` é o `<p className={p}>` do react-markdown.
const el =
  (Tag: Tag, className: string) =>
  ({ node: _node, ...props }: ComponentPropsWithoutRef<Tag> & { node?: unknown }) => (
    <Tag className={className} {...props} />
  )

const elements: Components = {
  p: el('p', p),
  strong: el('strong', strong),
  ul: el('ul', list),
  ol: el('ol', list),
  li: el('li', li),
  blockquote: el('blockquote', quote),
  hr: el('hr', hr),
  h1: el('h1', h1),
  h2: el('h2', h2),
  h3: el('h3', h3),
  h4: el('h4', h4),
  h5: el('h4', h4),
  h6: el('h4', h4),
  table: ({ children }) => (
    <div className={tableWrap}>
      <table className={table}>{children}</table>
    </div>
  ),
  th: el('th', th),
  td: el('td', td),
  input: el('input', task),
  a: ({ href, children }) => (
    <a className={a} href={href} target="_blank" rel="noreferrer">
      {children}
    </a>
  ),
  // O bloco de código tem o próprio invólucro (CodeBlock): sem <pre> de fora.
  pre: ({ children }) => <>{children}</>,
}

interface MarkdownProps {
  content: string
  /** Enquanto true, o texto ainda está chegando: blocos que só fazem sentido
   *  completos (mermaid) não são renderizados, para evitar erro/flicker. */
  streaming?: boolean
}

// Renderiza markdown com suporte a GFM (tabelas, listas de tarefas) e Blocos de
// diagrama. Cada elemento recebe uma classe do macro `style` pelo `components`
// (docs/FRONTEND.md, Componentes), com os tokens do S2.
// O Bloco de diagrama é renderizado client-side pelo componente <Mermaid>, que
// usa o 'pele' — a sintaxe continua sendo Mermaid, a biblioteca não.
//
// Os blocos vêm da MESMA extração que a detecção do reparo automático
// (mermaidBlocks.ts): fences, índices e fonte são um só. O <Mermaid> recebe o
// fonte extraído em ordem; se a extração e o parse do react-markdown
// divergirem num bloco, a guarda abaixo cai para o texto do próprio bloco —
// o render nunca mostra o diagrama errado, e a divergência só faz a detecção
// não acusar (fail-safe).
export default function Markdown({ content, streaming = false }: MarkdownProps) {
  const blocks = useMemo(() => extractMermaidBlocks(content), [content])
  // Contador de blocos mermaid vistos nesta passada de render (ordem de
  // documento = ordem de extração).
  let mermaidSeen = 0
  return (
    <div className={`${root} ${edges}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          ...elements,
          code: ({ className, children }) => {
            // bloco de código com linguagem mermaid
            const match = /language-(\w+)/.exec(className || '')
            const isInline = !className && typeof children === 'string' && !children.includes('\n')
            if (match?.[1] === 'mermaid') {
              const raw = normalizeBlockContent(String(children))
              const block = blocks[mermaidSeen++]
              const code = block && block.source === raw ? block.source : raw
              // Durante o streaming o diagrama ainda está incompleto: mostra o
              // código cru e só renderiza o SVG quando a resposta fecha.
              if (streaming) {
                return <CodeBlock language="mermaid" code={code} />
              }
              return <Mermaid code={code} />
            }
            if (isInline) {
              return <code className={`${inlineCode} ${mono}`}>{children}</code>
            }
            // bloco de código normal
            return <CodeBlock language={match?.[1] ?? ''} code={String(children).replace(/\n$/, '')} />
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}