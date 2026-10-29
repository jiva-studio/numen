/** Packing the layout more tightly before any relation is dropped. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { measureCrowding, packOptions } from './crowding'
import { getLimits } from './limits'
import { DEFAULT_OPTIONS } from '../options'
import { build } from '../../../fixtures/build'
import type { PlexFrame } from '../../frame'
import { RELATION_NAMES } from '../../relation'

const { nodeSize, minWidth, gap, squeeze, margin } = DEFAULT_OPTIONS

/** The neighbourhood on which a narrow window dropped relations. */
const COUNTS = { parent: 1, child: 6, jump: 1, sibling: 2 }

const findEscaped = (frame: PlexFrame, viewport: { width: number; height: number }) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin,
  )

const getChildWidth = (frame: PlexFrame) =>
  frame.nodes.find((node) => node.relation === 'child')!.width

describe('a box narrows and a gap closes before a relation is given up', () => {
  const NARROW = [
    { width: 440, height: 1000 },
    { width: 500, height: 1000 },
    { width: 600, height: 1000 },
    { width: 700, height: 1000 },
    { width: 900, height: 1000 },
  ]

  it.each(NARROW)('$width×$height places every node', (viewport) => {
    const frame = arrangePlex(build('A node', COUNTS), { options: { viewport } })
    expect(frame.overflow).toStrictEqual({})
  })

  it.each(NARROW)('$width×$height keeps every node inside the window', (viewport) => {
    const frame = arrangePlex(build('A node', COUNTS), { options: { viewport } })
    expect(findEscaped(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })

  it('draws a narrower box the less room the window has', () => {
    const boxes = NARROW.map((viewport) =>
      getChildWidth(arrangePlex(build('A node', COUNTS), { options: { viewport } })),
    )
    for (let i = 1; i < boxes.length; i++) {
      expect(boxes[i]!).toBeGreaterThanOrEqual(boxes[i - 1]!)
    }
    expect(boxes[0]!).toBeLessThan(nodeSize.width)
  })
})

describe('no closer than the window needs', () => {
  it('leaves a roomy window at the settings', () => {
    const options = { ...DEFAULT_OPTIONS, viewport: { width: 2000, height: 1400 } }
    expect(measureCrowding(options, COUNTS)).toStrictEqual(options)
  })

  it('draws every box at its widest where there is room for it', () => {
    const frame = arrangePlex(build('A node', COUNTS), {
      options: { viewport: { width: 2000, height: 1400 } },
    })
    expect(getChildWidth(frame)).toBe(nodeSize.width)
  })

  it('leaves a plex with no window alone', () => {
    expect(measureCrowding(DEFAULT_OPTIONS, COUNTS)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('packs nothing at a squeeze of one', () => {
    const options = {
      ...DEFAULT_OPTIONS,
      squeeze: 1,
      viewport: { width: 440, height: 1000 },
    }
    expect(measureCrowding(options, COUNTS)).toStrictEqual(options)
  })
})

describe('how closely a plex is packed', () => {
  it('is its settings at one', () => {
    expect(packOptions(DEFAULT_OPTIONS, 1)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('is the narrowest box and the closest gap at nothing', () => {
    const closest = packOptions(DEFAULT_OPTIONS, 0)
    expect(closest.nodeSize.width).toBe(minWidth)
    expect(closest.gap).toBeCloseTo(gap * squeeze)
  })

  it('never draws a box narrower than the narrowest, whatever it is packed to', () => {
    for (const tightness of [0, 0.1, 0.5, 0.9, 1]) {
      expect(packOptions(DEFAULT_OPTIONS, tightness).nodeSize.width).toBeGreaterThanOrEqual(
        minWidth,
      )
    }
  })

  it('closes every gap to the fraction of itself that the squeeze allows', () => {
    const closest = packOptions(DEFAULT_OPTIONS, 0)
    expect(closest.lineGap).toBeCloseTo(DEFAULT_OPTIONS.lineGap * squeeze)
    expect(closest.focusGap).toBeCloseTo(DEFAULT_OPTIONS.focusGap * squeeze)
  })

  it('leaves the heights, the counts and the window where they were', () => {
    const closest = packOptions(DEFAULT_OPTIONS, 0)
    expect(closest.nodeSize.height).toBe(nodeSize.height)
    expect(closest.maxPerLine).toBe(DEFAULT_OPTIONS.maxPerLine)
    expect(closest.maxLines).toBe(DEFAULT_OPTIONS.maxLines)
  })
})

describe('the loosest packing that fits everything', () => {
  const STEPS = 2 ** 7

  const canRelationAll = (options: typeof DEFAULT_OPTIONS) => {
    const limits = getLimits(options, COUNTS)
    return RELATION_NAMES.every(
      (relation) => limits[relation].perLine * limits[relation].lines >= COUNTS[relation],
    )
  }

  const getTightness = (packed: typeof DEFAULT_OPTIONS) =>
    (packed.nodeSize.width - minWidth) / (nodeSize.width - minWidth)

  it.each([
    { width: 440, height: 1000 },
    { width: 500, height: 1000 },
    { width: 600, height: 1000 },
  ])('$width×$height is found to a 128th, and one step looser relations less', (viewport) => {
    const options = { ...DEFAULT_OPTIONS, viewport }
    const packed = measureCrowding(options, COUNTS)
    const tightness = getTightness(packed)

    expect(canRelationAll(packed)).toBe(true)
    expect(tightness).toBeGreaterThanOrEqual(0)
    expect(tightness).toBeLessThan(1)
    expect(tightness * STEPS).toBeCloseTo(Math.round(tightness * STEPS))
    expect(canRelationAll(packOptions(options, tightness + 1 / STEPS))).toBe(false)
  })

  it('packs a window that is crowded at the settings tighter than the settings', () => {
    const options = { ...DEFAULT_OPTIONS, viewport: { width: 500, height: 1000 } }
    expect(getTightness(measureCrowding(options, COUNTS))).toBeGreaterThan(0)
  })

  it('leaves a window alone that holds the nodes with no relation to spare', () => {
    // Twelve to a line and four lines: forty-eight children fill the relation exactly.
    const options = { ...DEFAULT_OPTIONS, viewport: { width: 2000, height: 1400 } }
    const counts = { parent: 0, child: 48, jump: 0, sibling: 0 }
    expect(measureCrowding(options, counts)).toStrictEqual(options)
  })
})

describe('what no packing can fit', () => {
  it('is still reported and not drawn past the edge', () => {
    const viewport = { width: 400, height: 400 }
    const crowd = build('A node', { parent: 9, child: 40, jump: 9, sibling: 9 })
    const frame = arrangePlex(crowd, { options: { viewport } })

    expect(Object.keys(frame.overflow).length).toBeGreaterThan(0)
    expect(findEscaped(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })
})
