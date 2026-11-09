import { describe, expect, it } from 'vitest'
import { findDropPlace } from './dropPlace'

const BOXES = [
  { key: 'org.example.reef/stats', left: 100, right: 180 },
  { key: 'org.example.tide/status', left: 184, right: 264 },
]

describe('the place of a drop in a slot', () => {
  it('is before the entry whose midpoint lies right of the pointer', () => {
    expect(findDropPlace(BOXES, 120, 100)).toStrictEqual({ before: 'org.example.reef/stats', x: 100 })
    expect(findDropPlace(BOXES, 160, 100)).toStrictEqual({
      before: 'org.example.tide/status',
      x: 184,
    })
  })

  it('is the end of the slot right of the midpoint of the last entry', () => {
    expect(findDropPlace(BOXES, 250, 100)).toStrictEqual({ before: undefined, x: 264 })
  })

  it('is the left edge of an empty slot', () => {
    expect(findDropPlace([], 40, 12)).toStrictEqual({ before: undefined, x: 12 })
  })
})
