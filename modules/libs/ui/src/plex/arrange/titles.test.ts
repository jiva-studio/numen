/** Where along its line a title comes to rest. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { MIDDLE, routeEdges, routingFor } from './routing'
import { settleTitles } from './titles'
import { build } from '../fixtures/build'
import {
  headingOf,
  lengthOf,
  rulerOf,
  type PlacedEdge,
  type PlacedNode,
  type PlexNeighbourhood,
} from '../model'

/** A letter of a fixed width, so what a title needs is a matter of counting. */
const perLetter = (width: number) => (label: string) => width * [...label].length

const measureLabel = perLetter(6)

/** How deep a line of label type stands, as the window would have measured it. */
const labelDepth = 12

/** What every arrangement here measures its titles with. */
const measured = { measureLabel, labelDepth }

const VIEWPORT = { width: 1200, height: 800 }

interface Box {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

const meets = (one: Box, other: Box): boolean =>
  one.minX < other.maxX &&
  other.minX < one.maxX &&
  one.minY < other.maxY &&
  other.minY < one.maxY

/**
 * Where a title's words lie in the picture: the run of them along the line,
 * with half a line of type standing either side of the line.
 */
function boxOf(edge: PlacedEdge, width: (label: string) => number): Box {
  const arc = lengthOf(edge)
  const along = rulerOf(edge)
  const at = edge.heading === 'against' ? 1 - edge.wordsAt : edge.wordsAt
  const half = width(edge.words!) / 2 / arc
  const deep = labelDepth / 2

  const run = [0, 0.25, 0.5, 0.75, 1].map((step) => along(at - half + 2 * half * step))
  const box = { minX: Infinity, minY: Infinity, maxX: -Infinity, maxY: -Infinity }

  for (const [sample, point] of run.entries()) {
    const back = run[Math.max(sample - 1, 0)]!
    const on = run[Math.min(sample + 1, run.length - 1)]!
    const span = Math.hypot(on.x - back.x, on.y - back.y) || 1
    const acrossX = (Math.abs(on.y - back.y) / span) * deep
    const acrossY = (Math.abs(on.x - back.x) / span) * deep

    box.minX = Math.min(box.minX, point.x - acrossX)
    box.minY = Math.min(box.minY, point.y - acrossY)
    box.maxX = Math.max(box.maxX, point.x + acrossX)
    box.maxY = Math.max(box.maxY, point.y + acrossY)
  }
  return box
}

const boxAround = (node: PlacedNode): Box => ({
  minX: node.x - node.width / 2,
  minY: node.y - node.height / 2,
  maxX: node.x + node.width / 2,
  maxY: node.y + node.height / 2,
})

describe('a title finds room on its line', () => {
  it('keeps the middle of a line with nothing near it', () => {
    const alone: PlexNeighbourhood = {
      nodes: [
        { id: 'focus', title: 'Here', seat: 'focus' },
        { id: 'aside', title: 'Aside', seat: 'jump' },
      ],
      edges: [{ from: 'aside', to: 'focus', label: 'see also' }],
    }
    const frame = arrangePlex(alone, measured)

    expect(frame.edges[0]!.wordsAt).toBe(0.5)
  })

  it('stands every title of a fan clear of the others', () => {
    const fan = build('A node', { child: 5 })
    const frame = arrangePlex(fan, { options: { viewport: VIEWPORT }, ...measured })
    const boxes = frame.edges.map((edge) => boxOf(edge, measureLabel))

    expect(boxes.length).toBe(5)

    const piled: string[] = []
    for (let one = 0; one < boxes.length; one += 1) {
      for (let other = one + 1; other < boxes.length; other += 1) {
        if (meets(boxes[one]!, boxes[other]!)) piled.push(`${one} x ${other}`)
      }
    }
    expect(piled).toStrictEqual([])
  })

  it('slides the titles that are in the way, and leaves the rest at the middle', () => {
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const moved = frame.edges.filter((edge) => edge.wordsAt !== 0.5)

    expect(moved.length).toBeGreaterThan(0)
    expect(moved.length).toBeLessThan(frame.edges.length)
  })

  it('keeps the middle of a line with nowhere clear along it', () => {
    const pair: PlexNeighbourhood = {
      nodes: [
        { id: 'focus', title: 'Here', seat: 'focus' },
        { id: 'aside', title: 'Aside', seat: 'jump' },
      ],
      edges: [{ from: 'aside', to: 'focus', label: 'see also' }],
    }
    const placed = arrangePlex(pair, measured)
    const byId = new Map(placed.nodes.map((node) => [node.id, node]))
    const routing = routingFor(DEFAULT_OPTIONS, measureLabel, labelDepth)
    const routed = routeEdges(pair.edges, byId, routing)

    // A box over the whole picture, so no place on the line is clear of it.
    const blanket: PlacedNode = {
      id: 'blanket',
      title: 'Blanket',
      seat: 'sibling',
      x: 0,
      y: 0,
      width: 2000,
      height: 2000,
      order: 0,
      opacity: 1,
    }
    const settled = settleTitles(routed, [...placed.nodes, blanket], routing)

    expect(settled[0]!.wordsAt).toBe(0.5)
  })

  it('gives the same neighbourhood the same offsets twice over', () => {
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const once = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const again = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })

    expect(again.edges.map((edge) => edge.wordsAt)).toStrictEqual(
      once.edges.map((edge) => edge.wordsAt),
    )
  })

