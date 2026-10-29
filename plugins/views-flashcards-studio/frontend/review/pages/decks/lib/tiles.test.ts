import { describe, expect, it } from 'vitest'

import { getTiles } from './tiles'
import type { Preset } from '../types'

const preset = (fields: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: {
    goal: 'minutes',
    byDate: '',
    minutesADay: 20,
    newADay: 10,
    reviewsADay: 45,
    retention: 0.9,
    load: {},
    hasEvenLoad: true,
  },
  decks: ['decks/Words.md'],
  named: 1,
  reviewCards: 30,
  cards: 12,
  budget: { new: 10, reviews: 45, minutes: 20 },
  closes: { new: '', reviews: '', minutes: 'minutes_a_day' },
  reviewed: 0,
  reviewedNew: 0,
  reviewedReviews: 0,
  took: 0,
  paused: '',
  wrong: '',
  ...fields,
})

const today = '2026-09-05'

describe('which presets get a tile', () => {
  it('gives one to a preset whose decks name it and hold cards', () => {
    expect(getTiles([preset()], today)).toHaveLength(1)
  })

  it('gives none to a preset no deck names', () => {
    expect(getTiles([preset({ named: 0 })], today)).toHaveLength(0)
  })

  it('gives none to a preset whose decks hold no cards', () => {
    expect(getTiles([preset({ reviewCards: 0 })], today)).toHaveLength(0)
  })
})

describe('what a tile says', () => {
  it('names no goal for a preset read with no settings', () => {
    expect(getTiles([preset({ settings: null })], today)[0]?.goal).toBe('')
  })

  it('says nothing of what a session would ask where the preset schedules nothing', () => {
    const [tile] = getTiles([preset({ paused: 'no cards per day' })], today)

    expect(tile?.goal).toBe('no cards per day')
    expect(tile?.left).toBe('')
    expect(tile?.canOpen).toBe(false)
  })
})
