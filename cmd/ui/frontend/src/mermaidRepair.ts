import type { Conversation, MermaidFailure, MermaidRepair, MermaidRepairStart, MermaidReplacement } from './types'
import { parseBridgeError } from './types'
import {
  detectMermaidFailures,
  extractMermaidBlocks,
  replaceMermaidBlocks,
  validateMermaidRepair,
} from './mermaidBlocks'

// A ponte Wails do reparo: início, nova rodada dentro da MESMA tentativa
// (levando o feedback do renderizador de volta ao modelo) e conclusão
// (aceite decidido aqui, texto recomposto no backend). Injetável para os
// testes usarem ponte simulada. Cada chamada devolve o conjunto ACUMULADO de
// substituições da tentativa — o backend é quem funde rodada a rodada.
export interface MermaidRepairBridge {
  startRepair(messageID: string, failures: MermaidFailure[]): Promise<MermaidRepairStart>
  retryRepair(attemptID: string, failures: MermaidFailure[]): Promise<MermaidRepairStart>
  finishRepair(attemptID: string, accepted: boolean): Promise<Conversation>
}

// O orçamento de rodadas de modelo dentro de UMA tentativa: a primeira mais as
// voltas que levam o erro do renderizador de volta ao modelo. Medido em
// 2026-11-15: uma chamada conserta uma classe de defeito por vez (a aresta,
// depois o título do subgraph) — o conserto precisa do laço limitado, não de
// uma chamada feliz. É o mesmo número do backend (maxRepairRounds).
const MAX_REPAIR_ROUNDS = 3

/**
 * UMA tentativa automática de reparo para uma resposta recém-concluída cujos
 * blocos o pele rejeita. Chamada só depois do retorno do AskAgent — nunca
 * durante o streaming, ao reabrir histórico, nem para resposta de outro reparo
 * (o fluxo não cria resposta nova). Sem efeito colateral no render React.
 *
 * Resposta válida não faz nenhuma chamada. Com falhas, agrega todos os blocos
 * numa única tentativa e a leva até o fim em até MAX_REPAIR_ROUNDS rodadas de
 * modelo: cada rodada devolve candidatos, o texto é revalidado com o pele, e o
 * que ainda falha volta ao modelo como feedback. Aceite só quando todos os
 * blocos renderizam; esgotadas as rodadas, ou na falha de uma chamada, erro e
 * fonte originais ficam na tela. Devolve a conversa confirmada pelo backend (o
 * texto corrigido só existe depois dessa confirmação), ou null quando nada
 * mudou.
 */
export async function runMermaidRepair(
  conversationID: string,
  messageID: string,
  content: string,
  bridge: MermaidRepairBridge,
  onState: (repair: MermaidRepair) => void,
): Promise<Conversation | null> {
  void conversationID
  const firstFailures = detectMermaidFailures(extractMermaidBlocks(content))
  if (firstFailures.length === 0) return null
  const base: MermaidRepair = { attemptId: '', messageId: messageID, state: 'repairing' }
  onState(base)
  let attemptID = ''
  let failures = firstFailures
  let replacements: MermaidReplacement[] = []
  let accepted = false
  try {
    for (let round = 0; round < MAX_REPAIR_ROUNDS; round++) {
      const start = round === 0
        ? await bridge.startRepair(messageID, failures)
        : await bridge.retryRepair(attemptID, failures)
      if (round === 0) attemptID = start.attemptId
      replacements = start.replacements ?? []
      const composed = replaceMermaidBlocks(content, replacements)
      if (composed == null) break // a substituição mudaria a estrutura: recusa
      const remaining = detectMermaidFailures(extractMermaidBlocks(composed))
      if (remaining.length === 0) {
        accepted = validateMermaidRepair(content, firstFailures, replacements)
        break
      }
      failures = remaining
    }
  } catch (e) {
    onState({ ...base, state: 'failed', error: parseBridgeError(e).why })
    return null
  }
  try {
    return await bridge.finishRepair(attemptID, accepted)
  } catch (e) {
    onState({ ...base, state: 'failed', error: parseBridgeError(e).why })
    return null
  }
}
