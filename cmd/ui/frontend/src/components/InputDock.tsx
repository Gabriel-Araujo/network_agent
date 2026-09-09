import { FormEvent, useRef, useState } from 'react'
import { ArrowUp, Paperclip } from '@phosphor-icons/react'
import { TokenUsage } from '../types'

interface InputDockProps {
  onSend: (text: string) => void
  isWorking: boolean
  usageSummary: TokenUsage
}

const PLACEHOLDERS = ['Pergunte ou descreva uma tarefa...', 'Ex.: verifique se há updates pendentes', 'Ex.: escreva um resumo da reunião']

const fmtNum = (n: number): string => n.toLocaleString('pt-BR')

export default function InputDock({ onSend, isWorking, usageSummary }: InputDockProps) {
  const [value, setValue] = useState('')
  const taRef = useRef<HTMLTextAreaElement>(null)

  const resize = () => {
    const ta = taRef.current
    if (!ta) return
    ta.style.height = 'auto'
    ta.style.height = Math.min(ta.scrollHeight, 180) + 'px'
  }

  const submit = (e?: FormEvent) => {
    e?.preventDefault()
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

  return (
    <footer className="dock">
      <form className="dock__form" onSubmit={submit}>
        <button type="button" className="icon-btn" aria-label="Anexar arquivo">
          <Paperclip size={18} weight="regular" aria-hidden="true" />
        </button>
        <textarea
          ref={taRef}
          className="dock__input"
          rows={1}
          placeholder={PLACEHOLDERS[0]}
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
          className="btn-send"
          disabled={!value.trim() || isWorking}
          aria-label="Enviar mensagem"
        >
          <ArrowUp size={18} weight="bold" aria-hidden="true" />
        </button>
      </form>
      <p className="dock__hint">
        <span>
          Total de tokens de entrada: <strong>{fmtNum(usageSummary.input)}</strong>
        </span>
        <span className="dock__hint-sep" aria-hidden="true">·</span>
        <span>
          Total de tokens de saída: <strong>{fmtNum(usageSummary.output)}</strong>
        </span>
        <span className="dock__hint-sep" aria-hidden="true">·</span>
        <span>
          Total de tokens: <strong>{fmtNum(usageSummary.total)}</strong>
        </span>
        <span className="dock__hint-sep" aria-hidden="true">·</span>
        <span>
          Contexto utilizado: <strong>{fmtNum(usageSummary.contextUsed)}</strong>/
          <strong>{fmtNum(usageSummary.contextTotal)}</strong>
        </span>
      </p>
    </footer>
  )
}