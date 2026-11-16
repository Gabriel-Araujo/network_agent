import { describe, expect, it } from 'vitest'
import { runMermaidRepair, MermaidRepairBridge } from './mermaidRepair'
import type { Conversation, MermaidFailure, MermaidRepair, MermaidRepairStart } from './types'

// As mesmas fixtures medidas com o pele 1.13.0 (ver mermaidBlocks.test.ts):
// LINKSTYLE é a classe de produção (issue 29), SEMANTIC é modo sem gramática
// que o scanner do servidor não acusa (issue 16) — a classe do reparo do
// cliente.
const VALID = 'flowchart TD\n    A[origem] --> B[destino]'
const LINKSTYLE = 'flowchart TD\n    A --> B\n    linkStyle 0 stroke:#0A3;'
const LINKSTYLE_FIXED = 'flowchart TD\n    A --> B\n    linkStyle 0 stroke:#0A3'
const SEMANTIC =
  'sequenceDiagram\n    participant R1\n    R1->>R1: ping\n    activate R1\n    deactivate R1\n    deactivate R1'
const SEMANTIC_FIXED =
  'sequenceDiagram\n    participant R1\n    R1->>R1: ping\n    activate R1\n    deactivate R1'

const CONV: Conversation = {
  id: 'conv-1',
  title: 't',
  createdAt: '2026-11-15T00:00:00Z',
  updatedAt: '2026-11-15T00:00:00Z',
  provider: 'p',
  model: 'm',
  messages: [],
}

// O conteúdo da resposta é markdown: os blocos andam COM cercas, e o fonte que
// a ponte recebe é só o código.
const md = (block: string) => '```mermaid\n' + block + '\n```\n'

// A ponte simulada registra cada chamada e devolve o que o caso manda. O
// formato do erro espelha o bridgeError do Go (JSON dentro da string). O
// contrato de retryRepair devolve o conjunto ACUMULADO da tentativa, como o
// backend.
function fakeBridge(overrides: Partial<MermaidRepairBridge> = {}) {
  const calls: {
    start: { messageID: string; failures: MermaidFailure[] }[]
    retry: { attemptID: string; failures: MermaidFailure[] }[]
    finish: { attemptID: string; accepted: boolean }[]
  } = {
    start: [],
    retry: [],
    finish: [],
  }
  const fix = (failures: MermaidFailure[]) =>
    failures.map((f) => ({
      index: f.index,
      source: f.source === LINKSTYLE ? LINKSTYLE_FIXED : SEMANTIC_FIXED,
    }))
  const bridge: MermaidRepairBridge = {
    startRepair: async (messageID, failures) => {
      calls.start.push({ messageID, failures })
      return { attemptId: 'att-1', replacements: fix(failures) }
    },
    retryRepair: async (attemptID, failures) => {
      calls.retry.push({ attemptID, failures })
      return { attemptId: attemptID, replacements: fix(failures) }
    },
    finishRepair: async (attemptID, accepted) => {
      calls.finish.push({ attemptID, accepted })
      return CONV
    },
    ...overrides,
  }
  return { bridge, calls }
}

function stateRecorder() {
  const states: MermaidRepair[] = []
  return { states, onState: (r: MermaidRepair) => states.push(r) }
}

