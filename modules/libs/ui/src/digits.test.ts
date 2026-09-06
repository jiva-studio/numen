import { describe, expect, it } from 'vitest'
import { grouped, many, percent, plural } from './digits'

describe('a whole number', () => {
  it('is grouped in thousands', () => {
    expect(grouped(1)).toBe('1')
    expect(grouped(1000)).toBe('1 000')
    expect(grouped(123456)).toBe('123 456')
  })
})

describe('a share', () => {
  it('is read as a percentage', () => {
    expect(percent(0)).toBe('0%')
    expect(percent(0.5)).toBe('50%')
    expect(percent(1)).toBe('100%')
  })

  it('is rounded to the whole per cent', () => {
    expect(percent(0.3749)).toBe('37%')
    expect(percent(0.375)).toBe('38%')
  })
})

describe('the thing a count counts', () => {
  it('is singular where there is one of it', () => {
    expect(plural(1, 'deck')).toBe('deck')
    expect(plural(0, 'deck')).toBe('decks')
    expect(plural(2, 'card')).toBe('cards')
  })

  it('rounds a fraction before it is made plural', () => {
    expect(plural(1.4, 'card')).toBe('card')
    expect(plural(1.5, 'card')).toBe('cards')
  })

  // An `s` on the end reaches neither an irregular noun nor a phrase whose
  // plural falls inside it, so the whole plural can be handed in.
  it('takes the whole plural where an s does not give it', () => {
    expect(plural(1, 'child', 'children')).toBe('child')
    expect(plural(2, 'child', 'children')).toBe('children')
    expect(plural(3, 'day to learn it', 'days to learn it')).toBe('days to learn it')
  })
})

describe('a count and the thing it counts', () => {
  it('is singular where there is one of it', () => {
    expect(many(1, 'card')).toBe('1 card')
    expect(many(0, 'card')).toBe('0 cards')
    expect(many(2, 'day')).toBe('2 days')
  })

  // A curve answers in fractions, and the two windows read one number to a
  // person: the count is rounded before it is read and before it is made
  // plural, so nothing anywhere says "1.4 cards".
  it('rounds a fraction before it is read out', () => {
    expect(many(1.4, 'card')).toBe('1 card')
    expect(many(1.5, 'card')).toBe('2 cards')
    expect(many(0.6, 'minute')).toBe('1 minute')
  })
})
