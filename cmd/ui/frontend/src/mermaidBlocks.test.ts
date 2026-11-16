import { describe, expect, it } from 'vitest'
import {
  detectMermaidFailures,
  extractMermaidBlocks,
  normalizeBlockContent,
  replaceMermaidBlocks,
  validateMermaidRepair,
} from './mermaidBlocks'
import type { MermaidReplacement } from './types'

// Blocos medidos contra o pele 1.13.0 (o renderizador de verdade da UI):
// - VALID renderiza;
// - LINKSTYLE é a classe de produção do caso real (issue 29):
//   `linkStyle 0 stroke:#0A3;` falha nos DOIS renderizadores, e o conserto é
//   tirar o `;` (RENDER medido na tabela do ticket);
// - SEMANTIC é um modo que o scanner do servidor NÃO acusa (issue 16: omissão
//   sem gramática — "Trying to inactivate an inactive participant"): é a
//   classe que justifica o reparo sair do cliente.
const VALID = 'flowchart TD\n    A[origem] --> B[destino]'
const LINKSTYLE = 'flowchart TD\n    A --> B\n    linkStyle 0 stroke:#0A3;'
const LINKSTYLE_FIXED = 'flowchart TD\n    A --> B\n    linkStyle 0 stroke:#0A3'
const SEMANTIC =
  'sequenceDiagram\n    participant R1\n    R1->>R1: ping\n    activate R1\n    deactivate R1\n    deactivate R1'
const SEMANTIC_FIXED =
  'sequenceDiagram\n    participant R1\n    R1->>R1: ping\n    activate R1\n    deactivate R1'

describe('extractMermaidBlocks', () => {
  it('lista blocos em ordem de documento, com índice 1-based e fonte sem cerca', () => {
    const md = `prosa antes\n\n\`\`\`mermaid\n${VALID}\n\`\`\`\n\nprosa no meio\n\n\`\`\`mermaid\n${LINKSTYLE}\n\`\`\`\n`
    const blocks = extractMermaidBlocks(md)
    expect(blocks).toEqual([
      { index: 1, source: VALID },
      { index: 2, source: LINKSTYLE },
    ])
  })

  it('ignora cerca de outra linguagem e código inline', () => {
    const md = 'texto com `mermaid` inline\n\n```python\nprint("```mermaid")\n```\n'
    expect(extractMermaidBlocks(md)).toEqual([])
  })

  it('normaliza CRLF e remove UM \\n final como o backend (goldmark)', () => {
    const md = '```mermaid\r\nflowchart TD\r\n    A --> B\r\n```\r\n'
    expect(extractMermaidBlocks(md)[0].source).toBe('flowchart TD\n    A --> B')
    expect(normalizeBlockContent('a\n\n')).toBe('a\n')
    expect(normalizeBlockContent('a\r\nb\r\n')).toBe('a\nb')
  })

  it('aceita cerca ~~~ e bloco vazio, e fecha no EOF sem cerca de fechamento', () => {
    const md = `~~~mermaid\n${VALID}\n~~~\n\n\`\`\`mermaid\n\`\`\`\n\n\`\`\`mermaid\nflowchart TD\n    A --> B`
    const blocks = extractMermaidBlocks(md)
    expect(blocks.map((b) => b.source)).toEqual([VALID, '', 'flowchart TD\n    A --> B'])
  })
})

describe('replaceMermaidBlocks', () => {
  it('substitui só o índice pedido e preserva os demais blocos', () => {
    const md = `\`\`\`mermaid\n${VALID}\n\`\`\`\n\n\`\`\`mermaid\n${LINKSTYLE}\n\`\`\`\n`
    const replaced = replaceMermaidBlocks(md, [{ index: 2, source: LINKSTYLE_FIXED }])
    expect(replaced).not.toBeNull()
    expect(extractMermaidBlocks(replaced!)).toEqual([
      { index: 1, source: VALID },
      { index: 2, source: LINKSTYLE_FIXED },
    ])
  })

  it('recusa índice inválido ou duplicado', () => {
    const md = `\`\`\`mermaid\n${VALID}\n\`\`\`\n`
    expect(replaceMermaidBlocks(md, [{ index: 2, source: 'x' }])).toBeNull()
    expect(
      replaceMermaidBlocks(md, [
        { index: 1, source: 'x' },
        { index: 1, source: 'y' },
      ]),
    ).toBeNull()
  })

  it('recusa substituição que injetaria cerca e mudaria a estrutura', () => {
    const md = `\`\`\`mermaid\n${VALID}\n\`\`\`\n`
    expect(replaceMermaidBlocks(md, [{ index: 1, source: 'flowchart TD\n```\nA --> B' }])).toBeNull()
  })
})