describe('runMermaidRepair', () => {
  it('resposta válida não faz nenhuma chamada', async () => {
    const { bridge, calls } = fakeBridge()
    const { states, onState } = stateRecorder()
    const updated = await runMermaidRepair('conv-1', 'msg-1', md(VALID), bridge, onState)
    expect(updated).toBeNull()
    expect(calls.start).toHaveLength(0)
    expect(calls.finish).toHaveLength(0)
    expect(states).toEqual([])
  })

  it('falha que o scanner não acusa dispara a tentativa com a mensagem do pele', async () => {
    const { bridge, calls } = fakeBridge()
    const { states, onState } = stateRecorder()
    const updated = await runMermaidRepair('conv-1', 'msg-1', md(SEMANTIC), bridge, onState)
    expect(calls.start).toHaveLength(1)
    expect(calls.start[0].messageID).toBe('msg-1')
    expect(calls.start[0].failures).toHaveLength(1)
    expect(calls.start[0].failures[0].error).toContain('inactivate')
    expect(calls.finish).toEqual([{ attemptID: 'att-1', accepted: true }])
    expect(updated).toBe(CONV)
    expect(states[0].state).toBe('repairing')
  })

  it('múltiplos blocos defeituosos viram UMA tentativa só', async () => {
    const { bridge, calls } = fakeBridge()
    const { onState } = stateRecorder()
    const content = `\`\`\`mermaid\n${LINKSTYLE}\n\`\`\`\n\nprosa\n\n\`\`\`mermaid\n${SEMANTIC}\n\`\`\`\n\n\`\`\`mermaid\n${VALID}\n\`\`\`\n`
    await runMermaidRepair('conv-1', 'msg-1', content, bridge, onState)
    expect(calls.start).toHaveLength(1)
    expect(calls.start[0].failures.map((f) => f.index)).toEqual([1, 2])
    expect(calls.finish).toEqual([{ attemptID: 'att-1', accepted: true }])
  })

  it('candidato que toca bloco não reprovado é recusado na conclusão', async () => {
    const { bridge, calls } = fakeBridge({
      startRepair: async (messageID, failures) => {
        calls.start.push({ messageID, failures })
        // renderiza tudo, mas substitui o bloco 1 (bom) — fora do pedido
        return {
          attemptId: 'att-1',
          replacements: [
            { index: 1, source: 'flowchart TD\n    A[outro] --> B[destino]' },
            { index: 2, source: SEMANTIC_FIXED },
          ],
        }
      },
    })
    const { onState } = stateRecorder()
    const content = md(VALID) + '\n' + md(SEMANTIC)
    await runMermaidRepair('conv-1', 'msg-1', content, bridge, onState)
    expect(calls.retry).toHaveLength(0) // tudo renderiza: nenhuma volta ao modelo
    expect(calls.finish).toEqual([{ attemptID: 'att-1', accepted: false }])
  })

  it('resposta obsoleta recusada pela ponte vira erro visível, sem conclusão', async () => {
    const stale = JSON.stringify({
      code: 'invalid',
      what: 'Não foi possível reparar o diagrama',
      why: 'a resposta não é mais a última resposta da conversa',
      next: 'Peça ao agente outra forma de representar o diagrama.',
    })
    const { bridge, calls } = fakeBridge({
      startRepair: async () => {
        throw new Error(stale)
      },
    })
    const { states, onState } = stateRecorder()
    const updated = await runMermaidRepair('conv-1', 'msg-1', md(SEMANTIC), bridge, onState)
    expect(updated).toBeNull()
    expect(calls.finish).toHaveLength(0)
    expect(states.map((s) => s.state)).toEqual(['repairing', 'failed'])
    expect(states[1].error).toBe('a resposta não é mais a última resposta da conversa')
  })

  it('falha na conclusão vira erro visível', async () => {
    const { bridge, calls } = fakeBridge({
      finishRepair: async () => {
        calls.finish.push({ attemptID: 'att-1', accepted: true })
        throw new Error('a tentativa foi interrompida ou pertence a outra conversa')
      },
    })
    const { states, onState } = stateRecorder()
    const updated = await runMermaidRepair('conv-1', 'msg-1', md(SEMANTIC), bridge, onState)
    expect(updated).toBeNull()
    expect(states.map((s) => s.state)).toEqual(['repairing', 'failed'])
    expect(states[1].error).toBe('a tentativa foi interrompida ou pertence a outra conversa')
  })

  it('converge por rodadas: o que ainda falha volta ao modelo como feedback', async () => {
    const { bridge, calls } = fakeBridge({
      startRepair: async (messageID, failures) => {
        calls.start.push({ messageID, failures })
        // rodada 1: o bloco 1 conserta, o 2 continua quebrado (uma classe por vez)
        return {
          attemptId: 'att-1',
          replacements: [
            { index: 1, source: LINKSTYLE_FIXED },
            { index: 2, source: SEMANTIC },
          ],
        }
      },
      retryRepair: async (attemptID, failures) => {
        calls.retry.push({ attemptID, failures })
        // rodada 2: conjunto ACUMULADO, com o bloco 2 consertado
        return {
          attemptId: attemptID,
          replacements: [
            { index: 1, source: LINKSTYLE_FIXED },
            { index: 2, source: SEMANTIC_FIXED },
          ],
        }
      },
    })
    const { onState } = stateRecorder()
    const content = md(LINKSTYLE) + '\n' + md(SEMANTIC)
    const updated = await runMermaidRepair('conv-1', 'msg-1', content, bridge, onState)
    expect(updated).toBe(CONV)
    expect(calls.start).toHaveLength(1)
    expect(calls.retry).toHaveLength(1)
    // o feedback da rodada 2 é o candidato ainda quebrado
    expect(calls.retry[0].failures.map((f) => f.index)).toEqual([2])
    expect(calls.retry[0].failures[0].source).toBe(SEMANTIC)
    expect(calls.finish).toEqual([{ attemptID: 'att-1', accepted: true }])
  })

  it('esgota as rodadas e conclui recusado', async () => {
    const { bridge, calls } = fakeBridge({
      startRepair: async (messageID, failures) => {
        calls.start.push({ messageID, failures })
        return { attemptId: 'att-1', replacements: failures.map((f) => ({ index: f.index, source: f.source })) }
      },
      retryRepair: async (attemptID, failures) => {
        calls.retry.push({ attemptID, failures })
        return { attemptId: attemptID, replacements: failures.map((f) => ({ index: f.index, source: f.source })) }
      },
    })
    const { onState } = stateRecorder()
    await runMermaidRepair('conv-1', 'msg-1', md(SEMANTIC), bridge, onState)
    expect(calls.start).toHaveLength(1)
    expect(calls.retry).toHaveLength(2) // 3 rodadas de modelo no total
    expect(calls.finish).toEqual([{ attemptID: 'att-1', accepted: false }])
  })

  it('falha numa rodada posterior vira erro visível', async () => {
    const { bridge, calls } = fakeBridge({
      startRepair: async (messageID, failures) => {
        calls.start.push({ messageID, failures })
        return { attemptId: 'att-1', replacements: [{ index: 1, source: SEMANTIC }] }
      },
      retryRepair: async () => {
        throw new Error('a tentativa esgotou as rodadas de correção')
      },
    })
    const { states, onState } = stateRecorder()
    const updated = await runMermaidRepair('conv-1', 'msg-1', md(SEMANTIC), bridge, onState)
    expect(updated).toBeNull()
    expect(calls.finish).toHaveLength(0)
    expect(states.map((s) => s.state)).toEqual(['repairing', 'failed'])
    expect(states[1].error).toBe('a tentativa esgotou as rodadas de correção')
  })
})
