export {}

// Declaração à mão da superfície exposta ao front-end. A fonte da verdade é o
// binding gerado (wailsjs/go/main/App.d.ts) — isto acompanha o que ele expõe.
import type { Connection, Conversation, MermaidFailure, MermaidRepairStart } from './types'

declare global {
  interface Window {
    go?: {
      main: {
        App: {
          // Histórico de conversas (ticket 07). ListChats (mock) morreu.
          ListConversations(): Promise<Conversation[]>
          CreateConversation(title: string): Promise<Conversation>
          RenameConversation(id: string, title: string): Promise<Conversation>
          DeleteConversation(id: string): Promise<void>
          AskAgent(convID: string, prompt: string): Promise<Conversation>
          // Reparo automático de diagrama: início (chamada única ao modelo,
          // sem tools) e conclusão (aceite decidido pelo front, texto
          // recomposto no backend a partir da resposta persistida).
          StartMermaidRepair(
            convID: string,
            messageID: string,
            failures: MermaidFailure[],
          ): Promise<MermaidRepairStart>
          RetryMermaidRepair(
            convID: string,
            attemptID: string,
            failures: MermaidFailure[],
          ): Promise<MermaidRepairStart>
          FinishMermaidRepair(convID: string, attemptID: string, accepted: boolean): Promise<Conversation>
          // Gaveta de Logs (ticket 12): as linhas do sink de console desta
          // execução, e o arquivo onde elas também estão.
          GetLogLines(): Promise<string[]>
          GetLogPath(): Promise<string>
          // Conexão com o provedor (ticket 13): ela é background, então o
          // front pergunta o estado e repete enquanto for `connecting`.
          GetConnection(): Promise<Connection>
          Reconnect(): Promise<Connection>
        }
      }
    }
    runtime?: {
      EventsOn(eventName: string, cb: (...args: unknown[]) => void): void
      EventsOff(...eventNames: string[]): void
      ClipboardSetText(text: string): Promise<boolean>
    }
  }
}
