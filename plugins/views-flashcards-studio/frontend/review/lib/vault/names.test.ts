import { describe, expect, it } from 'vitest'

import { getDeckName } from './names'

describe('the name of a deck', () => {
  it('is the file, without the folders it remains in or its suffix', () => {
    expect(getDeckName('decks/Mammals.md')).toBe('Mammals')
    expect(getDeckName('Words.md')).toBe('Words')
    expect(getDeckName('a/b/c/Long name.md')).toBe('Long name')
  })

  it('is the path itself where there is nothing to take off it', () => {
    expect(getDeckName('Mammals')).toBe('Mammals')
    expect(getDeckName('')).toBe('')
  })

  it('keeps every dot in the name but the one before the suffix', () => {
    expect(getDeckName('decks/Vol. 2.md')).toBe('Vol. 2')
    expect(getDeckName('v1.2.md')).toBe('v1.2')
  })
})
