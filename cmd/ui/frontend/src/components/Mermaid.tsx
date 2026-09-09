import { useEffect, useRef, useState } from 'react'
import mermaid from 'mermaid'

// Configuração do mermaid para o tema dark da aplicação.
// startOnLoad:false -> a renderização é manual (via render()), evitando
// que o mermaid tente processar tudo automaticamente.
mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'loose',
})

let counter = 0

// Renderiza um diagrama mermaid como SVG inline, no client-side.
// Usa mermaid.render() de forma assíncrona (o mermaid é async), o que
// evita o erro 'runSync finished async' do rehype-mermaid (SSR).
export default function Mermaid({ code }: { code: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const [id] = useState(() => 'mermaid-' + ++counter)

  useEffect(() => {
    let cancelled = false
    async function render() {
      try {
        const { svg } = await mermaid.render(id, code)
        if (!cancelled && ref.current) {
          ref.current.innerHTML = svg
        }
      } catch (e) {
        // fallback: mostra o código bruto se o diagrama falhar
        if (!cancelled && ref.current) {
          ref.current.innerHTML = `<pre class="mermaid-error">${code}</pre>`
        }
      }
    }
    render()
    return () => {
      cancelled = true
    }
  }, [code, id])

  return <div ref={ref} className="mermaid" />
}