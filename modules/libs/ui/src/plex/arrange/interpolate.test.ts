/** The movement, tested without waiting for it. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { interpolatePlex } from './interpolate'
import { easeOut } from './math'
import type { PlexFrame, PlexNeighbourhood } from '../model'

/** Focus on `focus`, with two children and one parent. */
const before: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', label: 'Start', role: 'focus' },
    { id: 'up', label: 'Above', role: 'parent' },
    { id: 'a', label: 'A', role: 'child' },
    { id: 'b', label: 'B', role: 'child' },
  ],
  edges: [
    { from: 'up', to: 'focus' },
    { from: 'focus', to: 'a' },
    { from: 'focus', to: 'b' },
  ],
}

/**
 * `a` was chosen. Three children, not two: with two on each side every new
 * seat has a twin at the old coordinates and a distance test compares 0 to 0.
 */
const after: PlexNeighbourhood = {
  nodes: [
    { id: 'a', label: 'A', role: 'focus' },
    { id: 'focus', label: 'Start', role: 'parent' },
    { id: 'a1', label: 'A one', role: 'child' },
    { id: 'a2', label: 'A two', role: 'child' },
    { id: 'a3', label: 'A three', role: 'child' },
  ],
  edges: [
    { from: 'focus', to: 'a' },
    { from: 'a', to: 'a1' },
    { from: 'a', to: 'a2' },
    { from: 'a', to: 'a3' },
  ],
}

const from = arrangePlex(before)
const to = arrangePlex(after)

const node = (layout: PlexFrame, id: string) => {
  const found = layout.nodes.find((n) => n.id === id)
  if (!found) throw new Error(`no node ${id} in this frame`)
  return found
}

describe('the ends of the movement', () => {
  it('is the old arrangement at nothing', () => {
    expect(interpolatePlex(from, to, 0)).toBe(from)
  })

  it('is the new arrangement at all of it', () => {
    expect(interpolatePlex(from, to, 1)).toBe(to)
  })

  it('clamps anything outside', () => {
    expect(interpolatePlex(from, to, -3)).toBe(from)
    expect(interpolatePlex(from, to, 4)).toBe(to)
  })
})

describe('a node that is in both pictures travels between its seats', () => {
  it('carries the chosen node from where it was to the middle', () => {
    const start = node(from, 'a')
    const half = node(interpolatePlex(from, to, 0.5), 'a')
    const end = node(to, 'a')

    expect(end.x).toBe(0)
    expect(end.y).toBe(0)
    expect(Math.abs(half.y)).toBeLessThan(Math.abs(start.y))
    expect(Math.abs(half.y)).toBeGreaterThan(0)
    expect(half.y).toBe((start.y + end.y) / 2)
  })

  it('carries the old focus out to its new seat', () => {
    const start = node(from, 'focus')
    const half = node(interpolatePlex(from, to, 0.5), 'focus')
    const end = node(to, 'focus')

    expect(start.y).toBe(0)
    expect(end.y).toBeLessThan(0) // it is a parent now
    expect(half.y).toBe((start.y + end.y) / 2)
  })

  it('grows and shrinks the box as the role changes', () => {
    const half = node(interpolatePlex(from, to, 0.5), 'a')
    expect(half.width).toBeGreaterThan(node(from, 'a').width)
    expect(half.width).toBeLessThan(node(to, 'a').width)
  })

  it('never doubles back', () => {
    let previous = node(from, 'a').y
    for (let t = 0.05; t <= 1; t += 0.05) {
      const y = node(interpolatePlex(from, to, t), 'a').y
      expect(Math.abs(y)).toBeLessThanOrEqual(Math.abs(previous) + 1e-9)
      previous = y
    }
  })
})

describe('a node that is only in one picture', () => {
  it('unfolds a new node out of where the chosen one started', () => {
    const source = node(from, 'a')
    const early = node(interpolatePlex(from, to, 0.1), 'a1')
    const end = node(to, 'a1')

    // A tenth in, still near where the new focus set off.
    expect(Math.hypot(early.x - source.x, early.y - source.y)).toBeLessThan(
      Math.hypot(early.x - end.x, early.y - end.y),
    )
  })

  it('holds a new node invisible until the picture has begun to settle', () => {
    expect(node(interpolatePlex(from, to, 0.1), 'a1').opacity).toBe(0)
    expect(node(interpolatePlex(from, to, 0.9), 'a1').opacity).toBeGreaterThan(0.5)
    expect(node(to, 'a1').opacity).toBe(1)
  })

  it('keeps a departing node where it was and fades it out early', () => {
    const early = node(interpolatePlex(from, to, 0.1), 'b')
    const late = node(interpolatePlex(from, to, 0.6), 'b')
    const original = node(from, 'b')

    expect(early.x).toBe(original.x)
    expect(early.y).toBe(original.y)
    expect(early.opacity).toBeGreaterThan(0)
    expect(late.opacity).toBe(0)
  })

  it('lets go of a departing node once the movement is over', () => {
    expect(to.nodes.find((n) => n.id === 'b')).toBeUndefined()
    expect(interpolatePlex(from, to, 0.5).nodes.some((n) => n.id === 'b')).toBe(true)
  })
})

