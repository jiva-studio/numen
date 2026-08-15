/** Wrapping the arrangement into the window it is drawn in. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { limitsFor } from './limits'
import { DEFAULT_OPTIONS } from './options'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import { build } from '../fixtures/build'
import type { PlexFrame } from '../model'

const escapes = (frame: PlexFrame, viewport: { width: number; height: number }) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2,
  )

const WINDOWS = [
  { width: 1600, height: 1000 },
  { width: 1400, height: 900 },
  { width: 1200, height: 800 },
  { width: 900, height: 700 },
]

describe('nothing reaches past the edge', () => {
  it.each(WINDOWS)('$width×$height, heavily populated', (viewport) => {
    const heavy = build('A thought', { parent: 6, child: 21, jump: 7, sibling: 7 })
    const frame = arrangePlex(heavy, { options: { viewport } })
    expect(escapes(frame, viewport).map((n) => n.label)).toStrictEqual([])
  })

  it.each(WINDOWS)('$width×$height, past every limit', (viewport) => {
    const frame = arrangePlex(neighbourhoods.overcrowded, { options: { viewport } })
    expect(escapes(frame, viewport).map((n) => n.label)).toStrictEqual([])
  })

  it.each(Object.entries(neighbourhoods))('%s', (_name, neighbourhood) => {
    const viewport = { width: 1200, height: 800 }
    const frame = arrangePlex(neighbourhood, { options: { viewport } })
    expect(escapes(frame, viewport).map((n) => n.label)).toStrictEqual([])
  })
})

describe('the space above and below is used before anything is dropped', () => {
  it('wraps a wide row into more rows rather than hiding children', () => {
    const viewport = { width: 1400, height: 900 }
    const many = build('A thought', { parent: 2, child: 15, jump: 3, sibling: 3 })
    const frame = arrangePlex(many, { options: { viewport, maxPerLine: 9 } })

    expect(frame.overflow.child).toBeUndefined()
    const rows = new Set(
      frame.nodes.filter((n) => n.role === 'child').map((n) => n.y),
    )
    expect(rows.size).toBeGreaterThan(1)
  })

  it('lets a column run further down than a row may run wide', () => {
    // maxPerLine is a rule about how wide a row gets. Applying it to a column
    // would waste the height, which is the space a plex has most of.
    const viewport = { width: 1400, height: 900 }
    const sideHeavy = build('A thought', { child: 2, jump: 9 })
    const frame = arrangePlex(sideHeavy, { options: { viewport, maxPerLine: 5 } })

    expect(frame.overflow.jump).toBeUndefined()
    const columns = new Set(frame.nodes.filter((n) => n.role === 'jump').map((n) => n.x))
    expect(columns.size).toBe(1)
  })
})

describe('the two sides are read as a pair', () => {
  it('seats a short column as far out as a tall one', () => {
    // Worked out on its own reach, two jumps would sit nearer the focus than
    // seven siblings, and the plex would lean for a reason nobody can see.
    const lopsided = build('A thought', { child: 6, jump: 2, sibling: 7 })
    const frame = arrangePlex(lopsided, {
      options: { viewport: { width: 1600, height: 1000 } },
    })

    const near = (role: string) =>
      Math.min(
        ...frame.nodes.filter((n) => n.role === role).map((n) => Math.abs(n.x)),
      )
    expect(near('jump')).toBe(near('sibling'))
  })
})

describe('the row is measured against the window either way', () => {
  it('wraps a wide row even when nothing sits beside it', () => {
    // The row used to be checked only against the room a column needed, so a
    // plex with no jumps or siblings was never checked against the window.
    const viewport = { width: 700, height: 900 }
    const onlyChildren = build('A thought', { child: 12 })
    const frame = arrangePlex(onlyChildren, {
      options: { viewport, maxPerLine: 9 },
    })
    expect(escapes(frame, viewport).map((n) => n.label)).toStrictEqual([])
  })

})

describe('the rows keep the room when it runs out', () => {
  it('narrows a row only as far as it must to seat a column beside it', () => {
    const viewport = { width: 1000, height: 800 }
    const limits = limitsFor(
      { ...DEFAULT_OPTIONS, viewport },
      { parent: 0, child: 12, jump: 4, sibling: 0 },
    )
    expect(limits.child.perLine).toBeGreaterThan(1)
    expect(limits.jump.lines).toBeGreaterThanOrEqual(1)
  })

  it('gives a row the whole window when nothing sits beside it', () => {
    const viewport = { width: 1000, height: 800 }
    const alone = limitsFor(
      { ...DEFAULT_OPTIONS, viewport },
      { parent: 0, child: 12, jump: 0, sibling: 0 },
    )
    const crowded = limitsFor(
      { ...DEFAULT_OPTIONS, viewport },
      { parent: 0, child: 12, jump: 4, sibling: 0 },
    )
    expect(alone.child.perLine).toBeGreaterThanOrEqual(crowded.child.perLine)
  })
})

describe('the margin is a setting, not a number in the source', () => {
  it('keeps the drawing that far clear of the edge', () => {
    const viewport = { width: 1400, height: 900 }
    const many = build('A thought', { child: 20, jump: 4 })

    for (const margin of [0, 16, 120]) {
      const frame = arrangePlex(many, { options: { viewport, margin } })
      const reach = Math.max(...frame.nodes.map((n) => Math.abs(n.x) + n.width / 2))
      expect(reach).toBeLessThanOrEqual(viewport.width / 2 - margin)
    }
  })

  it('fits less as the margin grows', () => {
    const viewport = { width: 1000, height: 800 }
    const counts = { parent: 0, child: 12, jump: 4, sibling: 0 }
    const tight = limitsFor({ ...DEFAULT_OPTIONS, viewport, margin: 200 }, counts)
    const loose = limitsFor({ ...DEFAULT_OPTIONS, viewport, margin: 0 }, counts)
    expect(tight.child.perLine).toBeLessThan(loose.child.perLine)
  })
})

describe('given no window', () => {
  it('takes the limits the caller asked for literally', () => {
    const limits = limitsFor(DEFAULT_OPTIONS, {
      parent: 3,
      child: 3,
      jump: 3,
      sibling: 3,
    })
    for (const role of ['parent', 'child', 'jump', 'sibling'] as const) {
      expect(limits[role]).toStrictEqual({
        perLine: DEFAULT_OPTIONS.maxPerLine,
        lines: DEFAULT_OPTIONS.maxLines,
      })
    }
  })
})
