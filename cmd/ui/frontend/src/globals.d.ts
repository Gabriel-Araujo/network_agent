export {}

declare global {
  interface Window {
    go?: {
      main: {
        App: {
          AskAgent(prompt: string): Promise<string>
          ListChats(): Promise<Array<{ id: string; title: string; group: string }>>
        }
      }
    }
    runtime?: {
      EventsOn(eventName: string, cb: (...args: unknown[]) => void): void
      EventsOff(...eventNames: string[]): void
    }
  }
}