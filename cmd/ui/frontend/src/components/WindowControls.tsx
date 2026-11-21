import { useEffect, useState, type CSSProperties, type ReactNode } from 'react'
import { Application, Events, Window } from '@wailsio/runtime'
import { iconStyle, style } from '@react-spectrum/s2/style' with { type: 'macro' }
import MinimizarJanela from '../icons/MinimizarJanela'
import MaximizarJanela from '../icons/MaximizarJanela'
import Close from '@react-spectrum/s2/icons/Close'
import RestaurarJanela from '../icons/RestaurarJanela'
import { inWails } from '../types'

// Os botões da janela frameless (docs/FRONTEND.md, Shell e janela). Cada um é
// uma região `--wails-non-client-region` do modo de composição do Wails v3: o
// maximizar recebe `HTMAXBUTTON` e mostra o Snap Layouts no hover. O Wails
// repassa o mouse dessas regiões à página, então o clique e o :hover seguem no DOM.

const button = style({
  display: 'grid',
  placeItems: 'center',
  width: 34,
  height: 32,
  borderRadius: 'lg',
  borderStyle: 'none',
  backgroundColor: {
    default: 'transparent',
    ':hover': 'gray-100',
    isClose: { default: 'transparent', ':hover': 'negative' },
  },
  '--iconPrimary': {
    type: 'color',
    value: {
      default: 'neutral-subdued',
      ':hover': 'neutral',
      isClose: { default: 'neutral-subdued', ':hover': 'white' },
    },
  },
  transition: 'colors',
})

const region = (kind: string) => ({ '--wails-non-client-region': kind }) as CSSProperties

function WinButton({
  kind,
  label,
  onClick,
  children,
}: {
  kind: 'minimize' | 'maximize' | 'close'
  label: string
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      className={button({ isClose: kind === 'close' })}
      style={region(kind)}
      onClick={onClick}
      aria-label={label}
      title={label}
    >
      {children}
    </button>
  )
}

export default function WindowControls() {
  const [maximised, setMaximised] = useState(false)

  // O glifo acompanha o estado real da janela (snap, Win+setas, duplo clique
  // na caption): os eventos de janela do v3 avisam, e o IsMaximised confirma.
  useEffect(() => {
    if (!inWails()) return
    const sync = () => Window.IsMaximised().then(setMaximised).catch(() => {})
    sync()
    const offs = [
      Events.Types.Common.WindowMaximise,
      Events.Types.Common.WindowRestore,
      Events.Types.Common.WindowUnMaximise,
    ].map((name) => Events.On(name, sync))
    return () => offs.forEach((off) => off())
  }, [])

  const icon = iconStyle({ size: 'S' })
  return (
    <div className={style({ display: 'flex', gap: 2 })}>
      <WinButton kind="minimize" label="Minimizar" onClick={() => Window.Minimise()}>
        <MinimizarJanela styles={icon} />
      </WinButton>
      <WinButton
        kind="maximize"
        label={maximised ? 'Restaurar' : 'Maximizar'}
        onClick={() => Window.ToggleMaximise()}
      >
        {maximised ? <RestaurarJanela styles={icon} /> : <MaximizarJanela styles={icon} />}
      </WinButton>
      <WinButton kind="close" label="Fechar" onClick={() => Application.Quit()}>
        <Close styles={icon} />
      </WinButton>
    </div>
  )
}
