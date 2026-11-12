/** The line dragged out of a node bows as much as a connection does. */
import { describe, expect, it } from 'vitest'
import { createThreadPath } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import { getDraggedShape, getPendingThread, getReachThread } from './threads'

const getNumbers = (path: string): number[] =>
  [...path.matchAll(/-?\d+(?:\.\d+)?/g)].map((found) => Number(found[0]))

const frame: PlexFrame = {
  nodes: [
    { id: 'a', title: 'a', relation: 'focus', x: 0, y: 0, width: 40, height: 20, order: 0, opacity: 1 },
    { id: 'b', title: 'b', relation: 'child', x: 100, y: 50, width: 40, height: 20, order: 0, opacity: 1 },
  ],
  edges: [],
  extent: { minX: 0, minY: 0, maxX: 0, maxY: 0 },
  overflow: {},
}

describe('how far the line dragged out of a node bows', () => {
  const start = { x: 0, y: 0 }
  const to = { x: 100, y: 40 }

  it('is straight at a curvature of nothing', () => {
    const [, , firstX, firstY, secondX, secondY] = getNumbers(createThreadPath(start, to, 0))
    expect([firstX, firstY]).toEqual([0, 0])
    expect([secondX, secondY]).toEqual([100, 40])
  })

  it('reaches twice as far at twice the curvature', () => {
    const reach = (curvature: number) => getNumbers(createThreadPath(start, to, curvature))[2]!
    expect(reach(0.55)).toBe(50)
    expect(reach(1.1)).toBeCloseTo(2 * reach(0.55))
    expect(reach(0.275)).toBeCloseTo(reach(0.55) / 2)
  })

  it('is the same line as the one drawn before when no curvature is given', () => {
    expect(createThreadPath(start, to)).toBe(createThreadPath(start, to, 0.55))
  })

  it('is passed from the gesture to the line of each of its three drawings', () => {
    const straight = (path: string | null) => getNumbers(path!).slice(2, 4)
    expect(straight(getReachThread(frame, 'b', { x: 220, y: 150 }, 0))).toEqual([120, 50])
    expect(
      straight(
        getPendingThread(frame, {
          from: 'b',
          position: { x: 420, y: 250 },
          viewport: { width: 400, height: 200 },
          curvature: 0,
        }),
      ),
    ).toEqual([120, 50])
    const shape = getDraggedShape(
      frame,
      { relation: 'child', to: { x: 220, y: 150 } },
      { dropName: String, nodeSize: { width: 40, height: 20 }, curvature: 0 },
    )
    expect(straight(shape!.thread)).toEqual([0, 0])
  })
})
