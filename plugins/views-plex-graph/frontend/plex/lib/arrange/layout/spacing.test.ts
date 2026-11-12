/** Expanding the gaps into the space the window leaves. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { DEFAULT_OPTIONS } from '../options'
import { spacingAsSet, findSpacing, type Spacing } from './spacing'
import { build } from '../../../fixtures/build'
import type { PlexFrame } from '../../frame'

const { nodeSize, focusSize, gap, lineGap, focusGap, margin, spread } = DEFAULT_OPTIONS

const children = (frame: PlexFrame) => frame.nodes.filter((node) => node.relation === 'child')

const getSeparation = (frame: PlexFrame): number => {
  const nodes = children(frame)
  let smallest = Infinity
  nodes.forEach((a, index) => {
    for (const b of nodes.slice(index + 1)) {
      const sideways = Math.abs(a.x - b.x) - (a.width + b.width) / 2
      const lengthways = Math.abs(a.y - b.y) - (a.height + b.height) / 2
      smallest = Math.min(smallest, Math.max(sideways, lengthways))
    }
  })
  return smallest
}

const getNearestY = (frame: PlexFrame): number => Math.min(...children(frame).map((node) => node.y))

const findEscapedNodes = (frame: PlexFrame, viewport: { width: number; height: number }) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin,
  )

describe('a gap is never given less than its setting', () => {
  it('packs a window that holds the group to the settings and no closer', () => {
    const viewport = { width: 1200, height: 800 }
    const frame = arrangePlex(build('A node', { child: 11 }), { options: { viewport } })
    expect(getSeparation(frame)).toBeGreaterThanOrEqual(Math.min(gap, lineGap) - 0.5)
  })

  it('is the whole of it at a spread of one', () => {
    const viewport = { width: 2400, height: 1400 }
    const frame = arrangePlex(build('A node', { child: 1 }), {
      options: { viewport, spread: 1 },
    })
    expect(getNearestY(frame)).toBe(focusSize.height / 2 + focusGap + nodeSize.height / 2)
  })
})

describe('a gap opens into the room the window leaves', () => {
  it('spaces a short group out in a wide window', () => {
    const viewport = { width: 2400, height: 1400 }
    const settings = arrangePlex(build('A node', { child: 3 }))
    const frame = arrangePlex(build('A node', { child: 3 }), { options: { viewport } })
    expect(getSeparation(frame)).toBeGreaterThan(getSeparation(settings))
  })

  it('opens no further than the spread allows, however much room there is', () => {
    const roomy = arrangePlex(build('A node', { child: 3 }), {
      options: { viewport: { width: 4000, height: 3000 } },
    })
    expect(getSeparation(roomy)).toBeLessThanOrEqual(spread * Math.max(gap, lineGap) + 1e-6)
  })

  it('leaves the settings alone when there is no window to measure', () => {
    const frame = arrangePlex(build('A node', { child: 1 }))
    expect(getNearestY(frame)).toBe(focusSize.height / 2 + focusGap + nodeSize.height / 2)
  })
})

describe('the gap an edge crosses opens first', () => {
  it('places the first children further from the focus when the window has room', () => {
    const frame = arrangePlex(build('A node', { child: 1 }), {
      options: { viewport: { width: 2400, height: 1400 } },
    })
    expect(getNearestY(frame)).toBeGreaterThan(focusSize.height / 2 + focusGap + nodeSize.height / 2)
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

  it.each(WINDOWS)('$width×$height, every relation taken', (viewport) => {
    const full = build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })
    const frame = arrangePlex(full, { options: { viewport, spread: 4 } })
    expect(findEscapedNodes(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })
})

describe('findSpacing', () => {
  const options = { ...DEFAULT_OPTIONS, viewport: { width: 1200, height: 800 } }

  it('gives the settings when nothing at all fits', () => {
    expect(findSpacing(options, () => false)).toStrictEqual(spacingAsSet(options))
  })

  it('gives the settings when the window is not known', () => {
    expect(findSpacing(DEFAULT_OPTIONS, () => true)).toStrictEqual(spacingAsSet(DEFAULT_OPTIONS))
  })

  it('opens every gap to the spread when everything fits', () => {
    expect(findSpacing(options, () => true)).toStrictEqual({
      gap: options.gap * spread,
      lineGap: options.lineGap * spread,
      focusGap: options.focusGap * spread,
    })
  })

  it('asks about the focus gap before either of the others', () => {
    const request: (keyof Spacing)[] = []
    const activeSet = spacingAsSet(options)
    findSpacing(options, (spacing) => {
      for (const key of ['focusGap', 'lineGap', 'gap'] as const) {
        if (spacing[key] !== activeSet[key] && !request.includes(key)) request.push(key)
      }
      return true
    })
    expect(request).toStrictEqual(['focusGap', 'lineGap', 'gap'])
  })

  it('asks every time against the window the options give', () => {
    const windows = new Set<unknown>()
    findSpacing(options, (_spacing, viewport) => {
      windows.add(viewport)
      return true
    })
    expect([...windows]).toStrictEqual([options.viewport])
  })

  it('never gives a gap less than its setting, whatever spread it is given', () => {
    expect(findSpacing({ ...options, spread: 0.5 }, () => true)).toStrictEqual(
      spacingAsSet(options),
    )
  })

  it('asks nothing at all at a spread of one', () => {
    let request = 0
    findSpacing({ ...options, spread: 1 }, () => {
      request++
      return true
    })
    expect(request).toBe(0)
  })

  describe('how closely the widest opening is found', () => {
    const wideOptions = { ...options, spread: 3 }
    const activeSet = spacingAsSet(wideOptions)

    /** An arrangement that fits while no gap is opened past `limit` times its setting. */
    const fitsUpTo =
      (limit: number) =>
      (spacing: Spacing): boolean =>
        spacing.gap <= activeSet.gap * limit &&
        spacing.lineGap <= activeSet.lineGap * limit &&
        spacing.focusGap <= activeSet.focusGap * limit

    it('lands within a 256th of the range below the limit', () => {
      const found = findSpacing(wideOptions, fitsUpTo(1.5))
      for (const key of ['gap', 'lineGap', 'focusGap'] as const) {
        expect(found[key]).toBeLessThanOrEqual(activeSet[key] * 1.5)
        expect(found[key]).toBeGreaterThanOrEqual(activeSet[key] * (1.5 - 2 / 256))
      }
    })

    it('stops short of a limit that lies inside the last step', () => {
      expect(findSpacing(wideOptions, fitsUpTo(1 + 1 / 256))).toStrictEqual(activeSet)
    })
  })
})
