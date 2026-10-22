import { describe, expect, it } from 'vitest'
import { many, percent } from '@numen/ui'
import { WORDS } from './words'

const samplePoint = {
  value: 30,
  reviews: 12,
  minutes: 20,
  horizon: 14,
  clears: null as number | null,
  short: 0,
  cards: 40,
}

describe('WORDS.buys', () => {
  it('says a pile that no projected day clears is not within the horizon', () => {
    const lines = WORDS.buys('minutes', { ...samplePoint, clears: -1 })
    expect(lines[2]).toBe(`not within ${many(14, 'day')}`)
  })

  it('says a pile cleared on day zero is gone in zero days', () => {
    const lines = WORDS.buys('minutes', { ...samplePoint, clears: 0 })
    expect(lines[2]).toBe(`overdue gone in ${many(0, 'day')}`)
  })

  it('says nothing about the overdue where nothing is overdue', () => {
    expect(WORDS.buys('retention', samplePoint)).toEqual([
      `${percent(30)} remembered`,
      `${many(20, 'minute')} a day`,
    ])
  })

  it('names the card count of a session under a goal of minutes', () => {
    expect(WORDS.buys('minutes', samplePoint)[1]).toBe(`${many(12, 'card')} a session`)
  })

  it('adds the cards no pace reaches under a goal of a date', () => {
    expect(WORDS.buys('date', { ...samplePoint, short: 3 })).toEqual([
      `${many(30, 'day')} off`,
      `${many(20, 'minute')} a day`,
      '3 of 40 cannot get there',
    ])
    expect(WORDS.buys('date', samplePoint)).toHaveLength(2)
  })
})

describe('WORDS axis units', () => {
  it('reads a height in cards under minutes and in minutes under the other goals', () => {
    expect(WORDS.formatGoalCost('minutes', 5)).toBe(many(5, 'card'))
    expect(WORDS.formatGoalCost('date', 5)).toBe('5 min')
  })

  it('reads a width as a share, a day count or minutes', () => {
    expect(WORDS.formatGoalValue('retention', 90)).toBe(percent(90))
    expect(WORDS.formatGoalValue('date', 7)).toBe('7 d')
    expect(WORDS.formatGoalValue('minutes', 7)).toBe('7 min')
  })

  it('names the mark under each goal', () => {
    expect(WORDS.markName('retention')).toBe('most kept')
    expect(WORDS.markName('date')).toBe('first day it fits')
    expect(WORDS.markName('minutes')).toBe('time enough')
  })

  it('reads the value in the units of its goal', () => {
    expect(WORDS.value('retention', 90, 'Mon')).toBe(`${percent(90)} remembered`)
    expect(WORDS.value('date', 3, 'Mon')).toBe(`${many(3, 'day')} to Mon`)
    expect(WORDS.value('minutes', 3, 'Mon')).toBe(`${many(3, 'minute')} a day`)
  })
})

describe('WORDS.noCards', () => {
  it('speaks of one deck in the singular and of several by their count', () => {
    expect(WORDS.noCards(1)).toBe(
      'One deck points here and it holds no cards, so this preset schedules nothing.',
    )
    expect(WORDS.noCards(3)).toBe(
      '3 decks point here and they hold no cards, so this preset schedules nothing.',
    )
  })
})
