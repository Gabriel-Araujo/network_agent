import { FormEvent, forwardRef, useImperativeHandle, useRef, useState } from 'react'
import { TextArea } from 'react-aria-components'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import ArrowUpSend from '@react-spectrum/s2/icons/ArrowUpSend'

// O composer da Início e da Conversa (docs/FRONTEND.md, Componentes): superfície
// própria em RAC `TextArea` + botão de enviar próprio, com os tokens do S2. O
// anexo do protótipo não entra: anexo não existe no turno.

export interface InputDockHandle {
  /** Sugestão da Início: põe o texto no lugar do rascunho, foca e deixa o cursor no fim. */
  fill: (text: string) => void
  focus: () => void
}

interface InputDockProps {
  onSend: (text: string) => void
  isWorking: boolean
  /** O turno roda em outra Aba: o título dela, para a linha do envio bloqueado. */
  busyIn?: string
  /** false = sem provedor/modelo configurado: entrada bloqueada. */
  configReady?: boolean
  /** chamado quando o usuário tenta enviar com a config incompleta */
  onBlocked?: () => void
}

const dock = style({
  paddingX: 24,
  paddingTop: 8,
  paddingBottom: 20,
})

const composer = style({
  display: 'flex',
  alignItems: 'end',
  gap: 8,
  width: 'full',
  maxWidth: 720,
  marginX: 'auto',
  paddingTop: 8,
  paddingBottom: 8,
  paddingStart: 12,
  paddingEnd: 8,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: { default: 'gray-300', ':focus-within': 'gray-900' },
  borderRadius: 'xl',
  backgroundColor: 'gray-25',
  transition: 'default',
})

const blocked = style({ opacity: 0.55 })

const input = style({
  flexGrow: 1,
  minWidth: 0,
  minHeight: 44,
  maxHeight: 160,
  paddingY: 8,
  borderStyle: 'none',
  outlineStyle: 'none',
  backgroundColor: 'transparent',
  color: { default: 'neutral', ':disabled': 'disabled' },
  font: 'ui',
  resize: 'none',
  overflowX: 'hidden',
  overflowY: 'auto',
  '::placeholder': { color: 'gray-600' },
})

const send = style({
  display: 'grid',
  placeItems: 'center',
  flexShrink: 0,
  size: 34,
  borderRadius: 'full',
  borderStyle: 'none',
  backgroundColor: { default: 'accent', ':hover': 'accent-1000' },
  color: 'white',
  cursor: 'pointer',
  transition: 'default',
  scale: { default: 1, ':active': 0.94 },
  opacity: { default: 1, ':disabled': 0.38 },
})

const sendIcon = iconStyle({ size: 'XS', color: 'white' })

const note = style({
  marginX: 0,
  marginBottom: 0,
  marginTop: 8,
  textAlign: 'center',
  font: 'ui-sm',
  color: 'neutral-subdued',
})

const InputDock = forwardRef<InputDockHandle, InputDockProps>(function InputDock(
  { onSend, isWorking, busyIn, configReady = true, onBlocked },
  ref,
) {
  const [value, setValue] = useState('')
  const taRef = useRef<HTMLTextAreaElement>(null)

  const resize = () => {
    const ta = taRef.current
    if (!ta) return
    ta.style.height = 'auto'
    ta.style.height = Math.min(ta.scrollHeight, 160) + 'px'
  }

  const submit = (e?: FormEvent) => {
    e?.preventDefault()
    if (!configReady) {
      onBlocked?.()
      return
    }
    const text = value.trim()
    if (!text || isWorking) return
    onSend(text)
    setValue('')
    const ta = taRef.current
    if (ta) {
      ta.style.height = 'auto'
      ta.focus()
    }
  }

  useImperativeHandle(ref, () => ({
    fill: (text: string) => {
      setValue(text)
      const ta = taRef.current
      if (!ta) return
      ta.focus()
      // O valor só está no DOM depois do render; aí o cursor vai para o fim.
      requestAnimationFrame(() => {
        ta.setSelectionRange(text.length, text.length)
        resize()
      })
    },
    focus: () => taRef.current?.focus(),
  }))

  return (
    <footer className={dock}>
      <form className={`${composer} ${!configReady ? blocked : ''}`} onSubmit={submit}>
        <TextArea
          ref={taRef}
          className={input}
          rows={1}
          placeholder={
            configReady ? 'Pergunte ou descreva uma tarefa...' : 'Configure um provedor e um modelo para começar'
          }
          disabled={!configReady}
          value={value}
          onChange={(e) => {
            setValue(e.target.value)
            resize()
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              submit()
            }
          }}
          aria-label="Mensagem para o agente"
        />
        <button
          type="submit"
          className={send}
          disabled={!configReady || !value.trim() || isWorking}
          aria-label="Enviar mensagem"
        >
          <ArrowUpSend styles={sendIcon} />
        </button>
      </form>
      {!configReady && (
        <p className={note} role="status">
          Nenhum provedor/modelo configurado. Abra Configurações para cadastrar.
        </p>
      )}
      {configReady && busyIn !== undefined && (
        <p className={note} role="status">
          Newton está respondendo em <em>{busyIn}</em>
        </p>
      )}
    </footer>
  )
})

export default InputDock
