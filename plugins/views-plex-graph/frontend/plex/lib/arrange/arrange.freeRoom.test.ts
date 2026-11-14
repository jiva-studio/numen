/** A note is dropped only when the window has no free place for it. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getFootprint } from './layout/placement'
import type { PlacedNode } from '../node'
import type { PlexGraph, PlexNode, PlexRelation } from '../../../types'

const COUNTS: Readonly<Record<string, number>> = {
  parent: 3,
  sibling: 5,
  child: 4,
  reference: 24,
  jump: 2,
}
const WINDOW = { width: 800, height: 900 }
const ROOM = { top: 0, bottom: 56, width: 90 }
const NO_ROOM = { top: 0, bottom: 0, width: 0 }

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7
const roomOf = (node: PlexNode) => (node.relation === 'focus' ? NO_ROOM : ROOM)

function createGraph(): PlexGraph {
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Thermodynamics', relation: 'focus' }]
  for (const [relation, count] of Object.entries(COUNTS)) {
    for (let index = 0; index < count; index++) {
      const title = 'Abcdefghijklmnopqrstuv'.slice(0, 14 + ((index * 7) % 9))
      nodes.push({ id: `${relation}-${index}`, title, relation: relation as PlexRelation })
    }
  }
  return { nodes, edges: [] } as unknown as PlexGraph
}

const isOverlapping = (a: PlacedNode, b: PlacedNode): boolean => {
  const first = getFootprint(a, a.relation === 'focus' ? NO_ROOM : ROOM)
  const second = getFootprint(b, b.relation === 'focus' ? NO_ROOM : ROOM)
  const aY = a.y + (a.relation === 'focus' ? 0 : ROOM.bottom / 2)
  const bY = b.y + (b.relation === 'focus' ? 0 : ROOM.bottom / 2)
  return (
    Math.abs(a.x - b.x) < (first.width + second.width) / 2 &&
    Math.abs(aY - bY) < (first.height + second.height) / 2
  )
}

describe('a window that has free room next to the other groups', () => {
  const frame = arrangePlex(createGraph(), {
    options: { viewport: WINDOW },
    measure: byTitle,
    room: roomOf,
  })

  it('shows every sibling, since the siblings keep a free place above and below their column', () => {
    expect(frame.overflow.sibling).toBeUndefined()
  })

  it('keeps the notes it shows clear of each other', () => {
    const overlaps = frame.nodes.flatMap((a, index) =>
      frame.nodes.slice(index + 1).filter((b) => isOverlapping(a, b)),
    )

    expect(overlaps).toEqual([])
  })
})
