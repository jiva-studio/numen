import { afterEach, describe, expect, it, vi } from 'vitest'
import { resolvePopoverPosition } from './popoverPlacement'

describe('resolvePopoverPosition', () => {
  const popoverSize = { width: 320, height: 160 }
  const containerSize = { width: 1000, height: 800 }

  it('centers horizontally and places above when there is sufficient room', () => {
    const pos = resolvePopoverPosition({
      at: { x: 500, y: 400 },
      popoverSize,
      containerSize,
      margin: 12,
      gap: 12,
    })

    expect(pos).toEqual({
      left: 340, // 500 - 160
      top: 228, // 400 - 160 - 12
    })
  })

  it('clamps to left margin when anchor point is near left boundary', () => {
    const pos = resolvePopoverPosition({
      at: { x: 50, y: 400 },
      popoverSize,
      containerSize,
      margin: 12,
      gap: 12,
    })

    expect(pos.left).toBe(12)
    expect(pos.top).toBe(228)
  })

  it('clamps to right margin when anchor point is near right boundary', () => {
    const pos = resolvePopoverPosition({
      at: { x: 950, y: 400 },
      popoverSize,
      containerSize,
      margin: 12,
      gap: 12,
    })

    expect(pos.left).toBe(668) // 1000 - 320 - 12
    expect(pos.top).toBe(228)
  })

  it('flips below anchor point when there is not enough room above', () => {
    const pos = resolvePopoverPosition({
      at: { x: 500, y: 80 },
      popoverSize,
      containerSize,
      margin: 12,
      gap: 12,
    })

    expect(pos.left).toBe(340)
    expect(pos.top).toBe(92) // 80 + 12
  })

  it('clamps to top/left margins in top-left corner', () => {
    const pos = resolvePopoverPosition({
      at: { x: 30, y: 50 },
      popoverSize,
      containerSize,
      margin: 12,
      gap: 12,
    })

    expect(pos.left).toBe(12)
    expect(pos.top).toBe(62) // 50 + 12
  })

  it('handles narrow container gracefully', () => {
    const pos = resolvePopoverPosition({
      at: { x: 100, y: 100 },
      popoverSize: { width: 400, height: 200 },
      containerSize: { width: 300, height: 400 },
      margin: 12,
      gap: 12,
    })

    expect(pos.left).toBe(12)
  })

  it('clamps in tight vertical space preferring top or bottom half accordingly', () => {
    const tightContainer = { width: 800, height: 200 }
    const largePopover = { width: 300, height: 180 }

    const posTopHalf = resolvePopoverPosition({
      at: { x: 400, y: 50 },
      popoverSize: largePopover,
      containerSize: tightContainer,
    })
    expect(posTopHalf.top).toBe(12)

    const posBottomHalf = resolvePopoverPosition({
      at: { x: 400, y: 150 },
      popoverSize: largePopover,
      containerSize: tightContainer,
    })
    expect(posBottomHalf.top).toBe(12)
  })

  it('uses default dimensions when zero size is provided', () => {
    const pos = resolvePopoverPosition({
      at: { x: 400, y: 300 },
      popoverSize: { width: 0, height: 0 },
      containerSize: { width: 0, height: 0 },
    })
    expect(pos.left).toBeGreaterThan(0)
    expect(pos.top).toBeGreaterThan(0)
  })

  it('centres a popover of its own width on the anchor', () => {
    const pos = resolvePopoverPosition({
      at: { x: 500, y: 400 },
      popoverSize: { width: 240, height: 160 },
      containerSize,
    })

    expect(pos.left).toBe(380)
  })

  it('places above when the room above is exactly the margin', () => {
    const pos = resolvePopoverPosition({ at: { x: 500, y: 184 }, popoverSize, containerSize })

    expect(pos.top).toBe(12)
  })

  it('places below the anchor even where that is inside the margin', () => {
    const pos = resolvePopoverPosition({
      at: { x: 500, y: 4 },
      popoverSize,
      containerSize,
      gap: 0,
    })

    expect(pos.top).toBe(4)
  })

  it('places below when the popover ends exactly at the bottom margin', () => {
    const pos = resolvePopoverPosition({
      at: { x: 400, y: 8 },
      popoverSize: { width: 300, height: 180 },
      containerSize: { width: 800, height: 200 },
      gap: 0,
    })

    expect(pos.top).toBe(8)
  })

  describe('when the popover fits neither above nor below', () => {
    const tall = { width: 320, height: 200 }
    const short = { width: 1000, height: 400 }

    it('raises it to the bottom margin from the upper half', () => {
      const pos = resolvePopoverPosition({
        at: { x: 500, y: 190 },
        popoverSize: tall,
        containerSize: short,
      })

      expect(pos.top).toBe(188)
    })

    it('pins it to the top margin from the middle and the lower half', () => {
      for (const y of [200, 210]) {
        const pos = resolvePopoverPosition({
          at: { x: 500, y },
          popoverSize: tall,
          containerSize: short,
        })

        expect(pos.top).toBe(12)
      }
    })
  })

  describe('without a measured container', () => {
    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it('measures the window', () => {
      vi.stubGlobal('window', { innerWidth: 900, innerHeight: 500 })

      const pos = resolvePopoverPosition({
        at: { x: 880, y: 100 },
        popoverSize: { width: 0, height: 400 },
        containerSize: { width: 0, height: 0 },
      })

      expect(pos).toEqual({ left: 568, top: 88 })
    })

    it('assumes 800 by 600 where there is no window', () => {
      vi.stubGlobal('window', undefined)

      const pos = resolvePopoverPosition({
        at: { x: 790, y: 180 },
        popoverSize: { width: 0, height: 400 },
        containerSize: { width: 0, height: 0 },
      })

      expect(pos).toEqual({ left: 468, top: 188 })
    })
  })
})
