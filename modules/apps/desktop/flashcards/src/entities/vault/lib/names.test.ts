import { describe, expect, it } from 'vitest'

import { deckName } from './names'

describe('the name of a deck', () => {
  it('is the file, without the folders it stands in or its suffix', () => {
    expect(deckName('decks/Mammals.md')).toBe('Mammals')
    expect(deckName('Words.md')).toBe('Words')
    expect(deckName('a/b/c/Long name.md')).toBe('Long name')
  })

  it('is the path itself where there is nothing to take off it', () => {
    expect(deckName('Mammals')).toBe('Mammals')
    expect(deckName('')).toBe('')
  })
})
