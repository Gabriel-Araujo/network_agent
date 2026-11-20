import { useEffect, useRef } from 'react'
import { Message } from '../types'
import MessageBubble from './MessageBubble'
import QuestionNav from './QuestionNav'

// A Conversa (docs/FRONTEND.md, Abas e Conversas): os balões do transcript e o
// trilho de Perguntas. O estado vazio saiu daqui para o `Home`, que é a aba
// Início; a Conversa nasce no primeiro envio.

interface ChatAreaProps {
  messages: Message[]
  isWorking: boolean
}

export default function ChatArea({ messages, isWorking }: ChatAreaProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [messages, isWorking])

  // rola até uma mensagem específica (usado pelo navegador de perguntas)
  const navigateTo = (id: string) => {
    const el = scrollRef.current?.querySelector(`[data-msg-id="${id}"]`)
    el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }

  const userMessages = messages.filter((m) => m.role === 'user')

  return (
    <section className="chat" aria-label="Conversa">
      <div className="chat__scroll" ref={scrollRef}>
        <div className="chat__col">
          {messages.map((m) => (
            <MessageBubble key={m.id} message={m} />
          ))}
          <div ref={bottomRef} />
        </div>
      </div>

      <QuestionNav userMessages={userMessages} onNavigate={navigateTo} />
    </section>
  )
}
