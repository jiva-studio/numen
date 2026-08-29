/** Packing the picture closer, before anything is given up. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { crowdingFor, packed } from './crowding'
import { DEFAULT_OPTIONS } from './options'
import { build } from '../fixtures/build'
import type { PlexFrame } from '../model'

const { nodeSize, minWidth, gap, squeeze, margin } = DEFAULT_OPTIONS

/** The neighbourhood a narrow window used to give seats up on. */
const COUNTS = { parent: 1, child: 6, jump: 1, sibling: 2 }

const escapes = (frame: PlexFrame, viewport: { width: number; height: number }) =>
  frame.nodes.filter(
    (node) =>
      Math.abs(node.x) + node.width / 2 > viewport.width / 2 - margin ||
      Math.abs(node.y) + node.height / 2 > viewport.height / 2 - margin,
  )

const boxOf = (frame: PlexFrame) => frame.nodes.find((node) => node.seat === 'child')!.width

describe('a box narrows and a gap closes before a seat is given up', () => {
  const NARROW = [
    { width: 440, height: 1000 },
    { width: 500, height: 1000 },
    { width: 600, height: 1000 },
    { width: 700, height: 1000 },
    { width: 900, height: 1000 },
  ]

  it.each(NARROW)('$width×$height seats every node', (viewport) => {
    const frame = arrangePlex(build('A node', COUNTS), { options: { viewport } })
    expect(frame.overflow).toStrictEqual({})
  })

  it.each(NARROW)('$width×$height keeps every node inside the window', (viewport) => {
    const frame = arrangePlex(build('A node', COUNTS), { options: { viewport } })
    expect(escapes(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })

  it('draws a narrower box the less room the window has', () => {
    const boxes = NARROW.map((viewport) =>
      boxOf(arrangePlex(build('A node', COUNTS), { options: { viewport } })),
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
    expect(crowdingFor(options, COUNTS)).toStrictEqual(options)
  })

  it('draws every box at its widest where there is room for it', () => {
    const frame = arrangePlex(build('A node', COUNTS), {
      options: { viewport: { width: 2000, height: 1400 } },
    })
    expect(boxOf(frame)).toBe(nodeSize.width)
  })

  it('leaves a plex with no window alone', () => {
    expect(crowdingFor(DEFAULT_OPTIONS, COUNTS)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('packs nothing at a squeeze of one', () => {
    const options = {
      ...DEFAULT_OPTIONS,
      squeeze: 1,
      viewport: { width: 440, height: 1000 },
    }
    expect(crowdingFor(options, COUNTS)).toStrictEqual(options)
  })
})

describe('how closely a plex is packed', () => {
  it('is its settings at one', () => {
    expect(packed(DEFAULT_OPTIONS, 1)).toStrictEqual(DEFAULT_OPTIONS)
  })

  it('is the narrowest box and the closest gap at nothing', () => {
    const closest = packed(DEFAULT_OPTIONS, 0)
    expect(closest.nodeSize.width).toBe(minWidth)
    expect(closest.gap).toBeCloseTo(gap * squeeze)
  })

  it('never draws a box narrower than the narrowest, whatever it is packed to', () => {
    for (const tightness of [0, 0.1, 0.5, 0.9, 1]) {
      expect(packed(DEFAULT_OPTIONS, tightness).nodeSize.width).toBeGreaterThanOrEqual(
        minWidth,
      )
    }
  })

  it('leaves the heights, the counts and the window where they were', () => {
    const closest = packed(DEFAULT_OPTIONS, 0)
    expect(closest.nodeSize.height).toBe(nodeSize.height)
    expect(closest.maxPerLine).toBe(DEFAULT_OPTIONS.maxPerLine)
    expect(closest.maxLines).toBe(DEFAULT_OPTIONS.maxLines)
  })
})

describe('what no packing can seat', () => {
  it('is still reported rather than drawn past the edge', () => {
    const viewport = { width: 400, height: 400 }
    const crowd = build('A node', { parent: 9, child: 40, jump: 9, sibling: 9 })
    const frame = arrangePlex(crowd, { options: { viewport } })

    expect(Object.keys(frame.overflow).length).toBeGreaterThan(0)
    expect(escapes(frame, viewport).map((node) => node.title)).toStrictEqual([])
  })
})
