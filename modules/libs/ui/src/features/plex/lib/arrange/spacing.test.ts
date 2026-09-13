/** Opening the gaps into the room the window leaves. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { spacingAsSet, spacingFor, type Spacing } from './spacing'
import { build } from '../../fixtures/build'
import type { PlexFrame } from '../frame'

const { nodeSize, focusSize, gap, focusGap, margin, spread } = DEFAULT_OPTIONS

/** The step from one node to the next along a line, at the settings. */
const TIGHT_STEP = nodeSize.width + gap

/** A window wide enough for eleven children at the settings and no wider. */
const EXACT = { width: 11 * nodeSize.width + 10 * gap + 2 * margin, height: 1000 }

const children = (frame: PlexFrame) => frame.nodes.filter((node) => node.seat === 'child')

const stepAlong = (frame: PlexFrame) => {
  const [first, second] = children(frame)
  return second!.x - first!.x
}

const firstLine = (frame: PlexFrame) => {
  const line = children(frame)[0]!.y
  return children(frame).filter((node) => node.y === line)
}

const findEscaped = (frame: PlexFrame, viewport: { width: number; height: number }) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin,
  )

describe('a gap is never given less than its setting', () => {
  it('packs a crowded window to the settings and no closer', () => {
    const viewport = { width: 700, height: 600 }
    const frame = arrangePlex(build('A node', { child: 11 }), { options: { viewport } })
    expect(stepAlong(frame)).toBeGreaterThanOrEqual(TIGHT_STEP)
  })

  it('is the whole of it at a spread of one', () => {
    const viewport = { width: 2400, height: 1400 }
    const frame = arrangePlex(build('A node', { child: 5 }), {
      options: { viewport, spread: 1 },
    })
    expect(stepAlong(frame)).toBe(TIGHT_STEP)
    expect(firstLine(frame)[0]!.y).toBe(focusSize.height / 2 + focusGap + nodeSize.height / 2)
  })
})

describe('a gap opens into the room the window leaves', () => {
  it('spaces a short row out in a wide window', () => {
    const viewport = { width: 2400, height: 1400 }
    const frame = arrangePlex(build('A node', { child: 3 }), { options: { viewport } })
    expect(stepAlong(frame)).toBeGreaterThan(TIGHT_STEP)
  })

  it('opens no further than the spread allows, however much room there is', () => {
    const roomy = arrangePlex(build('A node', { child: 3 }), {
      options: { viewport: { width: 4000, height: 3000 } },
    })
    expect(stepAlong(roomy)).toBeLessThanOrEqual(nodeSize.width + gap * spread)
  })

  it('leaves the settings alone when there is no window to measure', () => {
    const frame = arrangePlex(build('A node', { child: 3 }))
    expect(stepAlong(frame)).toBe(TIGHT_STEP)
  })
})

describe('the gap an edge crosses opens first', () => {
  it('seats the first line further down when the width is already spoken for', () => {
    // Eleven children fill this window edge to edge, so nothing opens sideways.
    // The height is untouched, and the label an edge carries is read there.
    const frame = arrangePlex(build('A node', { child: 11 }), {
      options: { viewport: EXACT },
    })
    expect(stepAlong(frame)).toBe(TIGHT_STEP)
    expect(firstLine(frame)[0]!.y).toBeGreaterThan(
      focusSize.height / 2 + focusGap + nodeSize.height / 2,
    )
  })
})

describe('a line runs as long as the window holds', () => {
  it('seats eleven children on one line when the width is there', () => {
    const frame = arrangePlex(build('A node', { child: 11 }), {
      options: { viewport: { width: 1970, height: 1440 } },
    })
    expect(firstLine(frame)).toHaveLength(11)
  })

  it('wraps the same eleven when it is not', () => {
    const frame = arrangePlex(build('A node', { child: 11 }), {
      options: { viewport: { width: 1200, height: 800 } },
    })
    expect(firstLine(frame).length).toBeLessThan(11)
    expect(frame.overflow.child).toBeUndefined()
  })
})

describe('what opens still keeps the window', () => {
  const WINDOWS = [
    { width: 2400, height: 1400 },
    { width: 1970, height: 1440 },
    { width: 1200, height: 800 },
    { width: 700, height: 600 },
    { width: 560, height: 480 },
  ]

  it.each(WINDOWS)('$width×$height, every seat taken', (viewport) => {
    const full = build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })
    const frame = arrangePlex(full, { options: { viewport, spread: 4 } })
    expect(findEscaped(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })
})

describe('spacingFor', () => {
  const options = { ...DEFAULT_OPTIONS, viewport: { width: 1200, height: 800 } }

  it('gives the settings when nothing at all fits', () => {
    expect(spacingFor(options, () => false)).toStrictEqual(spacingAsSet(options))
  })

  it('gives the settings when the window is not known', () => {
    expect(spacingFor(DEFAULT_OPTIONS, () => true)).toStrictEqual(spacingAsSet(DEFAULT_OPTIONS))
  })

  it('opens every gap to the spread when everything fits', () => {
    expect(spacingFor(options, () => true)).toStrictEqual({
      gap: options.gap * spread,
      lineGap: options.lineGap * spread,
      focusGap: options.focusGap * spread,
    })
  })

  it('asks about the focus gap before either of the others', () => {
    const asked: (keyof Spacing)[] = []
    const set = spacingAsSet(options)
    spacingFor(options, (spacing) => {
      for (const key of ['focusGap', 'lineGap', 'gap'] as const) {
        if (spacing[key] !== set[key] && !asked.includes(key)) asked.push(key)
      }
      return true
    })
    expect(asked).toStrictEqual(['focusGap', 'lineGap', 'gap'])
  })
})
