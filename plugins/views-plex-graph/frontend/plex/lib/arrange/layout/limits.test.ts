/** Wrapping the arrangement into the window it is drawn in. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { getLimits } from './limits'
import { DEFAULT_OPTIONS } from '../options'
import { neighbourhoods } from '../../../fixtures/neighbourhoods'
import { build } from '../../../fixtures/build'
import type { PlexFrame } from '../../frame'

/**
 * Nodes that extend beyond the viewport minus the margin.
 *
 * The margin applies on both axes, as the setting specifies; comparing against
 * the bare half-viewport would let a row touch the top of the screen.
 */
const findEscaped = (
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
  // A window narrow enough that a column has to be given up and not drawn
  // past the edge.
  { width: 700, height: 600 },
  { width: 600, height: 520 },
  { width: 560, height: 480 },
]

describe('nothing reaches past the edge', () => {
  it.each(WINDOWS)('$width×$height, heavily populated', (viewport) => {
    const heavy = build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })
    const frame = arrangePlex(heavy, { options: { viewport } })
    expect(findEscaped(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })

  it.each(WINDOWS)('$width×$height, past every limit', (viewport) => {
    const frame = arrangePlex(neighbourhoods.overcrowded, { options: { viewport } })
    expect(findEscaped(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })

  it.each(Object.entries(neighbourhoods))('%s', (_name, neighbourhood) => {
    const viewport = { width: 1200, height: 800 }
    const frame = arrangePlex(neighbourhood, { options: { viewport } })
    expect(findEscaped(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })
})

describe('the space above and below is used before anything is dropped', () => {
  it('wraps a wide row into more rows and not hiding children', () => {
    const viewport = { width: 1400, height: 900 }
    const many = build('A node', { parent: 2, child: 15, jump: 3, sibling: 3 })
    const frame = arrangePlex(many, { options: { viewport, maxPerLine: 9 } })

    expect(frame.overflow.child).toBeUndefined()
    const rows = new Set(frame.nodes.filter((n) => n.relation === 'child').map((n) => n.y))
    expect(rows.size).toBeGreaterThan(1)
  })

  it('lets a column run further down than a row may run wide', () => {
    // maxPerLine is a rule about how wide a row gets. Applying it to a column
    // would waste the height, which is the space a plex has most of.
    const viewport = { width: 1400, height: 900 }
    const sideHeavy = build('A node', { child: 2, jump: 9 })
    const frame = arrangePlex(sideHeavy, { options: { viewport, maxPerLine: 5 } })

    expect(frame.overflow.jump).toBeUndefined()
    const columns = new Set(frame.nodes.filter((n) => n.relation === 'jump').map((n) => n.x))
    expect(columns.size).toBe(1)
  })
})

describe('the two sides are read as a pair', () => {
  it('fits a short column as far out as a tall one', () => {
    // Sized by its own extent, two jumps would sit nearer the focus than seven
    // siblings, and the layout would be visibly lopsided.
    const lopsided = build('A node', { child: 6, jump: 2, sibling: 7 })
    const frame = arrangePlex(lopsided, {
      options: { viewport: { width: 1600, height: 1000 } },
    })

    const getNearest = (relation: string) =>
      Math.min(...frame.nodes.filter((n) => n.relation === relation).map((n) => Math.abs(n.x)))
    expect(getNearest('jump')).toBe(getNearest('sibling'))
  })
})

describe('the row is measured against the window either way', () => {
  it('wraps a wide row even when nothing sits alongside it', () => {
    // A row is measured against the viewport whether or not anything is adjacent
    // to it: with no jumps and no siblings there is no column to reserve space
    // for, and the row still has to fit.
    const viewport = { width: 700, height: 900 }
    const onlyChildren = build('A node', { child: 12 })
    const frame = arrangePlex(onlyChildren, {
      options: { viewport, maxPerLine: 9 },
    })
    expect(findEscaped(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })
})

describe('the rows keep the room when it runs out', () => {
  it('narrows a row only as far as it must to fit a column alongside it', () => {
    const viewport = { width: 1000, height: 800 }
    const limits = getLimits(
      { ...DEFAULT_OPTIONS, viewport },
      { parent: 0, child: 12, jump: 4, sibling: 0 },
    )
    expect(limits.child.perLine).toBeGreaterThan(1)
    expect(limits.jump.lines).toBeGreaterThanOrEqual(1)
  })

  it('keeps the room when no row width fits a column at all', () => {
    // A viewport this narrow has no space for a jump or a sibling, and the space a
    // column cannot use is kept for the children.
    const viewport = { width: 600, height: 1400 }
    const counts = { parent: 1, child: 6, jump: 1, sibling: 2 }
    const limits = getLimits({ ...DEFAULT_OPTIONS, viewport }, counts)

    expect(limits.jump.lines).toBe(0)
    expect(limits.child.perLine).toBeGreaterThan(1)

    const frame = arrangePlex(build('A node', counts), { options: { viewport } })
    expect(frame.overflow.child).toBeUndefined()
    expect(findEscaped(frame, viewport).map((n) => n.title)).toStrictEqual([])
  })

  it('gives a row the whole window when nothing sits alongside it', () => {
    const viewport = { width: 1000, height: 800 }
    const alone = getLimits(
      { ...DEFAULT_OPTIONS, viewport },
      { parent: 0, child: 12, jump: 0, sibling: 0 },
    )
    const crowded = getLimits(
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
    const largeMarginLimits = getLimits({ ...DEFAULT_OPTIONS, viewport, margin: 200 }, counts)
    const zeroMarginLimits = getLimits({ ...DEFAULT_OPTIONS, viewport, margin: 0 }, counts)
    expect(largeMarginLimits.child.perLine).toBeLessThan(zeroMarginLimits.child.perLine)
  })
})

describe('how many lines a column may run to', () => {
  it('is the number it was allowed, whatever the window makes room for', () => {
    // A wide viewport has space for more columns than the setting permits, and the
    // setting limits the depth. Without the limit a side wraps into as many
    // columns as fit and reports no overflow.
    const viewport = { width: 2400, height: 1400 }
    const counts = { parent: 2, child: 2, jump: 60, sibling: 60 }

    for (const maxLines of [1, 2, 3]) {
      const limits = getLimits({ ...DEFAULT_OPTIONS, viewport, maxLines }, counts)
      expect(limits.jump.lines).toBeLessThanOrEqual(maxLines)
      expect(limits.sibling.lines).toBeLessThanOrEqual(maxLines)
    }
  })

  it('is none at all when the window leaves no room alongside the rows', () => {
    // Reported as overflow, which the reader can act on. Drawn past the edge of
    // the window, which they cannot.
    const viewport = { width: 420, height: 700 }
    const counts = { parent: 0, child: 6, jump: 4, sibling: 0 }
    const limits = getLimits({ ...DEFAULT_OPTIONS, viewport }, counts)
    expect(limits.jump.lines).toBe(0)
  })
})

describe('given no window', () => {
  it('takes the limits the caller requested for literally', () => {
    const limits = getLimits(DEFAULT_OPTIONS, {
      parent: 3,
      child: 3,
      jump: 3,
      sibling: 3,
    })
    for (const relation of ['parent', 'child', 'jump', 'sibling'] as const) {
      expect(limits[relation]).toStrictEqual({
        perLine: DEFAULT_OPTIONS.maxPerLine,
        lines: DEFAULT_OPTIONS.maxLines,
      })
    }
  })
})

describe('what a column asks of the rows', () => {
  const viewport = { width: 1000, height: 800 }
  const getRows = (jump: number, overrides: Partial<typeof DEFAULT_OPTIONS> = {}) =>
    getLimits(
      { ...DEFAULT_OPTIONS, viewport, ...overrides },
      { parent: 0, child: 12, jump, sibling: 0 },
    ).child.perLine

  it.each([1, 2, 3])(
    'leaves the row whole beside a column of %i, which stays level with the focus',
    (jump) => {
      expect(getRows(jump)).toBe(6)
    },
  )

  it('narrows the row once the column reaches the rows above and below', () => {
    expect(getRows(4)).toBe(3)
  })

  it('leaves the row whole when the column reaches exactly where the rows begin', () => {
    // Three nodes stand 70 from the axis; the first row begins 22 + 48 away.
    expect(getRows(3, { focusGap: 48 })).toBe(6)
    expect(getRows(3, { focusGap: 47 })).toBeLessThan(6)
  })

  it('keeps the widest row the window holds when no row width fits a column alongside it', () => {
    const narrow = getLimits(
      { ...DEFAULT_OPTIONS, viewport: { width: 600, height: 1400 } },
      { parent: 0, child: 6, jump: 4, sibling: 0 },
    )
    expect(narrow.child.perLine).toBe(3)
  })

  it('gives the row the whole window where the columns are empty', () => {
    expect(getRows(0)).toBe(6)
  })

  it('counts no row as a column, however long it runs', () => {
    const limits = getLimits(
      { ...DEFAULT_OPTIONS, viewport: { width: 1000, height: 2000 } },
      { parent: 0, child: 30, jump: 0, sibling: 0 },
    )
    expect(limits.child.perLine).toBe(6)
  })

  it('assigns single row when box exceeds window width', () => {
    const limits = getLimits(
      {
        ...DEFAULT_OPTIONS,
        nodeSize: { width: 600, height: 36 },
        viewport: { width: 500, height: 800 },
      },
      { parent: 0, child: 6, jump: 0, sibling: 0 },
    )
    expect(limits.child.perLine).toBe(1)
  })

  it('assigns single row when focus exceeds window width', () => {
    const limits = getLimits(
      {
        ...DEFAULT_OPTIONS,
        focusSize: { width: 600, height: 44 },
        viewport: { width: 500, height: 800 },
      },
      { parent: 0, child: 6, jump: 0, sibling: 0 },
    )
    expect(limits.child.perLine).toBe(1)
  })
})

describe('how deep the rows run', () => {
  it('counts the lines from the bottom of the focus box', () => {
    const limits = getLimits(
      { ...DEFAULT_OPTIONS, viewport: { width: 1000, height: 500 } },
      { parent: 1, child: 1, jump: 0, sibling: 0 },
    )
    expect(limits.child.lines).toBe(3)
    expect(limits.parent.lines).toBe(3)
  })
})

describe('the columns of a side', () => {
  const viewport = { width: 2400, height: 1400 }

  it('run to one line when one line holds them', () => {
    const limits = getLimits(
      { ...DEFAULT_OPTIONS, viewport },
      { parent: 0, child: 0, jump: 2, sibling: 2 },
    )
    expect(limits.jump.lines).toBe(1)
    expect(limits.sibling.lines).toBe(1)
  })

  it('run to one line when there is nothing in them', () => {
    const limits = getLimits(
      { ...DEFAULT_OPTIONS, viewport: { width: 1000, height: 800 } },
      { parent: 0, child: 5, jump: 0, sibling: 0 },
    )
    expect(limits.jump.lines).toBe(1)
    expect(limits.sibling.lines).toBe(1)
  })
})
