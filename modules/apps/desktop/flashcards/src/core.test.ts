import { describe, expect, it } from 'vitest'

import { Rating } from '@numen/protocol'

import { ahead, called, deckName, grades, rated } from './core'

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
    expect(grades).toEqual(['again', 'hard', 'good', 'easy'])
  })

  // What is written down is what the person pressed. Nothing downstream can
  // tell one rating from another, so a pair crossed here is a card sent away
  // for a week because a person said they had forgotten it.
  it('are written down as the four they name', () => {
    expect(rated.again).toBe(Rating.AGAIN)
    expect(rated.hard).toBe(Rating.HARD)
    expect(rated.good).toBe(Rating.GOOD)
    expect(rated.easy).toBe(Rating.EASY)
  })

  it('each have a word to press', () => {
    expect(called.again).toBe('Again')
    expect(called.hard).toBe('Hard')
    expect(called.good).toBe('Good')
    expect(called.easy).toBe('Easy')
  })
})
