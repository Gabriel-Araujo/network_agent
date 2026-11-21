import { ActionButton, Tooltip, TooltipTrigger, ToastQueue } from '@react-spectrum/s2'
import { css, iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Copy from '@react-spectrum/s2/icons/Copy'
import { Clipboard } from '@wailsio/runtime'
import { inWails } from '../types'

// O bloco de código (docs/FRONTEND.md, Componentes): cabeçalho com a linguagem da
// fence e o copiar, e o código embaixo. Sem realce de sintaxe: o FRR não tem
// gramática numa lib que valha o peso.

// A JetBrains Mono só entra por aqui: o `fontFamily` do macro aceita só tokens, e
// o token `code` é fixo (docs/FRONTEND.md, Fonte). O tamanho e a altura vão pelo
// macro, sem `font`, para não disputar o `font-family` com esta classe.
export const mono = css(`font-family: 'JetBrains Mono', ui-monospace, Consolas, monospace;`)

const wrap = style({
  marginY: 12,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  borderRadius: 'xl',
  backgroundColor: 'gray-25',
  overflow: 'hidden',
})

const head = style({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  paddingStart: 16,
  paddingEnd: 8,
  paddingY: 4,
  borderWidth: 0,
  borderBottomWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-300',
  backgroundColor: 'layer-1',
  font: 'ui-sm',
  color: 'neutral-subdued',
})

const pre = style({
  margin: 0,
  padding: 16,
  overflowX: 'auto',
  fontSize: 'code-sm',
  lineHeight: 'code',
  color: 'neutral',
})

const grow = style({ flexGrow: 1 })
const icon = iconStyle({ size: 'S' })

// Copia pelo Clipboard do Wails; fora do app (browser puro) cai no do navegador.
export async function copyText(text: string) {
  if (inWails()) await Clipboard.SetText(text)
  else await navigator.clipboard.writeText(text)
}

export default function CodeBlock({ language, code }: { language: string; code: string }) {
  const copy = () =>
    copyText(code).then(
      () => ToastQueue.positive('Código copiado.', { timeout: 5000 }),
      () => ToastQueue.negative('Não foi possível copiar.', { timeout: 5000 }),
    )

  return (
    <div className={wrap}>
      <div className={head}>
        <span className={grow}>{language}</span>
        <TooltipTrigger>
          <ActionButton isQuiet size="S" aria-label="Copiar código" onPress={copy}>
            <Copy styles={icon} />
          </ActionButton>
          <Tooltip>Copiar</Tooltip>
        </TooltipTrigger>
      </div>
      <pre className={`${pre} ${mono}`}>
        <code>{code}</code>
      </pre>
    </div>
  )
}
