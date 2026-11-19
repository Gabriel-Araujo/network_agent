// As Abas (docs/FRONTEND.md, Abas e Conversas): a Início é a chave `HOME`; as
// outras são o id da Conversa.

export const HOME = 'home'

/**
 * Fecha a aba `id`. Se ela era a focada, o foco vai para a vizinha da direita,
 * ou da esquerda, e a Início é o piso.
 */
export function closeTab<T extends { id: string }>(
  s: { tabs: T[]; focused: string },
  id: string,
): { tabs: T[]; focused: string } {
  const i = s.tabs.findIndex((t) => t.id === id)
  if (i < 0) return s
  const tabs = s.tabs.filter((t) => t.id !== id)
  const focused = s.focused === id ? ((tabs[i] ?? tabs[i - 1])?.id ?? HOME) : s.focused
  return { tabs, focused }
}
