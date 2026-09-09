import { useEffect, useRef } from 'react'
import { ArrowClockwise, Command, Cpu, FileText, Sparkle } from '@phosphor-icons/react'
import { Message } from '../types'
import MessageBubble from './MessageBubble'
import QuestionNav from './QuestionNav'

interface ChatAreaProps {
  messages: Message[]
  onSuggestion: (text: string) => void
  onNewChat: () => void
  isWorking: boolean
}

const SUGGESTIONS = [
  { icon: <Sparkle size={16} weight="duotone" />, label: 'Resumir as conversas de hoje' },
  { icon: <Cpu size={16} weight="duotone" />, label: 'Explicar a stack do projeto' },
  { icon: <FileText size={16} weight="duotone" />, label: 'Escrever um README do zero' },
  { icon: <ArrowClockwise size={16} weight="duotone" />, label: 'Revisar meu último prompt' },
]

export default function ChatArea({ messages, onSuggestion, onNewChat, isWorking }: ChatAreaProps) {
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

  if (messages.length === 0) {
    return (
      <section className="chat chat--empty" aria-label="Nova conversa">
        <div className="empty">
          <div className="empty__mark" aria-hidden="true">
            <Command size={26} weight="duotone" />
          </div>
          <h1 className="empty__title">O que vamos fazer hoje?</h1>
          <p className="empty__hint">Descreva a tarefa. O agente monta um plano, executa e mostra cada passo.</p>
          <div className="empty__suggestions">
            {SUGGESTIONS.map((s) => (
              <button
                key={s.label}
                className="chip"
                onClick={() => onSuggestion(s.label)}
                disabled={isWorking}
              >
                <span className="chip__icon" aria-hidden="true">
                  {s.icon}
                </span>
                {s.label}
              </button>
            ))}
          </div>
          <button className="link-btn" onClick={onNewChat}>
            Limpar conversa
          </button>
        </div>
      </section>
    )
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