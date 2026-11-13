/** Where a box of a cluster goes: the free spot nearest the middle of its group. */
import { describe, expect, it } from 'vitest'
import {
  findFreeSpot,
  listCandidatesAround,
  measureAlong,
  measureCost,
  SpotIndex,
  type Spot,
  type SpotRequest,
} from './freeSpot'
import { createRandom } from '../../../fixtures/random'

const SIZE = { width: 100, height: 30 }
const GAPS = { gap: 16, lineGap: 20 }
const at = (x: number, y: number) => ({ x, y, ...SIZE })

/** The distance a spot may sit inside a gap, which float error puts there. */
const TOLERANCE = 0.5

const isFree = (spot: Spot, taken: readonly Spot[], gaps: typeof GAPS): boolean =>
  new SpotIndex(taken).isFree(spot, gaps)

const isFreeAmong = (spot: Spot, taken: readonly Spot[], gaps: typeof GAPS): boolean =>
  taken.every(
    (other) =>
      Math.abs(spot.x - other.x) >=
        (spot.width + other.width) / 2 + Math.max(0, gaps.gap - TOLERANCE) ||
      Math.abs(spot.y - other.y) >=
        (spot.height + other.height) / 2 + Math.max(0, gaps.lineGap - TOLERANCE),
  )

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
    const spot = findFreeSpot(new SpotIndex([at(0, 0), first]), request([first]))

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
    expect(findFreeSpot(new SpotIndex([at(0, 0)]), request([], middle))).toStrictEqual({
      ...middle,
      ...SIZE,
    })
  })

  it('moves further out along the direction when every candidate next to the group is occupied', () => {
    const first = at(start.x, start.y)
    const wall = [-3, -2, -1, 0, 1, 2, 3].flatMap((column) =>
      [-1, 0, 1, 2].map((row) =>
        at(column * (SIZE.width + GAPS.gap), start.y + row * (SIZE.height + GAPS.lineGap)),
      ),
    )
    const spot = findFreeSpot(new SpotIndex([at(0, 0), ...wall]), request([first]))

    expect(isFree(spot, [at(0, 0), ...wall], GAPS)).toBe(true)
    expect(spot.y).toBeGreaterThan(start.y)
  })
})

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
        expect(index.isFree(probe, gaps), JSON.stringify(probe)).toBe(
          isFreeAmong(probe, taken, gaps),
        )
      }
    },
  )

  it('finds a spot the rule checked against every box calls free', () => {
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
      expect(isFreeAmong(findFreeSpot(index, request), taken, GAPS)).toBe(true)
    }
  })
})

describe('equal costs', () => {
  it('still give a free spot beside the group, where the direction differs from straight down by float noise only', () => {
    const box = at(0, 0)
    const taken = new SpotIndex([box])
    const spot = findFreeSpot(taken, {
      start: { x: 0, y: 0 },
      middle: { x: 0, y: 0 },
      direction: { x: 1e-12, y: 1 },
      size: SIZE,
      group: [box],
      gaps: GAPS,
    })

    expect(taken.isFree(spot, GAPS)).toBe(true)
  })
})

function findLeastCostSpot(index: SpotIndex, request: SpotRequest): Spot | undefined {
  const costOf = (spot: Spot) => Math.round(measureCost(spot, request) * 1e6)
  const alongOf = (spot: Spot) => Math.round(measureAlong(spot, request) * 1e6)
  const candidates = [
    request.middle,
    request.start,
    ...request.group.flatMap((box) => listCandidatesAround(box, request)),
  ]
  let best: Spot | undefined
  for (const point of candidates) {
    if (measureAlong(point, request) < -TOLERANCE) continue
    const spot = { x: point.x, y: point.y, ...request.size }
    if (request.fits && !request.fits(spot)) continue
    if (!index.isFree(spot, request.gaps)) continue
    if (
      !best ||
      costOf(spot) < costOf(best) ||
      (costOf(spot) === costOf(best) && alongOf(spot) < alongOf(best))
    ) {
      best = spot
    }
  }
  return best
}

describe('the pruning of boxes that cannot win', () => {
  const random = createRandom(1837)
  const between = (low: number, high: number) => low + random() * (high - low)
  const DIRECTIONS = [
    { x: 0, y: 1 },
    { x: 0, y: -1 },
    { x: 1, y: 0 },
    { x: -1, y: 0 },
    { x: -0.92, y: 0.39 },
  ]

  it('gives the spot of least cost among all candidates, for 200 random groups', () => {
    let compared = 0
    for (let round = 0; round < 200; round++) {
      const size = { width: between(40, 160), height: between(20, 50) }
      const gaps = { gap: between(0, 30), lineGap: between(0, 30) }
      const direction = DIRECTIONS[round % DIRECTIONS.length]!
      const start = { x: between(-200, 200), y: between(-200, 200) }
      const middle = {
        x: start.x + direction.x * between(0, 150),
        y: start.y + direction.y * between(0, 150),
      }
      const group: Spot[] = []
      const index = new SpotIndex([at(0, 0)])
      const base = { start, middle, direction, size, gaps }
      for (let placed = between(1, 9) | 0; placed > 0; placed--) {
        const spot = findFreeSpot(index, { ...base, group })
        index.add(spot)
        group.push(spot)
      }
      for (let others = between(0, 12) | 0; others > 0; others--) {
        index.add({ x: between(-400, 400), y: between(-300, 300), ...size })
      }
      const request: SpotRequest = {
        ...base,
        group,
        fits: round % 2 === 0 ? undefined : () => true,
      }

      const least = findLeastCostSpot(index, request)
      if (!least) continue
      compared++
      expect(findFreeSpot(index, request), `round ${round}`).toStrictEqual(least)
    }
    expect(compared).toBeGreaterThan(100)
  })
})

describe('a window the group has no room to grow deeper in', () => {
  const taken = at(0, 100)
  const window = { width: 200, height: 200 }
  const fits = (spot: Spot) =>
    Math.abs(spot.x) + spot.width / 2 <= window.width / 2 &&
    Math.abs(spot.y) + spot.height / 2 <= window.height / 2
  const request = {
    start: { x: 0, y: 60 },
    middle: { x: 0, y: 60 },
    direction: { x: 0, y: 1 },
    size: SIZE,
    group: [],
    gaps: GAPS,
  }
  const index = new SpotIndex([at(0, -200), taken])

  it('takes a free spot inside it that lies a little before the start, as far back as the box reaches', () => {
    const spot = findFreeSpot(index, { ...request, fits })

    expect(spot).toStrictEqual(at(0, 100 - SIZE.height - GAPS.lineGap))
    expect(measureAlong(spot, request)).toBeLessThan(-TOLERANCE)
    expect(measureAlong(spot, request)).toBeGreaterThanOrEqual(-SIZE.height / 2 - TOLERANCE)
  })

  it('does not go back past the start where it is given no window', () => {
    const spot = findFreeSpot(index, request)

    expect(measureAlong(spot, request)).toBeGreaterThanOrEqual(-TOLERANCE)
  })
})
