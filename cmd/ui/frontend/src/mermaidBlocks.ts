import { render } from 'pele'
import type { MermaidFailure, MermaidReplacement } from './types'

// A interpretação Markdown dos Blocos de diagrama — fences, índices e fonte —
// é UM módulo, compartilhado entre a renderização (<Markdown>) e a detecção de
// falhas do reparo automático. Ambos consomem a MESMA lista, então o bloco que
// a detecção acusa é exatamente o que a tela renderiza.
//
// A extração espelha a do backend (`mermaidSources`, cmd/ui/mermaid_source.go,
// goldmark): conteúdo entre cercas com `\r\n` normalizado e UM `\n` final
// removido, índice 1-based em ordem de documento. O backend confere cada falha
// contra o texto persistido com a mesma regra — divergência aqui não corrompe
// nada (o reparo é recusado com erro visível), mas este espelhamento é o que
// faz os dois lados concordarem no caso comum.

export interface MermaidBlock {
  /** Índice 1-based do bloco entre os blocos de diagrama da resposta. */
  index: number
  /** Código Mermaid, sem cercas. */
  source: string
}

interface ScannedBlock extends MermaidBlock {
  /** Linha da cerca de abertura (0-based). */
  open: number
  /** Linha da cerca de fechamento, ou lines.length quando não fecha. */
  close: number
  /** Indentação da cerca de abertura (re-indentação da substituição). */
  prefix: string
}

/** Normaliza o conteúdo de um bloco como o backend o extrai (goldmark). */
export function normalizeBlockContent(raw: string): string {
  return raw.replace(/\r\n/g, '\n').replace(/\n$/, '')
}

interface Fence {
  char: string
  len: number
  lang: string
  prefix: string
}

// Cercas de até 3 espaços de indentação (top-level do CommonMark). Cercas
// dentro de lista/blockquote não são alcançadas — e a guarda de fonte do
// <Markdown> impede render errado quando a extração e o parse do
// react-markdown divergem: nesse caso a detecção não acusa (fail-safe), nunca
// acusa o bloco errado.
function openFence(line: string): Fence | null {
  // `.` não casa `\r`, então o CR de uma linha CRLF vem explícito no fim.
  const m = /^( {0,3})(`{3,}|~{3,})(.*)\r?$/.exec(line)
  if (!m) return null
  const prefix = m[1]
  const run = m[2]
  const info = m[3]
  const char = run[0]
  // CommonMark: a info string de crase não pode conter crase.
  if (char === '`' && info.includes('`')) return null
  const lang = info.trim().split(/[\s,]+/)[0] ?? ''
  return { char, len: run.length, lang, prefix }
}

function closeFence(line: string, open: Fence): boolean {
  const m = /^( {0,3})(`{3,}|~{3,})[ \t]*\r?$/.exec(line)
  return !!m && m[2][0] === open.char && m[2].length >= open.len
}

function scanMermaid(markdown: string): ScannedBlock[] {
  const lines = markdown.split('\n')
  const out: ScannedBlock[] = []
  let i = 0
  while (i < lines.length) {
    const fence = openFence(lines[i])
    if (!fence) {
      i++
      continue
    }
    let j = i + 1
    while (j < lines.length && !closeFence(lines[j], fence)) j++
    if (fence.lang === 'mermaid') {
      const content = lines.slice(i + 1, j)
      let raw = content.join('\n')
      // Com cerca de fechamento, a última linha de conteúdo perdeu o `\n`
      // que a separava da cerca na divisão por linhas; repõe um só.
      if (j < lines.length && content.length > 0) raw += '\n'
      out.push({
        index: out.length + 1,
        source: normalizeBlockContent(raw),
        open: i,
        close: j,
        prefix: fence.prefix,
      })
    }
    i = j < lines.length ? j + 1 : j
  }
  return out
}

/** Lista os blocos de diagrama do markdown, na ordem do documento. */
export function extractMermaidBlocks(markdown: string): MermaidBlock[] {
  return scanMermaid(markdown).map(({ index, source }) => ({ index, source }))
}

