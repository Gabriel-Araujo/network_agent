import { useEffect, useRef } from 'react'
import { style } from '@react-spectrum/s2/style' with { type: 'macro' }
import { Message } from '../types'
import MessageBubble from './MessageBubble'
import QuestionNav from './QuestionNav'

// A Conversa (docs/FRONTEND.md, Abas e Conversas): os balões do transcript e o
// trilho de Perguntas. O estado vazio saiu daqui para o `Home`, que é a aba
// Início; a Conversa nasce no primeiro envio.

const chat = style({
  position: 'relative',
  display: 'flex',
  flexDirection: 'column',
  minHeight: 0,
  overflow: 'hidden',
})
// A div que contém as bolhas é quem rola.
const scroll = style({ flexGrow: 1, minHeight: 0, overflow: 'auto' })
const col = style({
  display: 'flex',
  flexDirection: 'column',
  gap: 20,
  width: 'full',
  // 720 px de texto mais 44 px de respiro de cada lado: o trilho de Perguntas fica no da direita.
  maxWidth: '[808px]',
  boxSizing: 'border-box',
  marginX: 'auto',
  paddingX: 44,
  paddingTop: 28,
  paddingBottom: 32,
})

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
    <section className={chat} aria-label="Conversa">
      <div className={scroll} ref={scrollRef}>
        <div className={col}>
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
