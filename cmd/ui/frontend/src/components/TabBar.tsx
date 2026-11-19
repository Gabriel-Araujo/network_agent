import type { PointerEvent, MouseEvent } from 'react'
import { SelectionIndicator, Tab, TabList } from 'react-aria-components'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Home from '@react-spectrum/s2/icons/Home'
import Close from '@react-spectrum/s2/icons/Close'
import { HOME } from '../tabs'

// A barra de Abas (docs/FRONTEND.md, Abas e Conversas): a Início fixa e uma aba
// por Conversa aberta. Fica dentro do `Tabs` do App, que é dono da seleção e dos
// painéis. O indicador desliza entre as abas como o do S2 `Tabs`.

export interface TabInfo {
  id: string
  title: string
  /** A Conversa está no turno: mostra o `dot`. */
  busy: boolean
}

const list = style({
  display: 'flex',
  alignItems: 'stretch',
  gap: 2,
  height: 46,
  flexShrink: 0,
  paddingX: 12,
  overflowX: 'auto',
  overflowY: 'hidden',
  scrollbarWidth: 'thin',
  borderBottomWidth: 1,
  borderStyle: 'solid',
  borderColor: 'gray-200',
  backgroundColor: 'base',
})

const tab = style({
  position: 'relative',
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  paddingX: 12,
  flexShrink: 0,
  maxWidth: 240,
  font: 'ui',
  fontWeight: { default: 'medium', isSelected: 'bold' },
  color: { default: 'neutral-subdued', isHovered: 'neutral', isSelected: 'neutral' },
  whiteSpace: 'nowrap',
  cursor: 'default',
  outlineStyle: { default: 'none', isFocusVisible: 'solid' },
  outlineColor: 'focus-ring',
  outlineWidth: 2,
  outlineOffset: -2,
  borderRadius: 'default',
  transition: 'default',
  scale: { default: 1, isPressed: 0.97 },
  '--iconPrimary': { type: 'color', value: '[currentColor]' },
})

const icon = iconStyle({ size: 'XS' })

const label = style({ minWidth: 0, truncate: true })

// Desliza de uma aba à outra pelo SharedElementTransition do RAC (0,2 s, como o S2).
const indicator = style({
  position: 'absolute',
  left: 8,
  right: 8,
  bottom: 0,
  height: 2,
  borderTopRadius: 'sm',
  backgroundColor: 'accent',
  transition: '[translate, width]',
  transitionDuration: 200,
  transitionTimingFunction: 'out',
})

const dot = style({ size: 6, borderRadius: 'full', backgroundColor: 'positive', flexShrink: 0 })

const close = style({
  display: 'grid',
  placeItems: 'center',
  size: 18,
  flexShrink: 0,
  borderRadius: 'sm',
  color: 'neutral-subdued',
  backgroundColor: { default: 'transparent', ':hover': 'gray-200' },
  '--iconPrimary': { type: 'color', value: '[currentColor]' },
})

// O "x" fica fora da ordem de Tab e fora da árvore de acessibilidade: pelo
// teclado, a aba fecha com Ctrl+W. Parar o pointerdown impede o usePress da aba
// de selecioná-la antes de fechar.
const stop = (e: PointerEvent | MouseEvent) => e.stopPropagation()

interface TabBarProps {
  tabs: TabInfo[]
  onClose: (id: string) => void
}

export default function TabBar({ tabs, onClose }: TabBarProps) {
  return (
    <TabList aria-label="Abas" className={list}>
      <Tab id={HOME} className={({ isSelected, isHovered, isFocusVisible, isPressed }) =>
        tab({ isSelected, isHovered, isFocusVisible, isPressed })}>
        <SelectionIndicator className={indicator} />
        <Home styles={icon} />
        <span className={label}>Início</span>
      </Tab>
      {tabs.map((t) => (
        <Tab
          key={t.id}
          id={t.id}
          className={({ isSelected, isHovered, isFocusVisible, isPressed }) =>
            tab({ isSelected, isHovered, isFocusVisible, isPressed })}
        >
          <SelectionIndicator className={indicator} />
          {t.busy && <span className={dot} aria-label="Newton está respondendo" role="img" />}
          <span className={label} title={t.title}>{t.title}</span>
          <span
            className={close}
            aria-hidden="true"
            onPointerDown={stop}
            onMouseDown={stop}
            onClick={(e) => {
              stop(e)
              onClose(t.id)
            }}
          >
            <Close styles={icon} />
          </span>
        </Tab>
      ))}
    </TabList>
  )
}
