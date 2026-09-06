import { describe, expect, it } from 'vitest'
import { grouped, many, percent } from './digits'

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
