/**
 * Where something standing over the page goes. Plain values, so the awkward
 * cases are cheap: a thing at the far edge, and one bigger than the area it
 * has to fit in.
 */
import { describe, expect, it } from 'vitest'
import { getPlaceBeside } from './place'

/** A span from 100 to 120, in an area of 240, with a thing 60 to place. */
const span = (over: Partial<Parameters<typeof getPlaceBeside>[0]> = {}) =>
  getPlaceBeside({ from: 100, to: 120, size: 60, room: 240, margin: 8, gap: 8, ...over })

describe('where a thing standing beside a span goes', () => {
  it('runs on from the far end of the span, clear of it', () => {
    expect(span()).toBe(128)
  })

  it('runs back from the near end where the far edge is nearer than its size', () => {
    expect(span({ room: 180 })).toBe(32)
  })

  it('is brought inside the edge where neither side has room for it', () => {
    expect(span({ from: 20, to: 40, size: 200 })).toBe(32)
  })

  it('sits at the near edge when it is larger than the area it is placed in', () => {
    expect(span({ size: 300 })).toBe(8)
  })

  it('touches the span it stands beside when it is given no gap', () => {
    expect(span({ gap: 0 })).toBe(120)
  })

  it('keeps whatever clearance it was given', () => {
    expect(span({ from: 0, to: 0, gap: 0, margin: 24 })).toBe(24)
    expect(span({ from: 0, to: 0, gap: 0, margin: 0 })).toBe(0)
  })
})
