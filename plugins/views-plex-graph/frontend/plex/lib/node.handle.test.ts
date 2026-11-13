/** Where the handle of a node sits, and the side it takes at the edge of the window. */
import { describe, expect, it } from 'vitest'
import { getHandle, HANDLE_RADIUS, HANDLE_STROKE, type PlacedNode } from './node'

const createNode = (overrides: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 0,
  y: 0,
  width: 100,
  height: 36,
  order: 0,
  opacity: 1,
  ...overrides,
})

const BEYOND = 2 * HANDLE_RADIUS + HANDLE_STROKE

describe('the handle of a node', () => {
  it('is on the trailing edge of a pill', () => {
    expect(getHandle(createNode())).toStrictEqual({ x: 50, y: 0 })
  })

  it('is beyond the trailing edge of a node without a pill', () => {
    expect(getHandle(createNode(), false)).toStrictEqual({ x: 50 + HANDLE_RADIUS, y: 0 })
  })

  it('is on the trailing side while its disc fits inside the window', () => {
    const node = createNode({ x: 400 - 50 - BEYOND })

    expect(getHandle(node, false, 800).x).toBe(50 + HANDLE_RADIUS)
  })

  it('is on the leading side of a node without a pill when its disc would leave the window', () => {
    const node = createNode({ x: 400 - 50 - BEYOND + 1 })

    expect(getHandle(node, false, 800)).toStrictEqual({ x: -(50 + HANDLE_RADIUS), y: 0 })
  })

  it('is on the edge of a pill, whatever the window', () => {
    const node = createNode({ x: 400 - 50 })

    expect(getHandle(node, true, 800)).toStrictEqual({ x: 50, y: 0 })
  })
})