  it('sets a title where the arrangement says, and not where the curve runs', () => {
    // A title read against its curve is set from the far end, so the fraction
    // it carries is a fraction of the line the words are read along.
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })

    for (const edge of frame.edges) {
      expect(edge.wordsAt, `${edge.from}->${edge.to}`).toBeGreaterThan(0)
      expect(edge.wordsAt, `${edge.from}->${edge.to}`).toBeLessThan(1)
    }
  })

  it('keeps a title clear of the boxes wherever the line lets it', () => {
    const crowd = build('A node', { parent: 2, jump: 3 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const boxes = frame.nodes.map(boxAround)

    const over = frame.edges.filter((edge) =>
      boxes.some((box) => meets(box, boxOf(edge, measureLabel))),
    )
    expect(over).toStrictEqual([])
  })

  it('reads the words the way the line runs where they end up', () => {
    // Two nodes of one row, joined to each other. The gap between their gates
    // is under the reach an edge holds, so each control point is thrown past
    // the far gate and the little line doubles back twice over: at the middle
    // it runs against the way it sets off and the way it arrives.
    const row: PlexNeighbourhood = {
      nodes: [
        { id: 'focus', title: 'Here', seat: 'focus' },
        { id: 'first', title: 'First', seat: 'parent' },
        { id: 'second', title: 'Second', seat: 'parent' },
      ],
      edges: [{ from: 'first', to: 'second', label: 'and' }],
    }

    const placed = arrangePlex(row)
    const byId = new Map(placed.nodes.map((node) => [node.id, node]))
    const bare = routeEdges(row.edges, byId, routingFor(DEFAULT_OPTIONS))[0]!
    expect(headingOf(bare, MIDDLE)).toBe('against')

    // Words taking three fifths of that line, so the title has one step to
    // either side of the middle and no further.
    const arc = lengthOf(bare)
    const routing = routingFor(DEFAULT_OPTIONS, () => 0.6 * arc, labelDepth)
    const routed = routeEdges(row.edges, byId, routing)

    // A box over the last quarter of the line, so the words slide back towards
    // its start, which is past the turn the middle stands on.
    const past = rulerOf(routed[0]!)(0.75)
    const covering: PlacedNode = {
      id: 'covering',
      title: 'Covering',
      seat: 'sibling',
      x: past.x + 100,
      y: past.y,
      width: 200,
      height: 200,
      order: 0,
      opacity: 1,
    }

    const settled = settleTitles(routed, [...placed.nodes, covering], routing)[0]!
    expect(settled.wordsAt).not.toBe(MIDDLE)
    expect(settled.heading).toBe('along')
    expect(headingOf(settled, settled.wordsAt)).toBe('along')
  })

  it('leaves every title at the middle where nothing measured the words', () => {
    const crowd = build('A node', { child: 5 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT } })

    expect(frame.edges.map((edge) => edge.wordsAt)).toStrictEqual(
      frame.edges.map(() => 0.5),
    )
  })
})
