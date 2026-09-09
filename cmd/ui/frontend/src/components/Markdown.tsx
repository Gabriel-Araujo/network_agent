import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import Mermaid from './Mermaid'

interface MarkdownProps {
  content: string
  /** Enquanto true, o texto ainda está chegando: blocos que só fazem sentido
   *  completos (mermaid) não são renderizados, para evitar erro/flicker. */
  streaming?: boolean
}

// Renderiza markdown com suporte a GFM (tabelas, listas de tarefas) e mermaid.
// Os estilos dos elementos ficam em App.css (classe .md-*), combinando com
// o dark glass azulado da aplicação.
// Mermaid é renderizado client-side pelo componente <Mermaid> (via pacote
// 'mermaid'), evitando a dependência de playwright do rehype-mermaid (SSR).
export default function Markdown({ content, streaming = false }: MarkdownProps) {
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
              const code = String(children).replace(/\n$/, '')
              // Durante o streaming o diagrama ainda está incompleto: mostra o
              // código cru e só renderiza o SVG quando a resposta fecha.
              if (streaming) {
                return <pre className="mermaid-pending">{code}</pre>
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