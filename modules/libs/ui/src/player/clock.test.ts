import { describe, expect, it } from 'vitest'
import { clock } from './clock'

describe('a moment written out', () => {
  it('is read as a clock, and carries an hour only where there is one', () => {
    expect(clock(0)).toBe('0:00')
    expect(clock(9_400)).toBe('0:09')
    expect(clock(125_000)).toBe('2:05')
    expect(clock(3_725_000)).toBe('1:02:05')
  })

  it('is the beginning for a moment before it', () => {
    expect(clock(-1)).toBe('0:00')
    expect(clock(Number.NEGATIVE_INFINITY)).toBe('0:00')
  })

  it('counts the minutes past the hour, not from the beginning', () => {
    expect(clock(3_600_000)).toBe('1:00:00')
    expect(clock(7_262_000)).toBe('2:01:02')
  })
})
