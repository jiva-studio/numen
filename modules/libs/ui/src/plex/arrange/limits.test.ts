/** Wrapping the arrangement into the window it is drawn in. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { limitsFor } from './limits'
import { DEFAULT_OPTIONS } from './options'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import { build } from '../fixtures/build'
import type { PlexFrame } from '../model'

/**
 * Anything that reaches past what the window leaves it.
 *
 * The margin counts, on both axes: it is what the setting means, and comparing
 * against the bare half-window lets a row run flush to the top of the screen
 * with nothing to say so.
 */
const escapes = (
  frame: PlexFrame,
  viewport: { width: number; height: number },
  margin = DEFAULT_OPTIONS.margin,
) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin,
  )

const WINDOWS = [
  { width: 1600, height: 1000 },
  { width: 1400, height: 900 },
  { width: 1200, height: 800 },
  { width: 900, height: 700 },
  // A window narrow enough that a column has to be given up rather than drawn
  // past the edge.
  { width: 700, height: 600 },
  { width: 600, height: 520 },
  { width: 560, height: 480 },
]

describe('nothing reaches past the edge', () => {
  it.each(WINDOWS)('$width×$height, heavily populated', (viewport) => {
    const heavy = build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })
    const frame = arrangePlex(heavy, { options: { viewport } })
    expect(escapes(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })

  it.each(WINDOWS)('$width×$height, past every limit', (viewport) => {
    const frame = arrangePlex(neighbourhoods.overcrowded, { options: { viewport } })
    expect(escapes(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })

  it.each(Object.entries(neighbourhoods))('%s', (_name, neighbourhood) => {
    const viewport = { width: 1200, height: 800 }
    const frame = arrangePlex(neighbourhood, { options: { viewport } })
    expect(escapes(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })
})

describe('the space above and below is used before anything is dropped', () => {
  it('wraps a wide row into more rows rather than hiding children', () => {
    const viewport = { width: 1400, height: 900 }
    const many = build('A node', { parent: 2, child: 15, jump: 3, sibling: 3 })
    const frame = arrangePlex(many, { options: { viewport, maxPerLine: 9 } })

    expect(frame.overflow.child).toBeUndefined()
    const rows = new Set(
      frame.nodes.filter((n) => n.seat === 'child').map((n) => n.y),
    )
    expect(rows.size).toBeGreaterThan(1)
  })

  it('lets a column run further down than a row may run wide', () => {
    // maxPerLine is a rule about how wide a row gets. Applying it to a column
    // would waste the height, which is the space a plex has most of.
    const viewport = { width: 1400, height: 900 }
    const sideHeavy = build('A node', { child: 2, jump: 9 })
    const frame = arrangePlex(sideHeavy, { options: { viewport, maxPerLine: 5 } })

    expect(frame.overflow.jump).toBeUndefined()
    const columns = new Set(frame.nodes.filter((n) => n.seat === 'jump').map((n) => n.x))
    expect(columns.size).toBe(1)
  })
})

describe('the two sides are read as a pair', () => {
  it('seats a short column as far out as a tall one', () => {
    // Worked out on its own reach, two jumps would sit nearer the focus than
    // seven siblings, and the plex would lean for a reason nobody can see.
    const lopsided = build('A node', { child: 6, jump: 2, sibling: 7 })
    const frame = arrangePlex(lopsided, {
      options: { viewport: { width: 1600, height: 1000 } },
    })

    const near = (seat: string) =>
      Math.min(
        ...frame.nodes.filter((n) => n.seat === seat).map((n) => Math.abs(n.x)),
      )
    expect(near('jump')).toBe(near('sibling'))
  })
})

describe('the row is measured against the window either way', () => {
  it('wraps a wide row even when nothing sits beside it', () => {
    // A row is measured against the window whether or not anything sits beside
    // it: with no jumps and no siblings there is no column to make room for,
    // and the row still has to fit.
    const viewport = { width: 700, height: 900 }
    const onlyChildren = build('A node', { child: 12 })
    const frame = arrangePlex(onlyChildren, {
      options: { viewport, maxPerLine: 9 },
    })
    expect(escapes(frame, viewport).map((n) => n.title)).toStrictEqual([])
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

  it('keeps the room when no row width seats a column at all', () => {
    // A window this narrow has nowhere to put a jump or a sibling, and the
    // room a column cannot use is the room the children are read in.
    const viewport = { width: 600, height: 1400 }
    const counts = { parent: 1, child: 6, jump: 1, sibling: 2 }
    const limits = limitsFor({ ...DEFAULT_OPTIONS, viewport }, counts)

    expect(limits.jump.lines).toBe(0)
    expect(limits.child.perLine).toBeGreaterThan(1)

    const frame = arrangePlex(build('A node', counts), { options: { viewport } })
    expect(frame.overflow.child).toBeUndefined()
    expect(escapes(frame, viewport).map((n) => n.title)).toStrictEqual([])
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
    const many = build('A node', { child: 20, jump: 4 })

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

describe('how many lines a column may run to', () => {
  it('is the number it was allowed, whatever the window makes room for', () => {
    // A wide window has room for more columns than the setting permits, and the
    // setting is what says how deep the picture is allowed to get. Without this
    // a side wraps into as many columns as will fit and reports no overflow.
    const viewport = { width: 2400, height: 1400 }
    const counts = { parent: 2, child: 2, jump: 60, sibling: 60 }

    for (const maxLines of [1, 2, 3]) {
      const limits = limitsFor({ ...DEFAULT_OPTIONS, viewport, maxLines }, counts)
      expect(limits.jump.lines).toBeLessThanOrEqual(maxLines)
      expect(limits.sibling.lines).toBeLessThanOrEqual(maxLines)
    }
  })

  it('is none at all when the window leaves no room beside the rows', () => {
    // Reported as overflow, which the reader can act on. Drawn past the edge of
    // the window, which they cannot.
    const viewport = { width: 420, height: 700 }
    const counts = { parent: 0, child: 6, jump: 4, sibling: 0 }
    const limits = limitsFor({ ...DEFAULT_OPTIONS, viewport }, counts)
    expect(limits.jump.lines).toBe(0)
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
    for (const seat of ['parent', 'child', 'jump', 'sibling'] as const) {
      expect(limits[seat]).toStrictEqual({
        perLine: DEFAULT_OPTIONS.maxPerLine,
        lines: DEFAULT_OPTIONS.maxLines,
      })
    }
  })
})
