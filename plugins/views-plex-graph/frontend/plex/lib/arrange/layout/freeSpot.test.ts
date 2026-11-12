/** Where a box of a cluster goes: the free spot nearest the middle of its group. */
import { describe, expect, it } from 'vitest'
import { findFreeSpot, isFree, SpotIndex, TOLERANCE } from './freeSpot'

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

  it('moves further out along the direction when every candidate next to the group is occupied', () => {
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

/** A small seeded generator, so the boxes are the same on every run. */
function createRandom(seed: number): () => number {
  let state = seed
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0
    return state / 2 ** 32
  }
}

describe('the index agrees with the plain list', () => {
  const random = createRandom(1837)
  const between = (low: number, high: number) => low + random() * (high - low)
  // Boxes up to 200 wide and 90 high, so many cross the 64 unit cell borders.
  const taken = Array.from({ length: 120 }, () => ({
    x: between(-400, 400),
    y: between(-300, 300),
    width: between(10, 200),
    height: between(10, 90),
  }))
  // Boxes whose edges lie exactly on a cell border.
  taken.push(
    { x: 64, y: 64, width: 128, height: 64 },
    { x: -64, y: 0, width: 64, height: 128 },
    { x: 0, y: -128, width: 192, height: 20 },
  )
  const index = new SpotIndex(taken)

  it.each([GAPS, { gap: 0, lineGap: 0 }, { gap: 70, lineGap: 130 }])(
    'answers isFree alike for probes everywhere, with gaps %o',
    (gaps) => {
      const probes = [
        ...Array.from({ length: 600 }, () => ({
          x: between(-500, 500),
          y: between(-400, 400),
          width: between(10, 150),
          height: between(10, 80),
        })),
        // On the borders, and one gap from a taken box.
        ...taken.flatMap((box) => [
          { x: box.x + (box.width + SIZE.width) / 2 + gaps.gap, y: box.y, ...SIZE },
          { x: box.x, y: box.y + (box.height + SIZE.height) / 2 + gaps.lineGap, ...SIZE },
          { x: 64 * Math.round(box.x / 64), y: 64 * Math.round(box.y / 64), ...SIZE },
        ]),
      ]
      for (const probe of probes) {
        expect(index.isFree(probe, gaps), JSON.stringify(probe)).toBe(isFree(probe, taken, gaps))
      }
    },
  )

  it('finds the same spot from the index and from the list, and the plain list calls it free', () => {
    for (const [x, y] of [
      [0, 0],
      [70, 60],
      [-130, 40],
      [200, -150],
    ] as const) {
      const request = {
        start: { x, y },
        middle: { x, y },
        direction: { x: 0, y: 1 },
        size: SIZE,
        group: [at(x, y)],
        gaps: GAPS,
      }
      const fromIndex = findFreeSpot(index, request)
      expect(fromIndex).toStrictEqual(findFreeSpot(taken, request))
      expect(isFree(fromIndex, taken, GAPS)).toBe(true)
    }
  })
})

describe('equal costs', () => {
  it('are decided by the order of the candidates when along differs by float noise only', () => {
    const box = at(0, 0)
    const spot = findFreeSpot([box], {
      start: { x: 0, y: 0 },
      middle: { x: 0, y: 0 },
      direction: { x: 1e-12, y: 1 },
      size: SIZE,
      group: [box],
      gaps: GAPS,
    })
    // The candidate on the right is listed before the one on the left.
    expect(spot.x).toBe(SIZE.width + GAPS.gap)
    expect(spot.y).toBe(0)
  })
})
