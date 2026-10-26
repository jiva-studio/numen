import { describe, expect, it } from 'vitest'
import type { PlexFrame } from '../../lib/frame'
import type { PlacedNode } from '../../lib/node'
import type { PlexRelation } from '../../lib/relation'
import { getDraggedShape, getPendingThread, getReachGhost, getReachThread } from './threads'

const SIZE = { width: 40, height: 20 }

const createNode = (
  id: string,
  relation: PlacedNode['relation'],
  x: number,
  y: number,
): PlacedNode => ({
  id,
  title: id,
  relation,
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

const getRelationLabel = (relation: PlexRelation): string => `new ${relation}`

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

describe('the line to a point awaiting a node', () => {
  const viewport = { width: 400, height: 200 }

  it('runs from the handle of the node to the point measured from the corner', () => {
    expect(getPendingThread(frame, { from: 'b', position: { x: 420, y: 250 }, viewport })).toBe(
      'M 120 50 C 170 50 170 150 220 150',
    )
  })

  it('is nothing off a node or without a point', () => {
    expect(
      getPendingThread(frame, { from: 'missing', position: { x: 420, y: 250 }, viewport }),
    ).toBeNull()
    expect(
      getPendingThread(frame, { from: null, position: { x: 420, y: 250 }, viewport }),
    ).toBeNull()
    expect(getPendingThread(frame, { from: 'b', position: null, viewport })).toBeNull()
  })
})

describe('the box a gesture would leave', () => {
  const to = { x: 30, y: 40 }
  const options = { relationName: getRelationLabel, nodeSize: SIZE }

  it('is a box with the relation name where the pointer is when letting go creates', () => {
    expect(getReachGhost({ kind: 'create', from: 'a', relation: 'child' }, to, options)).toEqual({
      id: 'ghost',
      title: 'new child',
      relation: 'child',
      x: 30,
      y: 40,
      width: 40,
      height: 20,
      order: 0,
      opacity: 1,
    })
  })

  it('is nothing when there is no outcome or the outcome links', () => {
    expect(getReachGhost(null, to, options)).toBeNull()
    expect(
      getReachGhost({ kind: 'link', from: 'a', to: 'b', relation: 'child' }, to, options),
    ).toBeNull()
  })

  it('is nothing without a pointer', () => {
    expect(
      getReachGhost({ kind: 'create', from: 'a', relation: 'child' }, null, options),
    ).toBeNull()
  })
})

describe('something dragged over the picture', () => {
  const to = { x: 30, y: 40 }
  const focusLast = createFrame([
    createNode('b', 'child', 100, 50),
    createNode('a', 'focus', 10, 20),
  ])
  const options = { dropName: getRelationLabel, nodeSize: SIZE }

  it('is a line from the focus to the pointer and the box letting go would leave', () => {
    const shape = getDraggedShape(focusLast, { relation: 'sibling', to }, options)

    expect(shape?.thread).toBe('M 10 20 C 20 20 20 40 30 40')
    expect(shape?.ghost).toEqual({
      id: 'dragged',
      title: 'new sibling',
      relation: 'sibling',
      x: 30,
      y: 40,
      width: 40,
      height: 20,
      order: 0,
      opacity: 1,
    })
  })

  it('is nothing without a relation, a pointer or a focus', () => {
    expect(getDraggedShape(focusLast, { relation: null, to }, options)).toBeNull()
    expect(getDraggedShape(focusLast, { relation: 'child', to: null }, options)).toBeNull()
    expect(
      getDraggedShape(
        createFrame([createNode('b', 'child', 0, 0)]),
        { relation: 'child', to },
        options,
      ),
    ).toBeNull()
  })
})
