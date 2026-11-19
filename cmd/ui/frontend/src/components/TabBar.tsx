import { type MouseEvent, type PointerEvent, useCallback, useEffect, useLayoutEffect, useRef } from 'react'
import { Tab, TabList } from 'react-aria-components'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import Home from '@react-spectrum/s2/icons/Home'
import Close from '@react-spectrum/s2/icons/Close'
import { HOME } from '../tabs'

// A barra de Abas (docs/FRONTEND.md, Abas e Conversas): a Início fixa e uma aba
// por Conversa aberta. Fica dentro do `Tabs` do App, que é dono da seleção e dos
// painéis. O indicador é um só e desliza entre as abas como o do S2 `Tabs`.
//
// O `SelectionIndicator` do RAC não acompanhava uma aba criada e selecionada no
// mesmo commit (a barra ficava na aba anterior); aqui o indicador mede a aba
// selecionada e se posiciona com uma transição de `translate`/`width`.

export interface TabInfo {
  id: string
  title: string
  /** A Conversa está no turno: mostra o `dot`. */
  busy: boolean
}

const bar = style({ position: 'relative', flexShrink: 0 })

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

// Sobrepõe a borda inferior da barra e desliza com a aba focada (0,2 s, como o S2).
const indicator = style({
  position: 'absolute',
  left: 0,
  bottom: 0,
  height: 2,
  width: 0,
  borderTopRadius: 'sm',
  backgroundColor: 'accent',
  pointerEvents: 'none',
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
  /** A aba focada (`HOME` ou o id da Conversa). */
  selected: string
  onClose: (id: string) => void
}

export default function TabBar({ tabs, selected, onClose }: TabBarProps) {
  const barRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const indicatorRef = useRef<HTMLSpanElement>(null)
  const firstPlacement = useRef(true)
  // A aba alvo vem do estado, não de `aria-selected`: na criação da aba o DOM
  // do RAC ainda pode estar com a seleção anterior quando o efeito roda.
  const selectedRef = useRef(selected)
  selectedRef.current = selected

  // Posiciona o indicador sob a aba selecionada. A transição CSS cuida do
  // deslize; o `translate` é em relação à barra, que é o pai posicionado.
  const place = useCallback(() => {
    const bar = barRef.current
    const list = listRef.current
    const ind = indicatorRef.current
    if (!bar || !list || !ind) return
    const tab = list.querySelector<HTMLElement>(`[role="tab"][data-tab="${CSS.escape(selectedRef.current)}"]`)
    if (!tab) return
    const b = bar.getBoundingClientRect()
    const r = tab.getBoundingClientRect()
    ind.style.translate = `${r.left - b.left + 8}px 0`
    ind.style.width = `${Math.max(0, r.width - 16)}px`
  }, [])

  useLayoutEffect(() => {
    const ind = indicatorRef.current
    if (!ind) return
    if (firstPlacement.current) {
      firstPlacement.current = false
      ind.style.transitionProperty = 'none'
    }
    place()
    if (ind.style.transitionProperty === 'none') {
      requestAnimationFrame(() => {
        ind.style.transitionProperty = ''
      })
    }
  }, [place, selected, tabs])

  useEffect(() => {
    const list = listRef.current
    if (!list) return
    const resize = new ResizeObserver(place)
    resize.observe(list)
    // A aba nova entra no DOM da coleção depois do render que mudou `selected`;
    // o MutationObserver posiciona o indicador quando ela aparece.
    const mutate = new MutationObserver(place)
    mutate.observe(list, {
      childList: true,
      subtree: true,
      characterData: true,
      attributes: true,
      attributeFilter: ['aria-selected', 'data-tab'],
    })
    list.addEventListener('scroll', place)
    return () => {
      resize.disconnect()
      mutate.disconnect()
      list.removeEventListener('scroll', place)
    }
  }, [place])

  return (
    <div ref={barRef} className={bar}>
      <TabList ref={listRef} aria-label="Abas" className={list}>
        <Tab
          id={HOME}
          data-tab={HOME}
          className={({ isSelected, isHovered, isFocusVisible, isPressed }) =>
            tab({ isSelected, isHovered, isFocusVisible, isPressed })}
        >
          <Home styles={icon} />
          <span className={label}>Início</span>
        </Tab>
        {tabs.map((t) => (
          <Tab
            key={t.id}
            id={t.id}
            data-tab={t.id}
            className={({ isSelected, isHovered, isFocusVisible, isPressed }) =>
              tab({ isSelected, isHovered, isFocusVisible, isPressed })}
          >
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
      <span ref={indicatorRef} className={indicator} aria-hidden="true" />
    </div>
  )
}
