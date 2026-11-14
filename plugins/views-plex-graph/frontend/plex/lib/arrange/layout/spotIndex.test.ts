/** Whether a box keeps the gaps from the boxes already placed. */
import { describe, expect, it } from 'vitest'
import { SpotIndex, type Spot } from './spotIndex'
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
})
