/**
 * Where a menu goes, and where the keyboard goes in it. Plain values, so the
 * awkward cases are cheap: a menu at the far corner, and one bigger than the
 * area it has to fit in.
 */
import { describe, expect, it } from 'vitest'
import { placeMenu, stepTo, type MenuItem } from './model'

const VIEWPORT = { width: 1000, height: 800 }
const SIZE = { width: 200, height: 300 }

const placeAt = (x: number, y: number, over: Partial<Parameters<typeof placeMenu>[0]> = {}) =>
  placeMenu({ at: { x, y }, size: SIZE, viewport: VIEWPORT, margin: 8, ...over })

describe('where a menu goes', () => {
  it('runs on from the point it was asked for', () => {
    expect(placeAt(100, 120)).toStrictEqual({ x: 100, y: 120 })
  })

  it('runs back over the point when the far edge is nearer than its length', () => {
    expect(placeAt(900, 700)).toStrictEqual({ x: 700, y: 400 })
  })

  it('folds one way and not the other, each edge on its own', () => {
    expect(placeAt(900, 100)).toStrictEqual({ x: 700, y: 100 })
    expect(placeAt(100, 700)).toStrictEqual({ x: 100, y: 400 })
  })

  it('is brought in off an edge it was asked for right against', () => {
    expect(placeAt(2, 1)).toStrictEqual({ x: 8, y: 8 })
  })

  it('is brought in where folding back would take it off the near edge', () => {
    // Nothing fits either side of the point, so it is shifted rather than
    // folded, and the whole of it is on screen.
    const placed = placeMenu({
      at: { x: 120, y: 120 },
      size: SIZE,
      viewport: { width: 260, height: 340 },
      margin: 8,
    })
    expect(placed).toStrictEqual({ x: 52, y: 32 })
  })

  it('sits at the near edge when it is larger than the area it is placed in', () => {
    const placed = placeMenu({
      at: { x: 40, y: 40 },
      size: SIZE,
      viewport: { width: 160, height: 200 },
      margin: 8,
    })
    expect(placed).toStrictEqual({ x: 8, y: 8 })
  })

  it('keeps whatever clearance it was given', () => {
    expect(placeAt(0, 0, { margin: 24 })).toStrictEqual({ x: 24, y: 24 })
    expect(placeAt(0, 0, { margin: 0 })).toStrictEqual({ x: 0, y: 0 })
  })
})

describe('where the keyboard goes', () => {
  const items: MenuItem[] = [
    { id: 'one', text: 'One' },
    { id: 'two', text: 'Two', disabled: true },
    { id: 'three', text: 'Three' },
  ]

  it('lands on the first from nowhere, and on the last counting back', () => {
    expect(stepTo(items, -1, 1)).toBe(0)
    expect(stepTo(items, 0, -1)).toBe(2)
  })

  it('passes over what cannot be chosen', () => {
    expect(stepTo(items, 0, 1)).toBe(2)
    expect(stepTo(items, 2, -1)).toBe(0)
  })

  it('wraps at either end', () => {
    expect(stepTo(items, 2, 1)).toBe(0)
    expect(stepTo(items, 0, -1)).toBe(2)
  })

  it('answers nothing when there is nothing to land on', () => {
    expect(stepTo([], -1, 1)).toBe(-1)
    expect(stepTo([{ id: 'only', text: 'Only', disabled: true }], -1, 1)).toBe(-1)
  })

  it('stays where it is when one item is all there is to be on', () => {
    const only: MenuItem[] = [{ id: 'only', text: 'Only' }]
    expect(stepTo(only, 0, 1)).toBe(0)
    expect(stepTo(only, 0, -1)).toBe(0)
  })
})
