/** The fractions the motion is worked in. */
import { describe, expect, it } from 'vitest'
import { clamp01 } from './math'

describe('clamp01', () => {
  it('keeps a fraction as it is', () => {
    expect(clamp01(0.4)).toBe(0.4)
  })

  it('holds a number outside the range to its nearest end', () => {
    expect(clamp01(-3)).toBe(0)
    expect(clamp01(7)).toBe(1)
  })

  it('reads not a number as all the way through', () => {
    expect(clamp01(Number.NaN)).toBe(1)
  })
})
