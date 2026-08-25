/**
 * The arithmetic behind a cap's marks, as plain numbers.
 *
 * A mark drawn larger than the box has to be stroked thinner by the same
 * amount, or it arrives heavier than the marks beside it — which is the
 * raggedness the marks were brought in to end. The story measures the ink that
 * comes out; this measures the numbers that go in.
 */
import { describe, expect, it } from 'vitest'
import { MARKS } from './marks'
import type { PaletteMark } from './model'

const EVERY = Object.entries(MARKS) as readonly [PaletteMark, (typeof MARKS)[PaletteMark]][]

describe('how thick a mark is stroked', () => {
  it('is thinner by exactly as much as the mark is drawn larger', () => {
    const ink = EVERY.map(([, mark]) => mark.stroke * mark.fills)
    expect(new Set(ink.map((one) => one.toFixed(6))).size).toBe(1)
  })

  it('is a width the grid a mark is drawn on can carry', () => {
    for (const [name, mark] of EVERY) {
      expect(mark.stroke, name).toBeGreaterThan(0)
      expect(mark.stroke, name).toBeLessThan(24)
      expect(mark.fills, name).toBeGreaterThan(0)
    }
  })
})

describe('what a key held is called', () => {
  it('is a word each, and no two keys are called the same', () => {
    const said = EVERY.map(([, mark]) => mark.said)
    expect(said.every((word) => word !== '')).toBe(true)
    expect(new Set(said).size).toBe(said.length)
  })
})
