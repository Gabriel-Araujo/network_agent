import { Button } from 'react-aria-components'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Draw from '@react-spectrum/s2/icons/Draw'
import Edit from '@react-spectrum/s2/icons/Edit'
import Lightbulb from '@react-spectrum/s2/icons/Lightbulb'
import Tools from '@react-spectrum/s2/icons/Tools'

// A tela Início (docs/FRONTEND.md, Início, barra de status e toast): título,
// subtítulo e as quatro sugestões fixas, uma por capacidade do Newton. Clicar
// só preenche o composer (App.tsx): um envio do Builder substitui o frr.conf do
// Workspace, então um clique não pode escrever no disco. O composer é o
// `InputDock` compartilhado com a Conversa.

const icon = iconStyle({ size: 'XS', color: 'accent' })

const SUGGESTIONS = [
  { icon: <Tools styles={icon} />, label: 'Montar um eBGP entre o AS 65001 e o AS 65002' },
  { icon: <Edit styles={icon} />, label: 'Adicionar BFD ao BGP do frr.conf' },
  { icon: <Draw styles={icon} />, label: 'Desenhar um iBGP com route reflector' },
  { icon: <Lightbulb styles={icon} />, label: 'Explicar como o OSPF elege o DR' },
]

const home = style({
  flexGrow: 1,
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 16,
  paddingX: 32,
  paddingY: 40,
  minHeight: 0,
  overflowY: 'auto',
  backgroundColor: 'base',
})

const title = style({
  margin: 0,
  font: 'heading-2xl',
  color: 'neutral',
  textAlign: 'center',
  letterSpacing: '-0.02em',
})

const subtitle = style({
  margin: 0,
  font: 'ui',
  color: 'neutral-subdued',
  textAlign: 'center',
  maxWidth: 460,
})

const suggestions = style({
  display: 'grid',
  gridTemplateColumns: {
    default: ['1fr', '1fr'],
    '@media (max-width: 1080px)': ['1fr'],
  },
  gap: 12,
  width: 'full',
  maxWidth: 600,
  marginTop: 16,
})

const card = style({
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  padding: 16,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: { default: 'gray-300', isHovered: 'gray-400' },
  borderRadius: 'xl',
  backgroundColor: { default: 'layer-1', isHovered: 'layer-2' },
  boxShadow: { default: 'none', isHovered: 'elevated' },
  font: 'ui',
  color: { default: 'neutral-subdued', isHovered: 'neutral' },
  textAlign: 'start',
  cursor: 'default',
  outlineStyle: { default: 'none', isFocusVisible: 'solid' },
  outlineColor: 'focus-ring',
  outlineWidth: 2,
  outlineOffset: 2,
  transition: 'default',
  scale: { default: 1, isPressed: 0.98 },
})

interface HomeProps {
  /** Põe a sugestão no composer da Início, sem enviar. */
  onSuggestion: (text: string) => void
}

export default function Home({ onSuggestion }: HomeProps) {
  return (
    <section className={home} aria-label="Início">
      <h1 className={title}>O que vamos fazer hoje?</h1>
      <p className={subtitle}>
        Descreva a tarefa. O Newton monta, ajusta, desenha ou explica configurações FRR.
      </p>
      <div className={suggestions}>
        {SUGGESTIONS.map((s) => (
          <Button
            key={s.label}
            className={({ isHovered, isFocusVisible, isPressed }) =>
              card({ isHovered, isFocusVisible, isPressed })
            }
            onPress={() => onSuggestion(s.label)}
          >
            {s.icon}
            <span>{s.label}</span>
          </Button>
        ))}
      </div>
    </section>
  )
}
