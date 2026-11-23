/** The fractions the motion is worked in. */
import { describe, expect, it } from 'vitest'
import { clampToUnit } from './math'

describe('clamp01', () => {
  it('keeps a fraction as it is', () => {
    expect(clampToUnit(0.4)).toBe(0.4)
  })

  it('holds a number outside the range to its nearest end', () => {
    expect(clampToUnit(-3)).toBe(0)
    expect(clampToUnit(7)).toBe(1)
  })

  it('reads not a number as all the way through', () => {
    expect(clampToUnit(Number.NaN)).toBe(1)
  })
})
