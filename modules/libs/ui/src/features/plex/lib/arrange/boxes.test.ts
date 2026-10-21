/** The boxes a node and a run of a line take, worked out without drawing them. */
import { describe, expect, it } from 'vitest'
import { getBox, getTitleBoxes, isOverlapping, runBoxes, type Box } from './boxes'
import type { PlacedEdge } from '../edge'
import type { PlacedNode } from '../node'

const square: Box = { minX: 0, minY: 0, maxX: 10, maxY: 10 }

describe('whether two boxes share room', () => {
  it('is so for boxes that cross', () => {
    expect(isOverlapping(square, { minX: 5, minY: 5, maxX: 15, maxY: 15 })).toBe(true)
  })

  it.each([
    ['to the right', { minX: 10, minY: 0, maxX: 20, maxY: 10 }],
    ['to the left', { minX: -10, minY: 0, maxX: 0, maxY: 10 }],
    ['below', { minX: 0, minY: 10, maxX: 10, maxY: 20 }],
    ['above', { minX: 0, minY: -10, maxX: 10, maxY: 0 }],
  ])('is not so for a box that only touches on the side %s', (_side, other) => {
    expect(isOverlapping(square, other)).toBe(false)
    expect(isOverlapping(other, square)).toBe(false)
  })
})

describe('the room a node takes', () => {
  it('is the box around it, and the distance kept apart on every side', () => {
    const node: PlacedNode = {
      id: 'n',
      title: 'N',
      relation: 'child',
      x: 100,
      y: 50,
      width: 40,
      height: 20,
      order: 0,
      opacity: 1,
    }
    expect(getBox(node, 3)).toStrictEqual({ minX: 77, minY: 37, maxX: 123, maxY: 63 })
  })
})

describe('the steps a title is tried in', () => {
  const record = (from: number, to: number, step: number) => {
    const request: [number, number][] = []
    const boxes = getTitleBoxes(
      (a, b) => {
        request.push([a, b])
        return square
      },
      { from, to },
      step,
    )
    return { asked: request, boxes }
  }

  it('runs from one end to the other in even steps, and no further', () => {
    const { asked, boxes } = record(0, 1, 0.25)
    expect(asked).toStrictEqual([
      [0, 0.25],
      [0.25, 0.5],
      [0.5, 0.75],
      [0.75, 1],
    ])
    expect(boxes).toHaveLength(4)
  })

  it('ends the last step at the end when the steps do not divide the run', () => {
    expect(record(0, 1, 0.4).asked).toStrictEqual([
      [0, 0.4],
      [0.4, 0.8],
      [0.8, 1],
    ])
  })

  it('is nothing for a run of no length', () => {
    expect(record(0.5, 0.5, 0.25).boxes).toStrictEqual([])
  })
})

describe('the room a run of a line takes', () => {
  /** A straight line, so a fraction of the way along is a fraction of the distance. */
  const straight = (from: { x: number; y: number }, to: { x: number; y: number }): PlacedEdge => ({
    from: 'a',
    to: 'b',
    opacity: 1,
    heading: 'along',
    wordsAt: 0.5,
    fromPoint: from,
    toPoint: to,
    control1: { x: from.x + (to.x - from.x) / 3, y: from.y + (to.y - from.y) / 3 },
    control2: { x: from.x + ((to.x - from.x) * 2) / 3, y: from.y + ((to.y - from.y) * 2) / 3 },
  })

  it('is the run itself, thickened by half the depth of the type across a level line', () => {
    const box = runBoxes(straight({ x: 100, y: 50 }, { x: 340, y: 50 }), 10)(0.25, 0.5)
    expect(box.minX).toBeCloseTo(160)
    expect(box.maxX).toBeCloseTo(220)
    expect(box.minY).toBeCloseTo(45)
    expect(box.maxY).toBeCloseTo(55)
  })

  it('is thickened across a slanted line by the share of the depth that faces each way', () => {
    const box = runBoxes(straight({ x: 0, y: 0 }, { x: 300, y: 400 }), 10)(0.25, 0.75)
    expect(box.minX).toBeCloseTo(75 - 4)
    expect(box.maxX).toBeCloseTo(225 + 4)
    expect(box.minY).toBeCloseTo(100 - 3)
    expect(box.maxY).toBeCloseTo(300 + 3)
  })

  it('is the point itself on a line of no length', () => {
    const point = { x: 0, y: 0 }
    const box = runBoxes(straight(point, point), 10)(0, 1)
    expect(box).toStrictEqual({ minX: 0, minY: 0, maxX: 0, maxY: 0 })
  })
})
