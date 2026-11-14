/** A small group is set out of the bounds of a larger one: a few jumps stay beside the focus, not inside the grid of many children. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { build } from '../../fixtures/build'
import { getFootprint, getFootprintCenterY, type NodeRoom } from './layout/placement'
import type { PlacedNode } from '../node'
import type { PlexNode } from '../../../types'

/** A note with names under its pill, and over it the name of its parent. */
const NAMES: NodeRoom = { top: 16, bottom: 60, width: 140 }
const PARENT_NAMES: NodeRoom = { top: 0, bottom: 45, width: 124 }

const getRoom = (node: PlexNode): NodeRoom => (node.relation === 'parent' ? PARENT_NAMES : NAMES)

interface Box {
  readonly id: string
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

const getBox = (node: PlacedNode): Box => {
  const room = node.relation === 'focus' ? { top: 0, bottom: 0, width: 0 } : getRoom(node)
  const { width, height } = getFootprint(node, room)
  const y = getFootprintCenterY(node.y, room)
  return {
    id: node.id,
    minX: node.x - width / 2,
    maxX: node.x + width / 2,
    minY: y - height / 2,
    maxY: y + height / 2,
  }
}

const isOverlapping = (a: Box, b: Box): boolean =>
  a.minX < b.maxX - 1e-6 &&
  b.minX < a.maxX - 1e-6 &&
  a.minY < b.maxY - 1e-6 &&
  b.minY < a.maxY - 1e-6

const getBounds = (boxes: readonly Box[]): Box => ({
  id: 'bounds',
  minX: Math.min(...boxes.map((box) => box.minX)),
  maxX: Math.max(...boxes.map((box) => box.maxX)),
  minY: Math.min(...boxes.map((box) => box.minY)),
  maxY: Math.max(...boxes.map((box) => box.maxY)),
})

describe.each([
  { width: 1800, height: 1900 },
  { width: 1850, height: 1820 },
])('a note with many children, few jumps and many siblings in $width×$height', (viewport) => {
  const graph = build('Levers', { parent: 1, child: 53, jump: 3, reference: 3, sibling: 39 })
  const frame = arrangePlex(graph, {
    options: { viewport },
    measure: (node) => 40 + node.title.length * 7,
    room: getRoom,
  })
  const ofRelation = (relation: string) => frame.nodes.filter((node) => node.relation === relation)

  it('shows every note', () => {
    expect(frame.overflow).toStrictEqual({})
    expect(frame.nodes).toHaveLength(graph.nodes.length)
  })

  it('sets no jump inside the bounds of the children', () => {
    const children = getBounds(ofRelation('child').map(getBox))
    const inside = ofRelation('jump')
      .map(getBox)
      .filter((jump) => isOverlapping(jump, children))
      .map((jump) => jump.id)
    expect(inside).toStrictEqual([])
  })

  it('sets the jumps left of the focus', () => {
    for (const jump of ofRelation('jump')) expect(jump.x, jump.id).toBeLessThan(0)
  })

  it('overlaps no two notes', () => {
    const boxes = frame.nodes.map(getBox)
    const overlaps = boxes.flatMap((a, index) =>
      boxes
        .slice(index + 1)
        .filter((b) => isOverlapping(a, b))
        .map((b) => `${a.id} / ${b.id}`),
    )
    expect(overlaps).toStrictEqual([])
  })
})
