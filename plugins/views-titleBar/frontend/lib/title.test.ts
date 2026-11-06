import { describe, expect, it } from 'vitest'
import { formatTitle } from './title'

describe('formatTitle', () => {
  it('names the application and the vault', () => {
    expect(formatTitle('numen', 'Orchard notes')).toBe('numen — Orchard notes')
  })

  it('names only the application when no vault is open', () => {
    expect(formatTitle('numen', '')).toBe('numen')
  })

  it('treats a vault name of spaces as no vault', () => {
    expect(formatTitle('numen', '   ')).toBe('numen')
  })
})
