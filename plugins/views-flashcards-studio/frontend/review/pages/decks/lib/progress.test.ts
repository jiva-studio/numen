import { describe, expect, it } from 'vitest'

import { getSpentShare, hasNothingToBegin } from './progress'
import type { Preset } from '../types'
import type { DeckCardsDue } from '../../../entities/vault'

const deck = (fields: Partial<DeckCardsDue> = {}): DeckCardsDue => ({
  deck: 'decks/Words.md',
  faces: 20,
  due: 0,
  new: 0,
  learned: 0,
  unbegun: 20,
  ...fields,
})

const preset = (fields: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: null,
  decks: ['decks/Words.md'],
  named: 1,
  faces: 20,
  cards: 0,
  budget: { new: 0, reviews: 45, minutes: 20 },
  closes: { new: '', reviews: '', minutes: 'minutes_a_day' },
  answered: 0,
  answeredNew: 0,
  answeredReviews: 0,
  took: 0,
  paused: '',
  wrong: '',
  ...fields,
})

describe('whether a deck holds nothing that can ever be requested for', () => {
  it('holds nothing where every face is unbegun and its preset begins none a day', () => {
    expect(hasNothingToBegin(deck(), preset())).toBe(true)
  })

  it('holds something where its preset begins cards', () => {
    expect(
      hasNothingToBegin(deck(), preset({ budget: { new: 5, reviews: 45, minutes: 20 } })),
    ).toBe(false)
  })

  it('is no such deck where it holds no faces at all', () => {
    expect(hasNothingToBegin(deck({ faces: 0, unbegun: 0 }), preset())).toBe(false)
  })

  it('is no such deck where nothing says which preset schedules it', () => {
    expect(hasNothingToBegin(deck(), undefined)).toBe(false)
  })
})

describe('how far through its day a preset remains against a budget of nothing', () => {
  it('remains at nothing, however much was answered', () => {
    const one = preset({
      budget: { new: 0, reviews: 0, minutes: 0 },
      closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: 'minutes_a_day' },
      answered: 7,
      answeredNew: 3,
      answeredReviews: 4,
      took: 5,
    })

    expect(getSpentShare(one)).toBe(0)
  })

  it('remains at nothing where nothing was answered either', () => {
    const one = preset({
      budget: { new: 0, reviews: 0, minutes: 0 },
      closes: { new: 'new_a_day', reviews: 'reviews_a_day', minutes: 'minutes_a_day' },
    })

    expect(getSpentShare(one)).toBe(0)
  })
})
