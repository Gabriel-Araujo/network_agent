import { useEffect, useState } from 'react'
import { Application, Window, goApp } from '../wails'

// Controles de janela para o modo frameless.
//
// Os glifos seguem o padrão do sistema operacional em vez de uma família de
// ícones genérica: no Windows são os caracteres da Segoe Fluent/MDL2 Assets
// (mesmos usados pelo Explorer), então o botão "maximizar" alterna para o
// glifo de "restaurar" quando a janela está maximizada, como o usuário espera.
// A coloração vem dos tokens do tema; só o hover de fechar usa o vermelho
// padrão do Windows.

// Segoe MDL2 Assets: E921 minimizar, E922 maximizar, E923 restaurar, E8BB fechar
const GLYPH = {
  min: '\uE921',
  max: '\uE922',
  restore: '\uE923',
  close: '\uE8BB',
}

export default function WindowControls() {
  const [maximised, setMaximised] = useState(false)

  // Mantém o glifo em sincronia com o estado real da janela (o usuário pode
  // maximizar por duplo clique na barra, snap do Windows ou Win+setas).
  useEffect(() => {
    if (!goApp()) return

    let alive = true
    const sync = async () => {
      try {
        const m = await Window.IsMaximised()
        if (alive) setMaximised(m)
      } catch {
        /* runtime indisponível (browser puro) */
      }
    }
    sync()
    const id = window.setInterval(sync, 400)
    return () => {
      alive = false
      clearInterval(id)
    }
  }, [])

  const minimise = () => Window.Minimise()
  const toggleMax = () => Window.ToggleMaximise()
  const close = () => Application.Quit()

  return (
    <div className="wincontrols">
      <button
        className="wincontrols__btn"
        onClick={minimise}
        aria-label="Minimizar"
        title="Minimizar"
      >
        <span className="wincontrols__glyph" aria-hidden="true">{GLYPH.min}</span>
      </button>
      <button
        className="wincontrols__btn"
        onClick={toggleMax}
        aria-label={maximised ? 'Restaurar' : 'Maximizar'}
        title={maximised ? 'Restaurar' : 'Maximizar'}
      >
        <span className="wincontrols__glyph" aria-hidden="true">
          {maximised ? GLYPH.restore : GLYPH.max}
        </span>
      </button>
      <button
        className="wincontrols__btn wincontrols__btn--close"
        onClick={close}
        aria-label="Fechar"
        title="Fechar"
      >
        <span className="wincontrols__glyph" aria-hidden="true">{GLYPH.close}</span>
      </button>
    </div>
  )
}