import { describe, expect, it } from 'vitest'
import { parseTheme } from './theme'

describe('parseTheme', () => {
  it('claro e escuro valem como estão', () => {
    expect(parseTheme('light')).toBe('light')
    expect(parseTheme('dark')).toBe('dark')
  })
  it('ausente ou desconhecido é o Sistema', () => {
    expect(parseTheme(null)).toBe('system')
    expect(parseTheme('system')).toBe('system')
    expect(parseTheme('sepia')).toBe('system')
  })
})
