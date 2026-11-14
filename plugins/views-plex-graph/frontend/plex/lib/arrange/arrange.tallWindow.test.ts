/** A wide note of a column on the right may sit where a narrower one before it left room, above or below it; a note never leaves its strip. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import type { PlexGraph, PlexNode, PlexRelation } from '../../../types'

const WINDOW = { width: 912.234, height: 1320.125 }
const SCALE = 1.2

const WIDTHS: Readonly<Record<PlexRelation | 'focus', readonly number[]>> = {
  focus: [450 / 1.22],
  parent: [120, 140],
  child: [110, 150, 100, 172, 130, 90, 160, 120],
  jump: [120, 172, 100, 140, 150],
  reference: [172, 172, 160, 172, 150, 172],
  sibling: [100, 120, 110, 165, 172.7],
}

function createGraph(): { graph: PlexGraph; widths: Map<string, number> } {
  const nodes: PlexNode[] = []
  const widths = new Map<string, number>()
  for (const [relation, list] of Object.entries(WIDTHS)) {
    list.forEach((width, index) => {
      const id = `${relation}-${index}`
      widths.set(id, width)
      nodes.push({ id, title: id, relation: relation as PlexRelation | 'focus' } as PlexNode)
    })
  }
  return { graph: { nodes, edges: [] } as unknown as PlexGraph, widths }
}

describe('a tall narrow window with a wide focus', () => {
  const { graph, widths } = createGraph()
  const frame = arrangePlex(graph, {
    options: {
      viewport: WINDOW,
      margin: 19.19,
      nodeSize: { width: 172.73, height: 25.85 },
      focusSize: { width: 450, height: 64.6 },
      gap: 16 * SCALE,
      lineGap: 20 * SCALE,
      focusGap: 56 * SCALE,
      minWidth: 72 * SCALE,
    },
    measure: (node) => widths.get(node.id) ?? 0,
    room: (node) =>
      node.relation === 'focus'
        ? { top: 0, bottom: 0, width: 0 }
        : { top: 0, bottom: 48, width: 120 },
  })
  it('keeps the notes it shows clear of each other and inside the window', () => {
    const footprints = frame.nodes.map((node) => {
      const room = node.relation === 'focus' ? 0 : 48
      const width = Math.max(node.width, node.relation === 'focus' ? 0 : 120)
      return {
        id: node.id,
        minX: node.x - width / 2,
        maxX: node.x + width / 2,
        minY: node.y - node.height / 2,
        maxY: node.y + node.height / 2 + room,
      }
    })
    const overlaps = footprints.flatMap((a, index) =>
      footprints
        .slice(index + 1)
        .filter((b) => a.minX < b.maxX && b.minX < a.maxX && a.minY < b.maxY && b.minY < a.maxY)
        .map((b) => `${a.id} / ${b.id}`),
    )

    expect(overlaps).toEqual([])
    expect(
      footprints.every((box) => box.maxX <= WINDOW.width / 2 && box.minX >= -WINDOW.width / 2),
    ).toBe(true)
  })

  it('keeps the siblings to the right of the focus and counts those that do not fit', () => {
    const focus = frame.nodes.find((node) => node.relation === 'focus')!
    const siblings = frame.nodes.filter((node) => node.relation === 'sibling')

    expect(siblings.length + (frame.overflow.sibling ?? 0)).toBe(WIDTHS.sibling.length)
    for (const sibling of siblings) {
      expect(sibling.x - sibling.width / 2, sibling.id).toBeGreaterThanOrEqual(
        focus.x + focus.width / 2,
      )
    }
  })
})
