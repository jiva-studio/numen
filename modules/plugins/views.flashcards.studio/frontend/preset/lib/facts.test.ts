/**
 * Which facts a place on the curve and the material state.
 */
import { describe, expect, it } from 'vitest'
import { getBuyFacts, getLearningFacts, getMaterialFacts } from './facts'

const targetPoint = {
  value: 30,
  reviews: 12,
  minutes: 25,
  horizon: 14,
  clears: null,
  short: 0,
  cards: 400,
}

describe('what a place buys', () => {
  it('is the minutes and the session under a goal of minutes', () => {
    expect(getBuyFacts('minutes', targetPoint).map((one) => one.kind)).toStrictEqual([
      'minutesADay',
      'cardsASession',
    ])
  })

  it('is the share remembered and the minutes under a goal of retention', () => {
    expect(getBuyFacts('retention', targetPoint).map((one) => one.kind)).toStrictEqual([
      'remembered',
      'minutesADay',
    ])
  })

  it('adds when the overdue is gone, or that it stays, where something is overdue', () => {
    expect(getBuyFacts('minutes', { ...targetPoint, clears: 3 }).at(-1)).toStrictEqual({
      kind: 'overdueGone',
      days: 3,
    })
    expect(getBuyFacts('minutes', { ...targetPoint, clears: 0 }).at(-1)).toStrictEqual({
      kind: 'overdueGone',
      days: 0,
    })
    expect(getBuyFacts('minutes', { ...targetPoint, clears: -1 }).at(-1)).toStrictEqual({
      kind: 'overdueStays',
      horizon: 14,
    })
  })

  it('is the days off and the minutes under a goal of a date, and says nothing of the overdue', () => {
    expect(getBuyFacts('date', { ...targetPoint, clears: 3 }).map((one) => one.kind)).toStrictEqual([
      'daysOff',
      'minutesADay',
    ])
  })

  it('adds the cards no pace reaches under a goal of a date only where there are any', () => {
    expect(getBuyFacts('date', { ...targetPoint, short: 3 }).at(-1)).toStrictEqual({
      kind: 'cannotGetThere',
      short: 3,
      cards: 400,
    })
  })
})

describe('the facts of the material', () => {
  it('leave out the overdue and the new where there are none', () => {
    expect(
      getMaterialFacts({ decks: 4, cards: 160, overdue: 0, unbegun: 0 }).map((one) => one.kind),
    ).toStrictEqual(['decks', 'cards'])
    expect(
      getMaterialFacts({ decks: 4, cards: 160, overdue: 45, unbegun: 0 }).map((one) => one.kind),
    ).toStrictEqual(['decks', 'cards', 'overdue'])
    expect(
      getMaterialFacts({ decks: 4, cards: 160, overdue: 45, unbegun: 30 }).map((one) => one.kind),
    ).toStrictEqual(['decks', 'cards', 'overdue', 'unbegun'])
  })
})

describe('the facts of what is learned', () => {
  it('are only what remains learned today where no day is specified', () => {
    expect(getLearningFacts(undefined, 10, 40)).toStrictEqual([
      { kind: 'learnedToday', learned: 10, cards: 40 },
    ])
  })

  it('put the day first: not yet, today, or the days to learn', () => {
    expect(getLearningFacts(-1, 10, 40)[0]).toStrictEqual({ kind: 'notYet' })
    expect(getLearningFacts(0, 40, 40)[0]).toStrictEqual({ kind: 'allToday' })
    expect(getLearningFacts(9, 10, 40)[0]).toStrictEqual({ kind: 'daysToLearn', days: 9 })
    expect(getLearningFacts(9, 10, 40)).toHaveLength(2)
  })
})
