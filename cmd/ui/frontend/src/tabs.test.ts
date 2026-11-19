import { describe, expect, it } from 'vitest'
import { HOME, closeTab } from './tabs'

const s = (focused: string, ...ids: string[]) => ({ tabs: ids.map((id) => ({ id })), focused })

describe('closeTab', () => {
  it('foca a vizinha da direita', () => {
    expect(closeTab(s('b', 'a', 'b', 'c'), 'b')).toEqual(s('c', 'a', 'c'))
  })
  it('sem vizinha à direita, foca a da esquerda', () => {
    expect(closeTab(s('c', 'a', 'b', 'c'), 'c')).toEqual(s('b', 'a', 'b'))
  })
  it('a Início é o piso', () => {
    expect(closeTab(s('a', 'a'), 'a')).toEqual(s(HOME))
  })
  it('fechar outra aba não muda o foco', () => {
    expect(closeTab(s('a', 'a', 'b'), 'b')).toEqual(s('a', 'a'))
  })
  it('aba que não está aberta não muda nada', () => {
    const st = s('a', 'a')
    expect(closeTab(st, 'x')).toBe(st)
  })
})
