/** A note of one group never crosses the border into the territory of another group; a group that does not fit leaves notes out. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getBox, isOverlapping } from './layout/boxes'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { PlexEdge, PlexGraph, PlexNode, PlexRelation } from '../../../types'

const COUNTS: Readonly<Partial<Record<PlexRelation, number>>> = {
  parent: 3,
  child: 20,
  jump: 12,
  sibling: 40,
}

function createGraph(counts: Readonly<Partial<Record<PlexRelation, number>>>): PlexGraph {
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Flywheels', relation: 'focus' }]
  const edges: PlexEdge[] = []
  for (const [relation, count] of Object.entries(counts) as [PlexRelation, number][]) {
    for (let index = 0; index < count; index++) {
      const id = `${relation}-${index}`
      nodes.push({ id, title: `${relation} ${index}`, relation })
      if (relation === 'child') edges.push({ from: 'focus', to: id })
      else if (relation === 'sibling') edges.push({ from: 'parent-0', to: id })
      else edges.push({ from: id, to: 'focus' })
    }
  }
  return { nodes, edges }
}

const list = (frame: PlexFrame, relation: PlexRelation): PlacedNode[] =>
  frame.nodes.filter((node) => node.relation === relation)

const getLeft = (node: PlacedNode): number => node.x - node.width / 2
const getRight = (node: PlacedNode): number => node.x + node.width / 2

const getOverlaps = (frame: PlexFrame): string[] =>
  frame.nodes.flatMap((a, index) =>
    frame.nodes
      .slice(index + 1)
      .filter((b) => isOverlapping(getBox(a, 0), getBox(b, 0)))
      .map((b) => `${a.id} / ${b.id}`),
  )

function findCrossings(frame: PlexFrame): string[] {
  const middle = [...list(frame, 'parent'), ...list(frame, 'child')]
  const reachLeft = Math.min(...middle.map(getLeft))
  const reachRight = Math.max(...middle.map(getRight))
  return [
    ...list(frame, 'jump').filter((node) => getRight(node) > reachLeft),
    ...list(frame, 'sibling').filter((node) => getLeft(node) < reachRight),
  ].map((node) => node.id)
}

describe.each([
  { width: 1200, height: 900 },
  { width: 700, height: 500 },
  { width: 1880, height: 1500 },
])(
  'forty siblings, twenty children, twelve jumps and three parents in $width by $height',
  (viewport) => {
    const graph = createGraph(COUNTS)
    const frame = arrangePlex(graph, { options: { viewport } })

    it('keeps every jump and sibling out of the territory of the parents and children', () => {
      expect(list(frame, 'parent').length).toBeGreaterThan(0)
      expect(list(frame, 'child').length).toBeGreaterThan(0)
      expect(findCrossings(frame)).toStrictEqual([])
    })

    it('keeps the notes apart', () => {
      expect(getOverlaps(frame)).toStrictEqual([])
    })

    it('counts each note it does not show', () => {
      for (const relation of ['parent', 'child', 'jump', 'sibling'] as const) {
        expect(list(frame, relation).length + (frame.overflow[relation] ?? 0), relation).toBe(
          COUNTS[relation],
        )
      }
    })
  },
)

describe('in a window that holds the whole graph', () => {
  it('shows every note at 1200 by 900', () => {
    const frame = arrangePlex(createGraph(COUNTS), {
      options: { viewport: { width: 1200, height: 900 } },
    })
    expect(frame.overflow).toStrictEqual({})
  })
})

describe('in a window too small for the graph', () => {
  it('leaves notes out rather than set one in the territory of another group', () => {
    const frame = arrangePlex(createGraph(COUNTS), {
      options: { viewport: { width: 700, height: 500 } },
    })
    const leftOut = Object.values(frame.overflow).reduce((total, count) => total + count, 0)
    expect(leftOut).toBeGreaterThan(0)
    expect(findCrossings(frame)).toStrictEqual([])
  })
})
