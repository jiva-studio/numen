/**
 * The drag is arithmetic, so it is checked without a pointer.
 *
 * A window is a number handed in: what the drag works out is the same on any
 * machine, and a short window is what the numbers are awkward at.
 */
import { describe, expect, it } from 'vitest'
import { getDraggedHeight, getTallest, LEAST } from './height'

const TALL = 900

describe('how tall the player is dragged', () => {
  it('follows the pointer between the two ends', () => {
    expect(getDraggedHeight(300, 50, TALL)).toBe(350)
    expect(getDraggedHeight(300, -50, TALL)).toBe(250)
  })

  it('stops at the shortest it is drawn', () => {
    expect(getDraggedHeight(LEAST, -1000, TALL)).toBe(LEAST)
  })

  it('stops at what the window has room for', () => {
    expect(getDraggedHeight(300, 1000, TALL)).toBe(getTallest(TALL))
  })

  it('is the shortest it is drawn in a window with no room at all', () => {
    expect(getTallest(0)).toBe(LEAST)
    expect(getDraggedHeight(300, 1000, 0)).toBe(LEAST)
  })

  it('leaves the page room above and below', () => {
    expect(getTallest(TALL)).toBeLessThan(TALL)
  })
})
