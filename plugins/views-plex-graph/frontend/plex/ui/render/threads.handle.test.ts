/** The line a reach drags starts where the handle is drawn. */
import { describe, expect, it } from 'vitest'
import { getPendingThread, getReachThread } from './threads'
import type { PlexFrame } from '../../lib/frame'
import { HANDLE_RADIUS, type PlacedNode } from '../../lib/node'

const node: PlacedNode = {
  id: 'b',
  title: 'b',
  relation: 'child',
  x: 100,
  y: 50,
  width: 40,
  height: 20,
  order: 0,
  opacity: 1,
}

const frame: PlexFrame = {
  nodes: [node],
  edges: [],
  extent: { minX: 0, minY: 0, maxX: 0, maxY: 0 },
  overflow: {},
}

describe('the line dragged from a node without a pill', () => {
  it('starts at the disc of the handle beyond the edge of the box', () => {
    const thread = getReachThread(frame, 'b', { x: 220, y: 150 }, { hasPill: false })

    expect(thread?.startsWith(`M ${100 + 20 + HANDLE_RADIUS} 50 `)).toBe(true)
  })

  it('starts at the leading side when the handle was drawn there', () => {
    const thread = getReachThread(frame, 'b', { x: -220, y: 150 }, {
      hasPill: false,
      viewportWidth: 260,
    })

    expect(thread?.startsWith(`M ${100 - 20 - HANDLE_RADIUS} 50 `)).toBe(true)
  })

  it('is the same for a pending line', () => {
    const thread = getPendingThread(frame, {
      from: 'b',
      position: { x: 600, y: 400 },
      viewport: { width: 1000, height: 800 },
      hasPill: false,
    })

    expect(thread?.startsWith(`M ${100 + 20 + HANDLE_RADIUS} 50 `)).toBe(true)
  })
})
