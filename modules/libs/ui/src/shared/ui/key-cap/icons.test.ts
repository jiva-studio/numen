/**
 * The arithmetic behind a cap's icons, as plain numbers.
 *
 * An icon drawn larger than the box has to be stroked thinner by the same
 * amount, or it arrives heavier than the icons beside it — which is the
 * raggedness the icons were brought in to end. The story measures the ink that
 * comes out; this measures the numbers that go in.
 */
import { describe, expect, it } from 'vitest'
import { ICONS } from './icons'
import type { PaletteIcon } from './keys'

const EVERY = Object.entries(ICONS) as readonly [PaletteIcon, (typeof ICONS)[PaletteIcon]][]

describe('how thick an icon is stroked', () => {
  it('is thinner by exactly as much as the icon is drawn larger', () => {
    const ink = EVERY.map(([, icon]) => icon.stroke * icon.fills)
    expect(new Set(ink.map((one) => one.toFixed(6))).size).toBe(1)
  })

  it('is a width the grid an icon is drawn on can carry', () => {
    for (const [name, icon] of EVERY) {
      expect(icon.stroke, name).toBeGreaterThan(0)
      expect(icon.stroke, name).toBeLessThan(24)
      expect(icon.fills, name).toBeGreaterThan(0)
    }
  })
})

describe('what a key held is called', () => {
  it('is a word each, and no two keys are called the same', () => {
    const said = EVERY.map(([, icon]) => icon.said)
    expect(said.every((word) => word !== '')).toBe(true)
    expect(new Set(said).size).toBe(said.length)
  })
})
