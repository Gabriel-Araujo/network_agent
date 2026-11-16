import { useMemo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import Mermaid from './Mermaid'
import { extractMermaidBlocks, normalizeBlockContent } from '../mermaidBlocks'

interface MarkdownProps {
  content: string
  /** Enquanto true, o texto ainda está chegando: blocos que só fazem sentido
   *  completos (mermaid) não são renderizados, para evitar erro/flicker. */
  streaming?: boolean
}

// Renderiza markdown com suporte a GFM (tabelas, listas de tarefas) e Blocos de
// diagrama. Os estilos dos elementos ficam em App.css (classe .md-*),
// combinando com o dark glass azulado da aplicação.
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
    <div className="md">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noreferrer">
              {children}
            </a>
          ),
          table: ({ children }) => (
            <div className="md-table-wrap">
              <table>{children}</table>
            </div>
          ),
          code: ({ className, children, ...props }) => {
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
                return <pre className="mermaid-source">{code}</pre>
              }
              return <Mermaid code={code} />
            }
            if (isInline) {
              return <code className={className}>{children}</code>
            }
            // bloco de código normal
            return (
              <pre className={className}>
                <code {...props}>{children}</code>
              </pre>
            )
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}