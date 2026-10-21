/**
 * Facts formatted as prose for preset sliders and tables.
 */
import { describe, expect, it } from 'vitest'
import {
  count,
  formatBuyFact,
  formatLearningFact,
  formatMaterialFact,
  sayAsSentence,
} from './formatFacts'

describe('sayAsSentence', () => {
  it('capitalizes the first letter and appends a period', () => {
    expect(sayAsSentence('an error occurred')).toBe('An error occurred.')
    expect(sayAsSentence('already capitalized')).toBe('Already capitalized.')
  })
})

describe('count', () => {
  it('rounds numbers to the nearest integer as string', () => {
    expect(count(3.2)).toBe('3')
    expect(count(3.8)).toBe('4')
    expect(count(0)).toBe('0')
  })
})

describe('formatBuyFact', () => {
  it('formats all buy fact kinds correctly', () => {
    expect(formatBuyFact({ kind: 'daysOff', days: 2 })).toBe('2 days off')
    expect(formatBuyFact({ kind: 'daysOff', days: 1 })).toBe('1 day off')
    expect(formatBuyFact({ kind: 'minutesADay', minutes: 15 })).toBe('15 minutes a day')
    expect(formatBuyFact({ kind: 'minutesADay', minutes: 1 })).toBe('1 minute a day')
    expect(formatBuyFact({ kind: 'remembered', share: 0.9 })).toBe('90% remembered')
    expect(formatBuyFact({ kind: 'cardsASession', cards: 20 })).toBe('20 cards a session')
    expect(formatBuyFact({ kind: 'cardsASession', cards: 1 })).toBe('1 card a session')
    expect(formatBuyFact({ kind: 'cannotGetThere', short: 5, cards: 100 })).toBe(
      '5 of 100 cannot get there',
    )
    expect(formatBuyFact({ kind: 'overdueGone', days: 3 })).toBe('overdue gone in 3 days')
    expect(formatBuyFact({ kind: 'overdueStays', horizon: 14 })).toBe('not within 14 days')
  })
})

describe('formatMaterialFact', () => {
  it('formats material facts with figures and pluralized names', () => {
    expect(formatMaterialFact({ kind: 'decks', count: 1 })).toStrictEqual({
      figure: '1',
      name: 'deck',
    })
    expect(formatMaterialFact({ kind: 'decks', count: 3 })).toStrictEqual({
      figure: '3',
      name: 'decks',
    })
    expect(formatMaterialFact({ kind: 'cards', count: 50 })).toStrictEqual({
      figure: '50',
      name: 'cards',
    })
    expect(formatMaterialFact({ kind: 'overdue', count: 12 })).toStrictEqual({
      figure: '12',
      name: 'overdue',
    })
    expect(formatMaterialFact({ kind: 'unbegun', count: 0 })).toStrictEqual({
      figure: '0',
      name: 'new',
    })
  })
})

describe('formatLearningFact', () => {
  it('formats learning facts with figures and names', () => {
    expect(formatLearningFact({ kind: 'notYet' })).toStrictEqual({
      figure: 'not yet',
      name: 'in the days ahead',
    })
    expect(formatLearningFact({ kind: 'allToday' })).toStrictEqual({
      figure: 'today',
      name: 'all of it learned',
    })
    expect(formatLearningFact({ kind: 'daysToLearn', days: 5 })).toStrictEqual({
      figure: '5',
      name: 'days to learn it',
    })
    expect(formatLearningFact({ kind: 'daysToLearn', days: 1 })).toStrictEqual({
      figure: '1',
      name: 'day to learn it',
    })
    expect(formatLearningFact({ kind: 'learnedToday', learned: 10, cards: 40 })).toStrictEqual({
      figure: '10 of 40',
      name: 'learned today',
    })
  })
})
