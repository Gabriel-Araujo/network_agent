import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import Markdown from './Markdown'
import { extractMermaidBlocks } from '../mermaidBlocks'

// Fixtures medidas com o pele 1.13.0 (ver mermaidBlocks.test.ts): VALID
// renderiza, SEMANTIC é rejeitado ("Trying to inactivate an inactive
// participant") — modo que o scanner do servidor não acusa.
const VALID = 'flowchart TD\n    A[origem] --> B[destino]'
const SEMANTIC =
  'sequenceDiagram\n    participant R1\n    R1->>R1: ping\n    activate R1\n    deactivate R1\n    deactivate R1'

describe('Markdown × extração compartilhada', () => {
  it('renderiza o SVG de cada bloco pela MESMA extração da detecção', () => {
    const content = `prosa\n\n\`\`\`mermaid\n${VALID}\n\`\`\`\n\n\`\`\`mermaid\n${VALID.replace(/origem/g, 'outro')}\n\`\`\`\n`
    // dois blocos na extração = dois renders
    expect(extractMermaidBlocks(content)).toHaveLength(2)
    const html = renderToStaticMarkup(<Markdown content={content} />)
    expect(html.match(/<svg/g)).toHaveLength(2)
    expect(html).not.toContain('mermaid--failed')
  })

  it('bloco rejeitado mostra erro E fonte, sem derrubar os blocos vizinhos', () => {
    const content = `\`\`\`mermaid\n${VALID}\n\`\`\`\n\n\`\`\`mermaid\n${SEMANTIC}\n\`\`\`\n`
    const html = renderToStaticMarkup(<Markdown content={content} />)
    // o bloco bom renderiza...
    expect(html).toContain('<svg')
    // ...e o rejeitado mantém o erro e o fonte originais visíveis
    expect(html).toContain('mermaid--failed')
    expect(html).toContain('O diagrama não renderizou')
    expect(html).toContain('inactivate')
    expect(html).toContain('participant R1')
  })

  it('durante o streaming mostra o fonte cru e não renderiza SVG', () => {
    const content = `\`\`\`mermaid\n${VALID}\n\`\`\`\n`
    const html = renderToStaticMarkup(<Markdown content={content} streaming />)
    expect(html).not.toContain('<svg')
    expect(html).toContain('mermaid-source')
    expect(html).toContain('A[origem]')
  })
})
