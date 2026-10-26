/**
 * What counts as an hour of the day.
 *
 * The awkward ones are the ends: midnight is an hour, twenty-four is not, and
 */
import { describe, expect, it } from 'vitest'
import { isClockTime } from './clock'

describe('an hour of the day', () => {
  it('is two digits, a colon and two digits', () => {
    expect(isClockTime('04:00')).toBe(true)
    expect(isClockTime('23:59')).toBe(true)
    expect(isClockTime('00:00')).toBe(true)
  })

  it('is not an hour past the end of the day', () => {
    expect(isClockTime('24:00')).toBe(false)
    expect(isClockTime('12:60')).toBe(false)
  })

  it('is not written with the digits left off', () => {
    expect(isClockTime('4:00')).toBe(false)
    expect(isClockTime('04:0')).toBe(false)
  })

  it('is not seconds, and not words', () => {
    expect(isClockTime('04:00:00')).toBe(false)
    expect(isClockTime('four')).toBe(false)
  })

  it('is not nothing at all', () => {
    expect(isClockTime('')).toBe(false)
  })
})
