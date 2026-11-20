import { useSyncExternalStore } from 'react'

// O tema do app (docs/FRONTEND.md, Tema). 'light' e 'dark' vão ao `colorScheme` do
// `Provider`; 'system' o omite. A escolha mora no `localStorage`, chave `theme`: o
// script do `index.html` a lê, de forma síncrona, antes da primeira pintura.
export type Theme = 'system' | 'light' | 'dark'

const KEY = 'theme'

/** O valor salvo; qualquer outra coisa (ou nada) é o Sistema. */
export const parseTheme = (v: string | null): Theme => (v === 'light' || v === 'dark' ? v : 'system')

const read = (): Theme => {
  try {
    return parseTheme(localStorage.getItem(KEY))
  } catch {
    return 'system'
  }
}

let current = read()
const listeners = new Set<() => void>()

export function setTheme(t: Theme) {
  current = t
  try {
    if (t === 'system') localStorage.removeItem(KEY)
    else localStorage.setItem(KEY, t)
  } catch {
    // sem localStorage o tema vale só até fechar
  }
  // O `Canvas` do <html> (pintado antes do React) acompanha a escolha.
  document.documentElement.style.colorScheme = t === 'system' ? 'light dark' : t
  listeners.forEach((l) => l())
}

export const useTheme = (): Theme =>
  useSyncExternalStore(
    (l) => {
      listeners.add(l)
      return () => listeners.delete(l)
    },
    () => current,
  )
