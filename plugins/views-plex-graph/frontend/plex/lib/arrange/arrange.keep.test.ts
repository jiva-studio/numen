/** A frame that keeps the notes an arrangement showed puts each of them where that arrangement put it, whatever else the window could hold. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import type { PlexGraph, PlexNode, PlexRelation } from '../../../types'

const COUNTS: Readonly<Record<string, number>> = { parent: 3, sibling: 6, child: 10, reference: 24 }
const WINDOWS = [
  { width: 600, height: 300 },
  { width: 800, height: 500 },
  { width: 1200, height: 800 },
]

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7
const roomOf = (node: PlexNode) =>
  node.relation === 'focus' ? { top: 0, bottom: 0, width: 90 } : { top: 0, bottom: 56, width: 90 }

function createGraph(): PlexGraph {
  const nodes: PlexNode[] = [{ id: 'focus', title: 'Thermodynamics', relation: 'focus' }]
  for (const [relation, count] of Object.entries(COUNTS)) {
    for (let index = 0; index < count; index++) {
      const title = 'Abcdefghijklmnopqrstuv'.slice(0, 8 + ((index * 7) % 13))
      nodes.push({ id: `${relation}-${index}`, title, relation: relation as PlexRelation })
    }
  }
  return { nodes, edges: [] }
}

const getPositions = (frame: ReturnType<typeof arrangePlex>) =>
  frame.nodes.map(({ id, x, y, width, height }) => ({ id, x, y, width, height }))

describe.each(WINDOWS)('a window of $width by $height', (viewport) => {
  const graph = createGraph()
  const given = { options: { viewport }, measure: byTitle, room: roomOf }
  const candidates = new Set(
    arrangePlex(graph, { ...given, room: undefined }).nodes.map((n) => n.id),
  )
  const first = arrangePlex(graph, { ...given, only: candidates })
  const shown = new Set(first.nodes.map((node) => node.id))

  it('shows fewer notes than the arrangement without rooms offered', () => {
    expect(shown.size).toBeLessThanOrEqual(candidates.size)
  })

  it('puts every note where the arrangement that showed it put it, when the inputs are the same', () => {
    const again = arrangePlex(graph, { ...given, only: candidates, keep: shown })

    expect(getPositions(again)).toStrictEqual(getPositions(first))
    expect(again.overflow).toStrictEqual(first.overflow)
  })

  it('shows none of the notes it cut when their rooms grow', () => {
    const taller = (node: PlexNode) => {
      const room = roomOf(node)
      return { ...room, bottom: room.bottom + 80 }
    }

    const again = arrangePlex(graph, { ...given, room: taller, only: candidates, keep: shown })

    expect(again.nodes.every((node) => shown.has(node.id))).toBe(true)
  })
})