describe('detectMermaidFailures (pele real)', () => {
  it('resposta válida não tem falha nenhuma', () => {
    expect(detectMermaidFailures(extractMermaidBlocks(`\`\`\`mermaid\n${VALID}\n\`\`\`\n`))).toEqual([])
  })

  it('acusa a classe de produção do linkStyle com a mensagem do pele', () => {
    const failures = detectMermaidFailures(extractMermaidBlocks(`\`\`\`mermaid\n${LINKSTYLE}\n\`\`\`\n`))
    expect(failures).toHaveLength(1)
    expect(failures[0].index).toBe(1)
    expect(failures[0].source).toBe(LINKSTYLE)
    expect(failures[0].error.trim()).not.toBe('')
  })

  it('acusa falha que o scanner do servidor não alcança (modo sem gramática)', () => {
    const failures = detectMermaidFailures(extractMermaidBlocks(`\`\`\`mermaid\n${SEMANTIC}\n\`\`\`\n`))
    expect(failures).toHaveLength(1)
    expect(failures[0].error).toContain('inactivate')
  })

  it('agrega os erros por resposta, com índices na ordem do documento', () => {
    const md = `\`\`\`mermaid\n${VALID}\n\`\`\`\n\n\`\`\`mermaid\n${LINKSTYLE}\n\`\`\`\n\n\`\`\`mermaid\n${SEMANTIC}\n\`\`\`\n`
    const failures = detectMermaidFailures(extractMermaidBlocks(md))
    expect(failures.map((f) => f.index)).toEqual([2, 3])
  })
})

describe('validateMermaidRepair', () => {
  const md = `\`\`\`mermaid\n${VALID}\n\`\`\`\n\n\`\`\`mermaid\n${LINKSTYLE}\n\`\`\`\n\n\`\`\`mermaid\n${SEMANTIC}\n\`\`\`\n`
  const failures = detectMermaidFailures(extractMermaidBlocks(md))

  it('aceita quando todo bloco reprovado é substituído por código que renderiza', () => {
    const replacements: MermaidReplacement[] = [
      { index: 2, source: LINKSTYLE_FIXED },
      { index: 3, source: SEMANTIC_FIXED },
    ]
    expect(validateMermaidRepair(md, failures, replacements)).toBe(true)
  })

  it('recusa candidato que continua inválido', () => {
    expect(validateMermaidRepair(md, failures, [{ index: 2, source: LINKSTYLE }, { index: 3, source: SEMANTIC_FIXED }])).toBe(false)
  })

  it('recusa quando um bloco reprovado ficou sem substituto', () => {
    expect(validateMermaidRepair(md, failures, [{ index: 2, source: LINKSTYLE_FIXED }])).toBe(false)
  })

  it('recusa substituição de bloco que não foi reprovado', () => {
    const replacements: MermaidReplacement[] = [
      { index: 1, source: VALID },
      { index: 2, source: LINKSTYLE_FIXED },
      { index: 3, source: SEMANTIC_FIXED },
    ]
    expect(validateMermaidRepair(md, failures, replacements)).toBe(false)
  })

  it('recusa candidato com índice duplicado (um substituto por bloco)', () => {
    const replacements: MermaidReplacement[] = [
      { index: 2, source: LINKSTYLE_FIXED },
      { index: 3, source: SEMANTIC_FIXED },
    ]
    expect(validateMermaidRepair(md, failures, [...replacements, { index: 2, source: 'x' }])).toBe(false)
  })
})
