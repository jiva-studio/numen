import { describe, expect, it } from 'vitest'
import type { PlexFrame } from '../../lib/frame'
import type { PlacedNode } from '../../lib/node'
import type { PlexRelatedSeat } from '../../lib/seat'
import { getDraggedShape, getReachGhost, getReachThread } from './threads'

const SIZE = { width: 40, height: 20 }

const createNode = (id: string, seat: PlacedNode['seat'], x: number, y: number): PlacedNode => ({
  id,
  title: id,
  seat,
  x,
  y,
  width: 40,
  height: 20,
  order: 0,
  opacity: 1,
})

const createFrame = (nodes: PlacedNode[]): PlexFrame => ({
  nodes,
  edges: [],
  extent: { minX: 0, minY: 0, maxX: 0, maxY: 0 },
  overflow: {},
})

const frame = createFrame([createNode('a', 'focus', 0, 0), createNode('b', 'child', 100, 50)])

const nameSeat = (seat: PlexRelatedSeat): string => `new ${seat}`

describe('the line dragged from a node', () => {
  it('runs from the handle of the node it left to the pointer', () => {
    expect(getReachThread(frame, 'b', { x: 220, y: 150 })).toBe('M 120 50 C 170 50 170 150 220 150')
  })

  it('is nothing off a node', () => {
    expect(getReachThread(frame, null, { x: 220, y: 150 })).toBeNull()
    expect(getReachThread(frame, 'missing', { x: 220, y: 150 })).toBeNull()
  })

  it('is nothing without a pointer', () => {
    expect(getReachThread(frame, 'b', null)).toBeNull()
  })
})

describe('the box a gesture would leave', () => {
  const to = { x: 30, y: 40 }

  it('is a box with the seat name where the pointer is when letting go creates', () => {
    expect(getReachGhost({ kind: 'create', from: 'a', seat: 'child' }, to, nameSeat, SIZE)).toEqual(
      {
        id: 'ghost',
        title: 'new child',
        seat: 'child',
        x: 30,
        y: 40,
        width: 40,
        height: 20,
        order: 0,
        opacity: 1,
      },
    )
  })

  it('is nothing when there is no outcome or the outcome links', () => {
    expect(getReachGhost(null, to, nameSeat, SIZE)).toBeNull()
    expect(
      getReachGhost({ kind: 'link', from: 'a', to: 'b', seat: 'child' }, to, nameSeat, SIZE),
    ).toBeNull()
  })

  it('is nothing without a pointer', () => {
    expect(
      getReachGhost({ kind: 'create', from: 'a', seat: 'child' }, null, nameSeat, SIZE),
    ).toBeNull()
  })
})

describe('something dragged over the picture', () => {
  const to = { x: 30, y: 40 }
  const focusLast = createFrame([
    createNode('b', 'child', 100, 50),
    createNode('a', 'focus', 10, 20),
  ])

  it('is a line from the focus to the pointer and the box letting go would leave', () => {
    const shape = getDraggedShape(focusLast, 'sibling', to, nameSeat, SIZE)

    expect(shape?.thread).toBe('M 10 20 C 20 20 20 40 30 40')
    expect(shape?.ghost).toEqual({
      id: 'dragged',
      title: 'new sibling',
      seat: 'sibling',
      x: 30,
      y: 40,
      width: 40,
      height: 20,
      order: 0,
      opacity: 1,
    })
  })

  it('is nothing without a seat, a pointer or a focus', () => {
    expect(getDraggedShape(focusLast, null, to, nameSeat, SIZE)).toBeNull()
    expect(getDraggedShape(focusLast, 'child', null, nameSeat, SIZE)).toBeNull()
    expect(
      getDraggedShape(createFrame([createNode('b', 'child', 0, 0)]), 'child', to, nameSeat, SIZE),
    ).toBeNull()
  })
})
