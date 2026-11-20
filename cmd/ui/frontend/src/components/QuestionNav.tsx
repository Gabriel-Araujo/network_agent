import { useState } from 'react'
import { style } from '@react-spectrum/s2/style' with { type: 'macro' }
import { Message } from '../types'

interface QuestionNavProps {
  userMessages: Message[]
  onNavigate: (id: string) => void
}

// O trilho de Perguntas (docs/FRONTEND.md, Componentes): uma barra por pergunta
// do usuário na borda direita, a partir da segunda. Passar o mouse ou focar abre
// o painel com as perguntas; clicar leva à mensagem correspondente.

const rail = style({
  position: 'absolute',
  insetEnd: 20,
  top: '[45%]',
  translateY: '[-50%]',
  zIndex: 30,
  display: 'flex',
  alignItems: 'center',
})
const marks = style({
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'end',
  gap: 4,
  paddingX: 2,
  paddingY: 4,
  width: 28,
  flexShrink: 0,
})
const mark = style({
  width: { default: 18, ':hover': 24, ':focus-visible': 24 },
  height: 4,
  padding: 0,
  borderStyle: 'none',
  borderRadius: 'full',
  backgroundColor: { default: 'gray-400', ':hover': 'accent', ':focus-visible': 'accent' },
  outlineStyle: 'none',
  cursor: 'pointer',
  transition: 'default',
})
const panel = style({
  position: 'absolute',
  insetEnd: '[100%]',
  top: '[50%]',
  translateY: '[-50%]',
  translateX: { default: 4, isOpen: 0 },
  width: 260,
  maxHeight: '[60vh]',
  overflowY: 'auto',
  padding: 12,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  borderRadius: 'xl',
  backgroundColor: 'elevated',
  boxShadow: 'elevated',
  opacity: { default: 0, isOpen: 1 },
  visibility: { default: 'hidden', isOpen: 'visible' },
  transition: 'default',
})
const title = style({
  margin: 0,
  marginBottom: 8,
  font: 'detail-sm',
  textTransform: 'uppercase',
  color: 'neutral-subdued',
})
const item = style({
  display: 'flex',
  alignItems: 'start',
  gap: 8,
  width: 'full',
  padding: 0,
  paddingX: 8,
  paddingY: 4,
  borderStyle: 'none',
  borderRadius: 'default',
  backgroundColor: { default: 'transparent', ':hover': 'gray-100', ':focus-visible': 'gray-100' },
  color: { default: 'neutral-subdued', ':hover': 'neutral', ':focus-visible': 'neutral' },
  font: 'ui',
  textAlign: 'start',
  cursor: 'pointer',
  outlineStyle: 'none',
  transition: 'default',
})
const num = style({
  flexShrink: 0,
  display: 'grid',
  placeItems: 'center',
  size: 16,
  marginTop: 2,
  borderRadius: 'sm',
  backgroundColor: 'accent-subtle',
  color: 'accent',
  font: 'detail-sm',
})
const text = style({
  minWidth: 0,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
})

export default function QuestionNav({ userMessages, onNavigate }: QuestionNavProps) {
  const [open, setOpen] = useState(false)
  if (userMessages.length < 2) return null

  return (
    <div
      className={rail}
      aria-label="Perguntas do usuário"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget)) setOpen(false)
      }}
    >
      <div className={marks}>
        {userMessages.map((m, i) => (
          <button
            key={m.id}
            className={mark}
            onClick={() => onNavigate(m.id)}
            aria-label={`Ir para pergunta ${i + 1}`}
          />
        ))}
      </div>

      <div className={panel({ isOpen: open })} role="tooltip">
        <p className={title}>Perguntas</p>
        {userMessages.map((m, i) => (
          <button key={m.id} className={item} onClick={() => onNavigate(m.id)} tabIndex={open ? 0 : -1}>
            <span className={num}>{i + 1}</span>
            <span className={text}>{m.content}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
