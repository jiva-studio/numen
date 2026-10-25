/**
 * Visible window computation and scroll target calculations.
 */
import { describe, expect, it } from 'vitest'
import { computeScrollTarget, computeVirtualWindow } from './virtual'

describe('computeVirtualWindow', () => {
  it('handles zero items', () => {
    const window = computeVirtualWindow({
      totalCount: 0,
      rowHeight: 24,
      viewportHeight: 240,
      scrollTop: 0,
    })
    expect(window).toEqual({
      startIndex: 0,
      endIndex: 0,
      offsetY: 0,
      totalHeight: 0,
    })
  })

  it('handles a single item', () => {
    const window = computeVirtualWindow({
      totalCount: 1,
      rowHeight: 24,
      viewportHeight: 240,
      scrollTop: 0,
    })
    expect(window).toEqual({
      startIndex: 0,
      endIndex: 1,
      offsetY: 0,
      totalHeight: 24,
    })
  })

  it('falls back to all items when viewport height is zero or negative', () => {
    const window = computeVirtualWindow({
      totalCount: 100,
      rowHeight: 24,
      viewportHeight: 0,
      scrollTop: 500,
    })
    expect(window).toEqual({
      startIndex: 0,
      endIndex: 100,
      offsetY: 0,
      totalHeight: 2400,
    })
  })

  it('falls back to all items when row height is zero or negative', () => {
    const window = computeVirtualWindow({
      totalCount: 100,
      rowHeight: 0,
      viewportHeight: 240,
      scrollTop: 0,
    })
    expect(window).toEqual({
      startIndex: 0,
      endIndex: 100,
      offsetY: 0,
      totalHeight: 0,
    })
  })

  it('computes visible window with overscan at top boundary', () => {
    const window = computeVirtualWindow({
      totalCount: 100,
      rowHeight: 24,
      viewportHeight: 240,
      scrollTop: 0,
      overscan: 5,
    })
    expect(window).toEqual({
      startIndex: 0,
      endIndex: 15,
      offsetY: 0,
      totalHeight: 2400,
    })
  })

  it('computes visible window in the middle of scroll', () => {
    const window = computeVirtualWindow({
      totalCount: 100,
      rowHeight: 24,
      viewportHeight: 240,
      scrollTop: 240,
      overscan: 3,
    })
    expect(window).toEqual({
      startIndex: 7,
      endIndex: 23,
      offsetY: 168,
      totalHeight: 2400,
    })
  })

  it('clamps endIndex at the bottom boundary', () => {
    const window = computeVirtualWindow({
      totalCount: 20,
      rowHeight: 24,
      viewportHeight: 240,
      scrollTop: 360,
      overscan: 5,
    })
    expect(window).toEqual({
      startIndex: 10,
      endIndex: 20,
      offsetY: 240,
      totalHeight: 480,
    })
  })
})

describe('computeScrollTarget', () => {
  const rowHeight = 24
  const viewportHeight = 240

  it('returns current scroll when row is already visible', () => {
    const target = computeScrollTarget({
      targetIndex: 6,
      rowHeight,
      viewportHeight,
      currentScrollTop: 100,
    })
    expect(target).toBe(100)
  })

  it('scrolls up when row is above current viewport', () => {
    const target = computeScrollTarget({
      targetIndex: 2,
      rowHeight,
      viewportHeight,
      currentScrollTop: 200,
    })
    expect(target).toBe(48)
  })

  it('scrolls down when row is below current viewport', () => {
    const target = computeScrollTarget({
      targetIndex: 12,
      rowHeight,
      viewportHeight,
      currentScrollTop: 0,
    })
    expect(target).toBe(72)
  })

  it('returns current scroll when viewport height is zero', () => {
    const target = computeScrollTarget({
      targetIndex: 12,
      rowHeight,
      viewportHeight: 0,
      currentScrollTop: 50,
    })
    expect(target).toBe(50)
  })
})
