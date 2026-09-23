/**
 * What counts as an hour of the day.
 *
 * The awkward ones are the ends: midnight is an hour, twenty-four is not, and
 * a field standing at nothing stands at no hour.
 */
import { describe, expect, it } from 'vitest'
import { onTheClock } from './clock'

describe('an hour of the day', () => {
  it('is two digits, a colon and two digits', () => {
    expect(onTheClock('04:00')).toBe(true)
    expect(onTheClock('23:59')).toBe(true)
    expect(onTheClock('00:00')).toBe(true)
  })

  it('is not an hour past the end of the day', () => {
    expect(onTheClock('24:00')).toBe(false)
    expect(onTheClock('12:60')).toBe(false)
  })

  it('is not written with the digits left off', () => {
    expect(onTheClock('4:00')).toBe(false)
    expect(onTheClock('04:0')).toBe(false)
  })

  it('is not seconds, and not words', () => {
    expect(onTheClock('04:00:00')).toBe(false)
    expect(onTheClock('four')).toBe(false)
  })

  it('is not nothing at all', () => {
    expect(onTheClock('')).toBe(false)
  })
})