/**
 * Recompõe o markdown aplicando as substituições nos blocos informados.
 * Devolve null quando a entrada é inválida (índice fora do intervalo,
 * duplicado) ou quando a substituição mudaria a estrutura Markdown — é o
 * mesmo pós-conferência do backend (`replaceMermaidSources`).
 */
export function replaceMermaidBlocks(
  markdown: string,
  replacements: MermaidReplacement[],
): string | null {
  const blocks = scanMermaid(markdown)
  const wanted = new Map<number, string>()
  for (const r of replacements) {
    if (r.index < 1 || r.index > blocks.length || wanted.has(r.index)) return null
    wanted.set(r.index, r.source)
  }
  const lines = markdown.split('\n')
  const out: string[] = []
  let cursor = 0
  for (const b of blocks) {
    const source = wanted.get(b.index)
    if (source == null) continue
    out.push(...lines.slice(cursor, b.open + 1))
    // Re-indenta a substituição pelo prefixo da cerca, como o backend.
    out.push(...source.replace(/\n/g, '\n' + b.prefix).split('\n'))
    if (b.close < lines.length) out.push(lines[b.close])
    cursor = b.close < lines.length ? b.close + 1 : b.close
  }
  out.push(...lines.slice(cursor))
  const composed = out.join('\n')
  const after = scanMermaid(composed)
  if (after.length !== blocks.length) return null
  for (const b of after) {
    const expected = wanted.get(b.index) ?? blocks[b.index - 1].source
    if (b.source !== expected) return null
  }
  return composed
}

/** A causa do pele, legível — a mesma exibida no Bloco não renderizado. */
export function peleErrorMessage(e: unknown): string {
  const msg = e instanceof Error ? e.message : String(e)
  // O aviso não pode ficar sem corpo: seria falha silenciosa.
  return msg.trim() || 'O pele não informou a causa.'
}

/**
 * Renderiza cada bloco com o pele e agrega os erros desta resposta. É a
 * autoridade do cliente: acusa o que a tela de fato recusa, mesmo quando o
 * scanner do servidor não alcançou (a classe das omissões medidas).
 */
export function detectMermaidFailures(blocks: MermaidBlock[]): MermaidFailure[] {
  const failures: MermaidFailure[] = []
  for (const b of blocks) {
    try {
      render(b.source)
    } catch (e) {
      failures.push({ index: b.index, source: b.source, error: peleErrorMessage(e) })
    }
  }
  return failures
}

/**
 * Decide o aceite do candidato: recompõe o texto original, verifica TODOS os
 * blocos de novo com o pele e só aceita se os blocos originalmente defeituosos
 * foram substituídos por código que renderiza e os demais continuam renderizando.
 */
export function validateMermaidRepair(
  markdown: string,
  failures: MermaidFailure[],
  replacements: MermaidReplacement[],
): boolean {
  const blocks = scanMermaid(markdown)
  const failed = new Set<number>()
  for (const f of failures) {
    if (f.index < 1 || f.index > blocks.length) return false
    failed.add(f.index)
  }
  const wanted = new Map<number, string>()
  for (const r of replacements) {
    // Substituto apenas para bloco solicitado, sem índice duplicado.
    if (!failed.has(r.index) || wanted.has(r.index)) return false
    if (r.source.trim() === '') return false
    wanted.set(r.index, r.source)
  }
  // Todo bloco reprovado precisa ter sido substituído.
  for (const idx of failed) {
    if (!wanted.has(idx)) return false
  }
  const composed = replaceMermaidBlocks(markdown, replacements)
  if (composed == null) return false
  const after = extractMermaidBlocks(composed)
  if (after.length !== blocks.length) return false
  for (const b of after) {
    const expected = failed.has(b.index) ? wanted.get(b.index)! : blocks[b.index - 1].source
    if (b.source !== expected) return false
    try {
      render(b.source)
    } catch {
      return false
    }
  }
  return true
}
