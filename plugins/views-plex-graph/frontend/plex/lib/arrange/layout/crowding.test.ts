/** Packing the layout more tightly before any relation is dropped. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { measureCrowding, packOptions } from './crowding'
import { DEFAULT_OPTIONS, type PlexOptions } from '../options'
import { build } from '../../../fixtures/build'
import type { PlexFrame } from '../../frame'

const { nodeSize, minWidth, gap, squeeze, margin } = DEFAULT_OPTIONS

/** The graph on which a narrow window dropped relations. */
const COUNTS = { parent: 1, child: 6, jump: 1, sibling: 2 }

const holdsAll = (frame: PlexFrame) => Object.keys(frame.overflow).length === 0

const findEscapedNodes = (frame: PlexFrame, viewport: { width: number; height: number }) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin,
  )

/** Whether the window holds every node of the graph at the given packing. */
const canHoldAll = (packed: PlexOptions): boolean => {
  const { viewport } = packed
  const frame = arrangePlex(build('A node', COUNTS), {
    options: { ...packed, viewport: undefined },
  })
  return holdsAll(frame) && findEscapedNodes(frame, viewport!).length === 0
}

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
    expect(findEscapedNodes(frame, viewport).map((node) => node.title)).toStrictEqual([])
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
    expect(measureCrowding(options, canHoldAll)).toStrictEqual(options)
  })

  it('draws every box at its widest where there is room for it', () => {
    const frame = arrangePlex(build('A node', COUNTS), {
      options: { viewport: { width: 2000, height: 1400 } },
    })
    expect(getChildWidth(frame)).toBe(nodeSize.width)
  })

  it('leaves a plex with no window alone', () => {
    expect(measureCrowding(DEFAULT_OPTIONS, canHoldAll)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('packs nothing at a squeeze of one', () => {
    const options = {
      ...DEFAULT_OPTIONS,
      squeeze: 1,
      viewport: { width: 440, height: 1000 },
    }
    expect(measureCrowding(options, canHoldAll)).toStrictEqual(options)
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
    expect(closest.maxPerCluster).toBe(DEFAULT_OPTIONS.maxPerCluster)
  })
})

describe('the loosest packing that fits everything', () => {
  const STEPS = 2 ** 7

  const getTightness = (packed: typeof DEFAULT_OPTIONS) =>
    (packed.nodeSize.width - minWidth) / (nodeSize.width - minWidth)

  it.each([
    { width: 440, height: 1000 },
    { width: 500, height: 1000 },
    { width: 600, height: 1000 },
  ])('$width×$height is found to a 128th, and one step looser leaves a note out', (viewport) => {
    const options = { ...DEFAULT_OPTIONS, viewport }
    const packed = measureCrowding(options, canHoldAll)
    const tightness = getTightness(packed)

    expect(canHoldAll(packed)).toBe(true)
    expect(tightness).toBeGreaterThanOrEqual(0)
    expect(tightness).toBeLessThan(1)
    expect(tightness * STEPS).toBeCloseTo(Math.round(tightness * STEPS))
    expect(canHoldAll(packOptions(options, tightness + 1 / STEPS))).toBe(false)
  })

  it('packs a window that is crowded at the settings tighter than the settings', () => {
    const options = { ...DEFAULT_OPTIONS, viewport: { width: 500, height: 1000 } }
    expect(getTightness(measureCrowding(options, canHoldAll))).toBeGreaterThan(0)
  })
})

describe('what no packing can fit', () => {
  it('is still reported and not drawn past the edge', () => {
    const viewport = { width: 400, height: 400 }
    const crowd = build('A node', { parent: 9, child: 40, jump: 9, sibling: 9 })
    const frame = arrangePlex(crowd, { options: { viewport } })

    expect(Object.keys(frame.overflow).length).toBeGreaterThan(0)
    expect(findEscapedNodes(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })
})
