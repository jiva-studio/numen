/** Where a box of a cluster goes: the free spot nearest the middle of its group. */
import { describe, expect, it } from 'vitest'
import { findFreeSpot, isFree, TOLERANCE } from './freeSpot'

const SIZE = { width: 100, height: 30 }
const GAPS = { gap: 16, lineGap: 20 }
const at = (x: number, y: number) => ({ x, y, ...SIZE })

describe('a spot is free', () => {
  const taken = [at(0, 0)]

  it('when it touches a taken box at exactly one gap sideways', () => {
    expect(isFree(at(SIZE.width + GAPS.gap, 0), taken, GAPS)).toBe(true)
    expect(isFree(at(-(SIZE.width + GAPS.gap), 0), taken, GAPS)).toBe(true)
  })

  it('when it touches a taken box at exactly one line gap below or above', () => {
    expect(isFree(at(0, SIZE.height + GAPS.lineGap), taken, GAPS)).toBe(true)
    expect(isFree(at(0, -(SIZE.height + GAPS.lineGap)), taken, GAPS)).toBe(true)
  })

  it('also when floating point error puts it a hair inside one gap', () => {
    expect(TOLERANCE).toBe(0.5)
    expect(isFree(at(SIZE.width + GAPS.gap - 0.1 * TOLERANCE, 0), taken, GAPS)).toBe(true)
  })

  it('not when it is one unit nearer than one gap sideways', () => {
    expect(isFree(at(SIZE.width + GAPS.gap - 1, 0), taken, GAPS)).toBe(false)
  })

  it('not when it is one unit nearer than one line gap below', () => {
    expect(isFree(at(0, SIZE.height + GAPS.lineGap - 1), taken, GAPS)).toBe(false)
  })

  it('not when it is on top of a taken box', () => {
    expect(isFree(at(10, 5), taken, GAPS)).toBe(false)
  })
})

describe('the free spot of a group', () => {
  const direction = { x: 0, y: 1 }
  const start = { x: 0, y: 200 }
  const request = (group: ReturnType<typeof at>[], middle = start) => ({
    start,
    middle,
    direction,
    size: SIZE,
    group,
    gaps: GAPS,
  })

  it('touches the one box already placed, at one gap, and is no nearer the focus than the start', () => {
    const first = at(start.x, start.y)
    const spot = findFreeSpot([at(0, 0), first], request([first]))

    const sideways = Math.abs(spot.x - first.x) - SIZE.width
    const lengthways = Math.abs(spot.y - first.y) - SIZE.height
    const touches =
      (Math.abs(sideways - GAPS.gap) < 1e-6 && Math.abs(spot.y - first.y) < SIZE.height) ||
      (Math.abs(lengthways - GAPS.lineGap) < 1e-6 && Math.abs(spot.x - first.x) < SIZE.width)
    expect(touches).toBe(true)
    expect(
      (spot.y - start.y) * direction.y + (spot.x - start.x) * direction.x,
    ).toBeGreaterThanOrEqual(-0.5)
    expect(isFree(spot, [at(0, 0), first], GAPS)).toBe(true)
  })

  it('is the middle itself when nothing else is there', () => {
    const middle = { x: 40, y: 260 }
    expect(findFreeSpot([at(0, 0)], request([], middle))).toStrictEqual({ ...middle, ...SIZE })
  })

  it('goes further out along the direction when every candidate beside the group is taken', () => {
    const first = at(start.x, start.y)
    const wall = [-3, -2, -1, 0, 1, 2, 3].flatMap((column) =>
      [-1, 0, 1, 2].map((row) =>
        at(column * (SIZE.width + GAPS.gap), start.y + row * (SIZE.height + GAPS.lineGap)),
      ),
    )
    const spot = findFreeSpot([at(0, 0), ...wall], request([first]))

    expect(isFree(spot, [at(0, 0), ...wall], GAPS)).toBe(true)
    expect(spot.y).toBeGreaterThan(start.y)
  })
})
