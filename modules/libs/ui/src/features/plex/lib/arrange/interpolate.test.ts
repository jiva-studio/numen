/** The movement, tested without waiting for it. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { interpolatePlex } from './interpolate'
import { easeOut } from './math'
import { MIDDLE } from './edges/routing'
import { ARROW_LENGTH, getHeading, getLength, createPointOnCurve } from '../edge'
import type { PlexFrame } from '../frame'
import type { PlexNeighbourhood } from '../neighbourhood'

/** Focus on `focus`, with two children and one parent. */
const before: PlexNeighbourhood = {
  nodes: [
    { id: 'focus', title: 'Start', relation: 'focus' },
    { id: 'up', title: 'Above', relation: 'parent' },
    { id: 'a', title: 'A', relation: 'child' },
    { id: 'b', title: 'B', relation: 'child' },
  ],
  edges: [
    { from: 'up', to: 'focus' },
    { from: 'focus', to: 'a' },
    { from: 'focus', to: 'b' },
  ],
}

/**
 * `a` was chosen. Three children, not two: with two on each side every new
 * relation has a twin at the old coordinates and a distance test compares 0 to 0.
 */
const after: PlexNeighbourhood = {
  nodes: [
    { id: 'a', title: 'A', relation: 'focus' },
    { id: 'focus', title: 'Start', relation: 'parent' },
    { id: 'a1', title: 'A one', relation: 'child' },
    { id: 'a2', title: 'A two', relation: 'child' },
    { id: 'a3', title: 'A three', relation: 'child' },
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

describe('a node that is in both pictures travels between its relations', () => {
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

  it('carries the old focus out to its new relation', () => {
    const start = node(from, 'focus')
    const half = node(interpolatePlex(from, to, 0.5), 'focus')
    const end = node(to, 'focus')

    expect(start.y).toBe(0)
    expect(end.y).toBeLessThan(0) // it is a parent now
    expect(half.y).toBe((start.y + end.y) / 2)
  })

  it('grows and shrinks the box as the relation changes', () => {
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

  it('puts a new node halfway between the chosen node and its own relation', () => {
    const source = node(from, 'a')
    const end = node(to, 'a1')
    const middle = node(interpolatePlex(from, to, 0.5), 'a1')

    expect(source.x).not.toBe(0)
    expect(source.y).not.toBe(0)
    expect(middle.x).toBeCloseTo((source.x + end.x) / 2)
    expect(middle.y).toBeCloseTo((source.y + end.y) / 2)
  })

  it('finds the chosen node wherever it remains in the new picture', () => {
    const reordered: PlexFrame = { ...to, nodes: [...to.nodes].reverse() }
    const source = node(from, 'a')
    const end = node(to, 'a1')
    const middle = node(interpolatePlex(from, reordered, 0.5), 'a1')

    expect(middle.x).toBeCloseTo((source.x + end.x) / 2)
    expect(middle.y).toBeCloseTo((source.y + end.y) / 2)
  })

  it('unfolds a new node out of the middle when the chosen node was not in the old picture', () => {
    const stranger = arrangePlex({
      nodes: [
        { id: 'elsewhere', title: 'Elsewhere', relation: 'focus' },
        { id: 'e1', title: 'E one', relation: 'child' },
      ],
      edges: [],
    })
    const arrival = node(stranger, 'e1')
    const middle = node(interpolatePlex(from, stranger, 0.5), 'e1')

    expect(middle.x).toBeCloseTo(arrival.x / 2)
    expect(middle.y).toBeCloseTo(arrival.y / 2)
  })

  it('shows every node of either picture once', () => {
    const ids = interpolatePlex(from, to, 0.5).nodes.map((n) => n.id)
    expect([...ids].sort()).toStrictEqual(['a', 'a1', 'a2', 'a3', 'b', 'focus', 'up'])
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
  it('meets the border of a box that is halfway to its new relation', () => {
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
    const retainedEdge = frame.edges.find((e) => e.from === 'focus' && e.to === 'a')
    expect(retainedEdge?.opacity).toBe(1)
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

/**
 * A title belongs to a line that has settled. The words it was cut to and the
 * way round they are read come from the arrangements being moved between, and
 * a curve in flight decides neither.
 */
describe('the title a line carries while the picture moves', () => {
  const getEdgeKey = (edge: { from: string; to: string }) => `${edge.from}->${edge.to}`

  it('holds the way round the words are read', () => {
    const resolved = new Map(to.edges.map((edge) => [getEdgeKey(edge), edge.heading]))

    // A child promoted to the focus swings its edge across the page, and
    // halfway over it runs the other way round from the way it ends.
    const swung = interpolatePlex(from, to, 0.5).edges.find((e) => e.to === 'a')!
    expect(swung.heading).toBe('along')
    expect(getHeading(swung, MIDDLE)).toBe('against')

    for (let t = 0.05; t < 1; t += 0.05) {
      for (const edge of interpolatePlex(from, to, t).edges) {
        const heading = resolved.get(getEdgeKey(edge))
        if (heading === undefined) continue
        expect(edge.heading, `${getEdgeKey(edge)} at ${t.toFixed(2)}`).toBe(heading)
      }
    }
  })

  it('holds the words a settled line was cut to', () => {
    // Wide letters and a long label, so the cut is well short of the whole.
    const measureLabelWidth = (label: string) => 30 * [...label].length
    const arrangeWithLabels = (neighbourhood: PlexNeighbourhood) =>
      arrangePlex(
        {
          ...neighbourhood,
          edges: neighbourhood.edges.map((edge) => ({
            ...edge,
            label: 'the scene in the assembly',
          })),
        },
        { measureLabel: measureLabelWidth },
      )

    const start = arrangeWithLabels(before)
    const end = arrangeWithLabels(after)
    const trimmedWords = end.edges.find((e) => e.to === 'a')!.words!
    expect(trimmedWords.endsWith('…')).toBe(true)

    for (let t = 0.05; t < 1; t += 0.05) {
      const edge = interpolatePlex(start, end, t).edges.find((e) => e.to === 'a')!
      expect(edge.words, `at ${t.toFixed(2)}`).toBe(trimmedWords)
    }
  })

  it('holds where along the line the words sit', () => {
    const measureLabelWidth = (label: string) => 6 * [...label].length

    // A window that wraps the fan, at the settings: what this is about is a
    // title moving off the middle, not how much room the window has to spare.
    const arrangeWithTitles = (neighbourhood: PlexNeighbourhood) =>
      arrangePlex(
        {
          ...neighbourhood,
          edges: neighbourhood.edges.map((edge) => ({ ...edge, label: 'contains' })),
        },
        {
          measureLabel: measureLabelWidth,
          options: { viewport: { width: 900, height: 800 }, spread: 1 },
        },
      )

    // A fan wide enough that its titles cannot all keep the middle.
    const fanned: PlexNeighbourhood = {
      nodes: [
        ...after.nodes,
        { id: 'a4', title: 'A four', relation: 'child' },
        { id: 'a5', title: 'A five', relation: 'child' },
        { id: 'a6', title: 'A six', relation: 'child' },
      ],
      edges: [
        ...after.edges,
        { from: 'a', to: 'a4' },
        { from: 'a', to: 'a5' },
        { from: 'a', to: 'a6' },
      ],
    }

    const start = arrangeWithTitles(before)
    const end = arrangeWithTitles(fanned)
    const resolved = new Map(end.edges.map((edge) => [getEdgeKey(edge), edge.wordsAt]))

    // Some of the lines out of one node had to move their titles to stand
    // clear, and those are the ones a movement could set crawling.
    expect(end.edges.some((edge) => edge.wordsAt !== 0.5)).toBe(true)

    for (let t = 0.05; t < 1; t += 0.05) {
      for (const edge of interpolatePlex(start, end, t).edges) {
        const offset = resolved.get(getEdgeKey(edge))
        if (offset === undefined) continue
        expect(edge.wordsAt, `${getEdgeKey(edge)} at ${t.toFixed(2)}`).toBe(offset)
      }
    }
  })

  it('holds the title of the arrangement it is leaving for a line only there', () => {
    const marked: PlexFrame = {
      ...from,
      edges: from.edges.map((edge) =>
        edge.to === 'b' ? { ...edge, heading: 'against' as const, words: 'went that way…' } : edge,
      ),
    }
    const going = interpolatePlex(marked, to, 0.5).edges.find((e) => e.to === 'b')!

    expect(going.heading).toBe('against')
    expect(going.words).toBe('went that way…')
    expect(getHeading(going, MIDDLE)).toBe('along')
  })
})

describe('the arrow a line carries while the picture moves', () => {
  const markArrows = (neighbourhood: PlexNeighbourhood): PlexNeighbourhood => ({
    ...neighbourhood,
    edges: neighbourhood.edges.map((edge) =>
      edge.to === 'a' ? { ...edge, arrow: 'to' as const } : edge,
    ),
  })

  const start = arrangePlex(markArrows(before))
  const end = arrangePlex(markArrows(after))

  it('keeps it on the end of the line, wherever the line has got to', () => {
    for (let t = 0.05; t < 1; t += 0.05) {
      const edge = interpolatePlex(start, end, t).edges.find((e) => e.to === 'a')!
      expect(edge.arrowhead, `at ${t.toFixed(2)}`).toBeDefined()
      expect(edge.arrowhead!.at.x).toBe(edge.toPoint.x)
      expect(edge.arrowhead!.at.y).toBe(edge.toPoint.y)
    }
  })

  it('keeps it aimed along the piece of curve it covers, wherever the line has got to', () => {
    for (let t = 0.05; t < 1; t += 0.05) {
      const edge = interpolatePlex(start, end, t).edges.find((e) => e.to === 'a')!
      const behindPoint = createPointOnCurve(edge)(1 - ARROW_LENGTH / getLength(edge))
      const chord =
        (Math.atan2(edge.toPoint.y - behindPoint.y, edge.toPoint.x - behindPoint.x) * 180) / Math.PI
      expect(edge.arrowhead!.angle, `at ${t.toFixed(2)}`).toBeCloseTo(chord, 1)
    }
  })
})

describe('the frame the viewport is fitted to', () => {
  it('travels from one extent to the other and not jumping', () => {
    const half = interpolatePlex(from, to, 0.5).extent
    expect(half.minX).toBe((from.extent.minX + to.extent.minX) / 2)
    expect(half.maxY).toBe((from.extent.maxY + to.extent.maxY) / 2)
  })
})

describe('a frame is decided by its inputs and nothing else', () => {
  it('gives the same numbers for the same moment', () => {
    expect(interpolatePlex(from, to, 0.37)).toStrictEqual(interpolatePlex(from, to, 0.37))
  })

  it('reports what the new arrangement left out, not the old one', () => {
    expect(interpolatePlex(from, to, 0.5).overflow).toStrictEqual(to.overflow)
  })
})

describe('the shape of the movement is a setting', () => {
  it('lets a caller say when an arrival starts to show', () => {
    const early = interpolatePlex(from, to, { progress: 0.2, motion: { arriveAfter: 0 } })
    const late = interpolatePlex(from, to, { progress: 0.2, motion: { arriveAfter: 0.9 } })

    expect(node(early, 'a1').opacity).toBeGreaterThan(0)
    expect(node(late, 'a1').opacity).toBe(0)
  })

  it('lets a caller say when a departure has finished going', () => {
    const brief = interpolatePlex(from, to, { progress: 0.3, motion: { leaveBefore: 0.2 } })
    const lingering = interpolatePlex(from, to, { progress: 0.3, motion: { leaveBefore: 0.9 } })

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
        { id: 'here', title: 'Here', relation: 'focus' },
        { id: 'across', title: 'Across', relation: 'jump' },
      ],
      edges: [{ from: 'across', to: 'here' }],
    }
    const acrossRoute: PlexNeighbourhood = {
      nodes: [
        { id: 'across', title: 'Across', relation: 'focus' },
        { id: 'here', title: 'Here', relation: 'jump' },
      ],
      edges: [{ from: 'here', to: 'across' }],
    }

    const interpolated = interpolatePlex(arrangePlex(here), arrangePlex(acrossRoute), 0.5)
    expect(interpolated.edges).toHaveLength(1)
    expect(interpolated.edges[0]?.opacity).toBe(1)
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
