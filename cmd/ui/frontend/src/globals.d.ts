export {}

// Declaração à mão da superfície exposta ao front-end. A fonte da verdade é o
// binding gerado (wailsjs/go/main/App.d.ts) — isto acompanha o que ele expõe.
import type { Conversation } from './types'

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
        }
      }
    }
    runtime?: {
      EventsOn(eventName: string, cb: (...args: unknown[]) => void): void
      EventsOff(...eventNames: string[]): void
    }
  }
}
