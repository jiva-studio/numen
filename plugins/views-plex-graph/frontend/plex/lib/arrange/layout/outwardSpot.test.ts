/** A free spot found by walking outward never lies behind the start of its group. */
import { describe, expect, it } from 'vitest'
import { findFreeSpot, TOLERANCE } from './freeSpot'

const SIZE = { width: 144, height: 60 }
const GAPS = { gap: 16, lineGap: 20 }
const ORIGIN = { x: 0, y: 0 }

describe('a spot found when the start is occupied', () => {
  it.each([30, 45, 60, 75, 120, 150, 210, 300])(
    'is not behind the start along a direction of %i degrees',
    (degrees) => {
      const radians = (degrees * Math.PI) / 180
      const direction = { x: Math.cos(radians), y: Math.sin(radians) }

      const spot = findFreeSpot([{ ...ORIGIN, ...SIZE }], {
        start: ORIGIN,
        middle: ORIGIN,
        direction,
        size: SIZE,
        group: [],
        gaps: GAPS,
      })

      expect(spot.x * direction.x + spot.y * direction.y).toBeGreaterThanOrEqual(-TOLERANCE)
    },
  )
})