describe('the edges follow the boxes', () => {
  it('meets the border of a box that is halfway to its new seat', () => {
    const frame = interpolatePlex(from, to, 0.4)
    const byId = new Map(frame.nodes.map((n) => [n.id, n]))

    for (const edge of frame.edges) {
      const target = byId.get(edge.to)!
      const dx = Math.abs(edge.toPoint.x - target.x)
      const dy = Math.abs(edge.toPoint.y - target.y)
      const onBorder =
        Math.abs(dx - target.width / 2) < 1e-9 || Math.abs(dy - target.height / 2) < 1e-9
      expect(onBorder, `edge to ${edge.to} left its box behind`).toBe(true)
    }
  })

  it('keeps an edge that both pictures have at full strength throughout', () => {
    const frame = interpolatePlex(from, to, 0.5)
    const kept = frame.edges.find((e) => e.from === 'focus' && e.to === 'a')
    expect(kept?.opacity).toBe(1)
  })

  it('fades an edge in or out with the node it belongs to', () => {
    const early = interpolatePlex(from, to, 0.1)
    expect(early.edges.find((e) => e.to === 'a1')?.opacity).toBe(0)
    expect(early.edges.find((e) => e.to === 'b')?.opacity).toBeGreaterThan(0)

    const late = interpolatePlex(from, to, 0.8)
    expect(late.edges.find((e) => e.to === 'a1')?.opacity).toBeGreaterThan(0)
    // Out of sight well before it is out of the frame.
    expect(late.edges.find((e) => e.to === 'b')?.opacity).toBe(0)
    expect(to.edges.find((e) => e.to === 'b')).toBeUndefined()
  })
})

describe('the frame the viewport is fitted to', () => {
  it('travels from one extent to the other rather than jumping', () => {
    const half = interpolatePlex(from, to, 0.5).extent
    expect(half.minX).toBe((from.extent.minX + to.extent.minX) / 2)
    expect(half.maxY).toBe((from.extent.maxY + to.extent.maxY) / 2)
  })
})

describe('a frame is decided by its inputs and nothing else', () => {
  it('gives the same numbers for the same moment', () => {
    expect(interpolatePlex(from, to, 0.37)).toStrictEqual(
      interpolatePlex(from, to, 0.37),
    )
  })

  it('reports what the new arrangement left out, not the old one', () => {
    expect(interpolatePlex(from, to, 0.5).overflow).toStrictEqual(to.overflow)
  })
})

describe('the shape of the movement is a setting', () => {
  it('lets a caller say when an arrival starts to show', () => {
    const early = interpolatePlex(from, to, 0.2, { motion: { arriveAfter: 0 } })
    const late = interpolatePlex(from, to, 0.2, { motion: { arriveAfter: 0.9 } })

    expect(node(early, 'a1').opacity).toBeGreaterThan(0)
    expect(node(late, 'a1').opacity).toBe(0)
  })

  it('lets a caller say when a departure has finished going', () => {
    const brief = interpolatePlex(from, to, 0.3, { motion: { leaveBefore: 0.2 } })
    const lingering = interpolatePlex(from, to, 0.3, { motion: { leaveBefore: 0.9 } })

    expect(node(brief, 'b').opacity).toBe(0)
    expect(node(lingering, 'b').opacity).toBeGreaterThan(0)
  })
})

describe('a line whose ends swap', () => {
  it('stays drawn all the way across', () => {
    // A jump is the same relationship from either end, so travelling along one
    // turns `there → here` into `here → there`. Read as two lines, the one the
    // reader is following is the one that fades out.
    const here: PlexNeighbourhood = {
      nodes: [
        { id: 'here', label: 'Here', role: 'focus' },
        { id: 'across', label: 'Across', role: 'jump' },
      ],
      edges: [{ from: 'across', to: 'here' }],
    }
    const across: PlexNeighbourhood = {
      nodes: [
        { id: 'across', label: 'Across', role: 'focus' },
        { id: 'here', label: 'Here', role: 'jump' },
      ],
      edges: [{ from: 'here', to: 'across' }],
    }

    const moving = interpolatePlex(arrangePlex(here), arrangePlex(across), 0.5)
    expect(moving.edges).toHaveLength(1)
    expect(moving.edges[0]?.opacity).toBe(1)
  })
})

describe('the easing', () => {
  it('starts and ends where it should', () => {
    expect(easeOut(0)).toBe(0)
    expect(easeOut(1)).toBe(1)
  })

  it('covers most of the distance early, so the end is the slow part', () => {
    expect(easeOut(0.5)).toBeGreaterThan(0.75)
    expect(easeOut(0.9)).toBeGreaterThan(0.99)
  })

  it('never goes backwards', () => {
    for (let t = 0; t < 1; t += 0.05) {
      expect(easeOut(t + 0.05)).toBeGreaterThanOrEqual(easeOut(t))
    }
  })
})
