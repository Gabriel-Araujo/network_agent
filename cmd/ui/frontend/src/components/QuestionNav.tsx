import { Message } from '../types'

interface QuestionNavProps {
  userMessages: Message[]
  onNavigate: (id: string) => void
}

// Lista de pequenas barras horizontais (uma por pergunta do usuário),
// ao lado da scrollbar. Aparece a partir do segundo input do usuário.
// Hover revela um balão com os inputs; clicar leva à mensagem correspondente.
export default function QuestionNav({ userMessages, onNavigate }: QuestionNavProps) {
  if (userMessages.length < 2) return null

  return (
    <div className="question-nav" aria-label="Perguntas do usuário">
      <div className="question-nav__bars">
        {userMessages.map((m, i) => (
          <button
            key={m.id}
            className="question-nav__bar"
            onClick={() => onNavigate(m.id)}
            aria-label={`Ir para pergunta ${i + 1}`}
            title={`Pergunta ${i + 1}`}
          />
        ))}
      </div>

      <div className="question-nav__popover" role="tooltip">
        <p className="question-nav__popover-title">Perguntas</p>
        {userMessages.map((m, i) => (
          <button
            key={m.id}
            className="question-nav__item"
            onClick={() => onNavigate(m.id)}
          >
            <span className="question-nav__item-num">{i + 1}</span>
            <span className="question-nav__item-text">{m.content}</span>
          </button>
        ))}
      </div>
    </div>
  )
}