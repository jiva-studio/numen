/** With the grid setting on, the pills of a side line up whatever the views over and under them hold; off, each note keeps only its own room. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getFootprint, getFootprintCenterY, type NodeRoom } from './layout/placement'
import type { PlexGraph, PlexNode } from '../../../types'

const WIDTH = 120
const COUNT = 6

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Focus', relation: 'focus' },
    ...Array.from({ length: COUNT }, (_, index): PlexNode => ({
      id: `child-${index}`,
      title: `Child ${index}`,
      relation: 'child',
    })),
  ],
  edges: Array.from({ length: COUNT }, (_, index) => ({ from: 'focus', to: `child-${index}` })),
}

const room = (node: PlexNode): NodeRoom => {
  if (node.relation === 'focus') return { top: 0, bottom: 0, width: WIDTH }
  const index = Number(node.id.split('-')[1])
  return { top: index === 0 ? 70 : 10, bottom: index === 1 ? 90 : 20, width: WIDTH }
}

const arrange = (alignToGrid: boolean) =>
  arrangePlex(graph, {
    options: { alignToGrid, viewport: { width: 1600, height: 1200 } },
    measure: () => WIDTH,
    room,
  }).nodes.filter((node) => node.relation === 'child')

const rowsOf = (nodes: ReturnType<typeof arrange>) => [...new Set(nodes.map((node) => node.y))]

describe('aligning the notes to a grid', () => {
  it('puts the pills of one row at one height', () => {
    expect(rowsOf(arrange(true)).length).toBeLessThan(rowsOf(arrange(false)).length)
  })

  it('leaves no footprints overlapping', () => {
    const boxes = arrange(true).map((node) => {
      const footprint = getFootprint(node, room(node))
      const y = getFootprintCenterY(node.y, room(node))
      return { x: node.x, y, ...footprint }
    })
    const overlaps = boxes.flatMap((a, index) =>
      boxes
        .slice(index + 1)
        .filter(
          (b) =>
            Math.abs(a.x - b.x) < (a.width + b.width) / 2 - 1e-6 &&
            Math.abs(a.y - b.y) < (a.height + b.height) / 2 - 1e-6,
        ),
    )

    expect(overlaps).toHaveLength(0)
  })

  it('does not line pills up when off', () => {
    const [first, second] = arrange(false)

    expect(first!.y - second!.y).not.toBe(0)
  })
})
