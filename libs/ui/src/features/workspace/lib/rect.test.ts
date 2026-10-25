/** Places measured against the box they sit in. */
import { describe, expect, it } from 'vitest'
import { getPlaceInBox } from './rect'

describe('getPlaceInBox', () => {
  it('measures from the corner of the box', () => {
    const box = { x: 100, y: 40, width: 300, height: 200 }
    expect(getPlaceInBox({ x: 130, y: 90 }, box)).toStrictEqual({ x: 30, y: 50 })
  })
})
