/**
 * A note keeps room for the names over and under its pill, and the arrangement keeps every
 * other note out of that room.
 */
import { describe, expect, it } from 'vitest'
import { arrangePlex, type ArrangeInput } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { graphs } from '../../fixtures/graphs'
import { build } from '../../fixtures/build'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { PlexGraph, PlexNode } from '../../../types'

interface Room {
  top: number
  bottom: number
  width: number
}
type Rooms = (node: PlexNode, index?: number) => Room

const NONE: Room = { top: 0, bottom: 0, width: 0 }
const { margin } = DEFAULT_OPTIONS
const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

const arrange = (
  graph: PlexGraph,
  viewport?: { width: number; height: number },
  room?: Rooms,
): PlexFrame => {
  const input: ArrangeInput & { room?: Rooms } = {
    measure: byTitle,
    ...(viewport ? { options: { viewport } } : {}),
    ...(room ? { room } : {}),
  }
  return arrangePlex(graph, input)
}

const namesUnder: Rooms = (node) =>
  node.relation === 'focus' ? NONE : { top: 0, bottom: 56, width: 90 }

const namesOverParents: Rooms = (node) =>
  node.relation === 'parent' ? { top: 60, bottom: 0, width: 90 } : NONE

const namesAround: Rooms = (node, index = 0) =>
  node.relation === 'focus'
    ? NONE
    : { top: index % 2 === 0 ? 0 : 28, bottom: index % 3 === 0 ? 0 : 42, width: 80 }

const roomsByIndex = (graph: PlexGraph): Rooms => {
  const order = new Map(graph.nodes.map((node, index) => [node.id, index]))
  return (node) => namesAround(node, order.get(node.id))
}

interface Footprint {
  id: string
  minX: number
  maxX: number
  minY: number
  maxY: number
}

const getFootprint = (node: PlacedNode, room: Room): Footprint => {
  const width = Math.max(node.width, room.width)
  return {
    id: node.id,
    minX: node.x - width / 2,
    maxX: node.x + width / 2,
    minY: node.y - node.height / 2 - room.top,
    maxY: node.y + node.height / 2 + room.bottom,
  }
}

const isOverlapping = (a: Footprint, b: Footprint): boolean =>
  a.minX < b.maxX - 1e-6 &&
  b.minX < a.maxX - 1e-6 &&
  a.minY < b.maxY - 1e-6 &&
  b.minY < a.maxY - 1e-6

const getFootprints = (graph: PlexGraph, frame: PlexFrame, rooms: Rooms): Footprint[] => {
  const nodes = new Map(graph.nodes.map((node) => [node.id, node]))
  return frame.nodes.map((placed) => getFootprint(placed, rooms(nodes.get(placed.id)!)))
}

const getFootprintOverlaps = (footprints: readonly Footprint[]): string[] =>
  footprints.flatMap((a, index) =>
    footprints
      .slice(index + 1)
      .filter((b) => isOverlapping(a, b))
      .map((b) => `${a.id} / ${b.id}`),
  )

describe('a frame with no room reserved', () => {
  it.each(Object.entries(graphs))(
    'is the frame of %s as it was, with no room and with a room of zeros',
    (_, graph) => {
      const window = { width: 1200, height: 800 }

      expect(arrange(graph, window, () => NONE)).toStrictEqual(arrange(graph, window))
      expect(arrange(graph, undefined, () => NONE)).toStrictEqual(arrange(graph))
    },
  )
})

describe('names under the notes of a relation', () => {
  const window = { width: 2400, height: 1400 }

  it('keep the footprints of notes from overlapping, the names included', () => {
    const graph = build('A note', { child: 9 })

    const frame = arrange(graph, window, namesUnder)

    expect(frame.nodes).toHaveLength(10)
    expect(getFootprintOverlaps(getFootprints(graph, frame, namesUnder))).toEqual([])
  })

  it('put two children in one column at least a pill and its names apart', () => {
    const graph = build('A note', { child: 9 })
    const pillHeight = arrange(graph, window).nodes[1]!.height

    const children = arrange(graph, window, namesUnder).nodes.filter((n) => n.relation === 'child')

    for (const [index, a] of children.entries()) {
      for (const b of children.slice(index + 1)) {
        const isSharingColumn = Math.abs(a.x - b.x) < (a.width + b.width) / 2
        if (isSharingColumn) {
          expect(Math.abs(a.y - b.y)).toBeGreaterThanOrEqual(pillHeight + 56 - 1e-6)
        }
      }
    }
  })
})

