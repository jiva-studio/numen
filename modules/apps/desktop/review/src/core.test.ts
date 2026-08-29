import { describe, expect, it } from 'vitest'

import { ahead, called, deckName, rated, said } from './core'

describe('how long a card is away for', () => {
  it('is read at the coarsest a person reads it by', () => {
    expect(ahead(30)).toBe('1m')
    expect(ahead(11 * 60)).toBe('11m')
    expect(ahead(90 * 60)).toBe('2h')
    expect(ahead(3 * 24 * 3600)).toBe('3d')
    expect(ahead(60 * 24 * 3600)).toBe('2mo')
    expect(ahead(3 * 365 * 24 * 3600)).toBe('3y')
  })

  // A card coming round in seconds is a card coming round now, and there is no
  // smaller word to say it in.
  it('is never less than a minute', () => {
    expect(ahead(0)).toBe('1m')
    expect(ahead(5)).toBe('1m')
  })
})

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

describe('the four answers', () => {
  it('are asked in the order they get harder to say', () => {
    expect(said).toEqual(['again', 'hard', 'good', 'easy'])
  })

  it('each have a word to press and a rating to write down', () => {
    for (const how of said) {
      expect(called[how]).toBeTruthy()
      expect(rated[how]).toBeGreaterThan(0)
    }
  })
})