describe('names over the parents', () => {
  const window = { width: 900, height: 700 }

  it('keep every other pill out of the strip over a parent', () => {
    const graph = build('A note', { parent: 12 })

    const frame = arrange(graph, window, namesOverParents)

    expect(getFootprintOverlaps(getFootprints(graph, frame, namesOverParents))).toEqual([])
    for (const parent of frame.nodes.filter((node) => node.relation === 'parent')) {
      const strip = {
        id: `${parent.id}/names`,
        minX: parent.x - 45,
        maxX: parent.x + 45,
        minY: parent.y - parent.height / 2 - 60,
        maxY: parent.y - parent.height / 2,
      }
      const covering = frame.nodes
        .filter((other) => other.id !== parent.id)
        .filter((other) => isOverlapping(strip, getFootprint(other, NONE)))
      expect(covering.map((node) => node.id)).toEqual([])
    }
  })

  it('leave the pill of a parent the size it is without names', () => {
    const graph = build('A note', { parent: 12 })
    const bare = arrange(graph, window)

    const frame = arrange(graph, window, namesOverParents)

    for (const node of frame.nodes) {
      const same = bare.nodes.find((one) => one.id === node.id)
      if (same) expect([node.width, node.height]).toEqual([same.width, same.height])
    }
  })
})

describe('names in every fixture', () => {
  const windows = [
    { width: 2400, height: 1400 },
    { width: 1200, height: 800 },
    { width: 700, height: 600 },
  ]

  it.each(Object.entries(graphs))('never have a pill of another note on them in %s', (_, graph) => {
    const rooms = roomsByIndex(graph)
    for (const window of windows) {
      const frame = arrange(graph, window, rooms)
      expect(getFootprintOverlaps(getFootprints(graph, frame, rooms))).toEqual([])
    }
  })

  it.each(Object.entries(graphs))('give the same frame twice in %s', (_, graph) => {
    const rooms = roomsByIndex(graph)
    for (const window of windows) {
      expect(arrange(graph, window, rooms)).toStrictEqual(arrange(graph, window, rooms))
    }
  })
})

describe('names counted against the window', () => {
  const window = { width: 700, height: 600 }
  const graph = build('A note', { parent: 3, child: 8, jump: 3, sibling: 3 })

  it('show no more notes than the same graph without names, and account for every note', () => {
    const bare = arrange(graph, window)
    const named = arrange(graph, window, namesUnder)

    const overflow = (frame: PlexFrame) =>
      Object.values(frame.overflow).reduce((sum, count) => sum + (count ?? 0), 0)
    expect(named.nodes.length).toBeLessThanOrEqual(bare.nodes.length)
    expect(named.nodes.length + overflow(named)).toBe(graph.nodes.length)
    expect(bare.nodes.length + overflow(bare)).toBe(graph.nodes.length)
  })

  it('keep every footprint of a shown note inside the window less the margin', () => {
    const frame = arrange(graph, window, namesUnder)

    for (const one of getFootprints(graph, frame, namesUnder)) {
      expect(one.minX).toBeGreaterThanOrEqual(-window.width / 2 + margin - 1e-6)
      expect(one.maxX).toBeLessThanOrEqual(window.width / 2 - margin + 1e-6)
      expect(one.minY).toBeGreaterThanOrEqual(-window.height / 2 + margin - 1e-6)
      expect(one.maxY).toBeLessThanOrEqual(window.height / 2 - margin + 1e-6)
    }
  })
})

describe('a pill with names', () => {
  it('is the pill it is without, and its lines end at it', () => {
    const graph = build('A note', { parent: 2, child: 6 })
    const window = { width: 2400, height: 1400 }
    const bare = arrange(graph, window)

    const frame = arrange(graph, window, namesUnder)

    for (const node of frame.nodes) {
      const same = bare.nodes.find((one) => one.id === node.id)!
      expect([node.width, node.height]).toEqual([same.width, same.height])
    }
    const byId = new Map(frame.nodes.map((node) => [node.id, node]))
    const distance = (point: { x: number; y: number }, node: PlacedNode) =>
      Math.hypot(
        Math.max(Math.abs(point.x - node.x) - node.width / 2, 0),
        Math.max(Math.abs(point.y - node.y) - node.height / 2, 0),
      )
    for (const edge of frame.edges) {
      expect(distance(edge.fromPoint, byId.get(edge.from)!)).toBeLessThanOrEqual(2)
      expect(distance(edge.toPoint, byId.get(edge.to)!)).toBeLessThanOrEqual(2)
    }
  })
})

describe('a relation where half the notes have names', () => {
  const window = { width: 2400, height: 1400 }
  const graph = build('A note', { child: 8 })
  const half: Rooms = (node) =>
    node.relation === 'child' && Number(node.id.split('-').at(-1)) % 2 === 0
      ? { top: 0, bottom: 56, width: 90 }
      : NONE

  it('overlaps nothing', () => {
    const frame = arrange(graph, window, half)

    expect(getFootprintOverlaps(getFootprints(graph, frame, half))).toEqual([])
  })

  it('takes less height than the same relation with names under every note', () => {
    const every: Rooms = (node) =>
      node.relation === 'child' ? { top: 0, bottom: 56, width: 90 } : NONE
    const heightOf = (frame: PlexFrame) => frame.extent.maxY - frame.extent.minY

    expect(heightOf(arrange(graph, window, half))).toBeLessThan(
      heightOf(arrange(graph, window, every)),
    )
  })
})
