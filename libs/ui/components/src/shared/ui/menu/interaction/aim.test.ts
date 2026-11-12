import { afterEach, describe, expect, it, vi } from 'vitest'
import type { MenuItem } from '../items/item'
import {
  advancePointerTrack,
  createSubmenuAim,
  isPointInBox,
  isPointInSafeTriangle,
  readEventPoint,
  SUBMENU_CLOSE_GRACE_MS,
  SUBMENU_OPEN_DELAY_MS,
  type MenuBox,
} from './aim'

const PARENT_A: MenuItem = {
  id: 'move',
  text: 'Move to',
  children: [{ id: 'inbox', text: 'Inbox' }],
}

const PARENT_B: MenuItem = {
  id: 'share',
  text: 'Share with',
  children: [{ id: 'team', text: 'Team' }],
}

const LEAF: MenuItem = { id: 'copy', text: 'Copy path' }

const SUBMENU_RIGHT: MenuBox = { left: 200, right: 360, top: 40, bottom: 200 }
const SUBMENU_LEFT: MenuBox = { left: 20, right: 180, top: 40, bottom: 200 }

afterEach(() => {
  vi.useRealTimers()
})

describe('isPointInBox', () => {
  it('accepts points inside and on edges of a positive-area box', () => {
    expect(isPointInBox({ x: 250, y: 100 }, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInBox({ x: 200, y: 40 }, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInBox({ x: 360, y: 200 }, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInBox({ x: 199, y: 100 }, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInBox({ x: 361, y: 100 }, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInBox({ x: 250, y: 39 }, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInBox({ x: 250, y: 201 }, SUBMENU_RIGHT)).toBe(false)
  })

  it('rejects missing or zero-area boxes', () => {
    expect(isPointInBox({ x: 0, y: 0 }, null)).toBe(false)
    expect(isPointInBox({ x: 10, y: 10 }, { left: 10, right: 10, top: 0, bottom: 20 })).toBe(false)
    expect(isPointInBox({ x: 10, y: 10 }, { left: 20, right: 10, top: 0, bottom: 20 })).toBe(false)
    expect(isPointInBox({ x: 10, y: 10 }, { left: 0, right: 20, top: 10, bottom: 10 })).toBe(false)
    expect(isPointInBox({ x: 10, y: 10 }, { left: 0, right: 20, top: 20, bottom: 10 })).toBe(false)
  })
})

describe('isPointInSafeTriangle', () => {
  it('accepts diagonal moves toward a right-side submenu including boundary edges', () => {
    const origin = { x: 100, y: 60 }
    expect(isPointInSafeTriangle({ x: 150, y: 95 }, origin, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 150, y: 50 }, origin, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 150, y: 130 }, origin, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 200, y: 100 }, origin, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 150, y: 49 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 150, y: 131 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 201, y: 100 }, origin, SUBMENU_RIGHT)).toBe(false)
  })

  it('rejects vertical or backward moves when submenu is on the right', () => {
    const origin = { x: 100, y: 60 }
    expect(isPointInSafeTriangle({ x: 100, y: 60 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 100, y: 95 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 90, y: 60 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 150, y: 180 }, origin, SUBMENU_RIGHT)).toBe(false)
  })

  it('accepts diagonal moves toward a left-flipped submenu and from centerX', () => {
    const origin = { x: 260, y: 60 }
    expect(isPointInSafeTriangle({ x: 220, y: 100 }, origin, SUBMENU_LEFT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 270, y: 60 }, origin, SUBMENU_LEFT)).toBe(false)
    // Origin exactly at centerX (280 for SUBMENU_RIGHT) aims toward submenu.left (200)
    expect(isPointInSafeTriangle({ x: 240, y: 80 }, { x: 280, y: 60 }, SUBMENU_RIGHT)).toBe(true)
  })

  it('rejects missing origin, missing or zero-area submenu, or origin on the submenu edge', () => {
    expect(isPointInSafeTriangle({ x: 150, y: 95 }, null, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 150, y: 95 }, { x: 100, y: 60 }, null)).toBe(false)
    expect(
      isPointInSafeTriangle(
        { x: 150, y: 95 },
        { x: 100, y: 60 },
        { left: 200, right: 200, top: 40, bottom: 200 },
      ),
    ).toBe(false)
    expect(
      isPointInSafeTriangle(
        { x: 150, y: 130 },
        { x: 100, y: 60 },
        { left: 200, right: 360, top: 200, bottom: 200 },
      ),
    ).toBe(false)
    expect(
      isPointInSafeTriangle(
        { x: 150, y: 130 },
        { x: 100, y: 60 },
        { left: 200, right: 360, top: 200, bottom: 100 },
      ),
    ).toBe(false)
    expect(isPointInSafeTriangle({ x: 200, y: 80 }, { x: 200, y: 60 }, SUBMENU_RIGHT)).toBe(false)
  })

  it('reads pointer coordinates and advances the safe-triangle pointer track', () => {
    expect(readEventPoint(undefined)).toBeNull()
    expect(readEventPoint(new MouseEvent('mousemove', { clientX: 0, clientY: 0 }))).toBeNull()
    expect(readEventPoint(new MouseEvent('mousemove', { clientX: 0, clientY: 15 }))).toEqual({
      x: 0,
      y: 15,
    })
    expect(readEventPoint(new MouseEvent('mousemove', { clientX: 15, clientY: 0 }))).toEqual({
      x: 15,
      y: 0,
    })

    const initial = { origin: null, current: null }
    expect(advancePointerTrack(initial, null)).toBe(initial)
    const first = advancePointerTrack(initial, { x: 10, y: 20 })
    expect(first).toEqual({ origin: null, current: { x: 10, y: 20 } })
    expect(advancePointerTrack(first, null)).toBe(first)
    expect(advancePointerTrack(first, { x: 11, y: 20 })).toBe(first)
    expect(advancePointerTrack(first, { x: 10, y: 22 })).toEqual({
      origin: { x: 10, y: 20 },
      current: { x: 10, y: 22 },
    })
    expect(advancePointerTrack(first, { x: 9, y: 19 })).toEqual({
      origin: { x: 10, y: 20 },
      current: { x: 9, y: 19 },
    })
  })
})

function createAimHarness() {
  let now = 1_000
  let activeId: string | null = null
  let submenuBox: MenuBox | null = SUBMENU_RIGHT
  let insideSubmenu = false
  let insideMenu = true
  const openedIds: string[] = []
  let closeCount = 0

  const aim = createSubmenuAim({
    clock: () => now,
    getActiveId: () => activeId,
    getSubmenuBox: () => submenuBox,
    isInsideSubmenu: () => insideSubmenu,
    isInsideMenu: () => insideMenu,
    onOpen: (item) => {
      activeId = item.id
      openedIds.push(item.id)
    },
    onClose: () => {
      activeId = null
      closeCount += 1
    },
  })

  return {
    aim,
    openedIds,
    getActiveId: () => activeId,
    getCloseCount: () => closeCount,
    advance: (ms: number) => {
      now += ms
    },
    setSubmenuBox: (box: MenuBox | null) => {
      submenuBox = box
    },
    setInsideSubmenu: (val: boolean) => {
      insideSubmenu = val
    },
    setInsideMenu: (val: boolean) => {
      insideMenu = val
    },
  }
}

describe('createSubmenuAim open delay and hover intent', () => {
  it('opens a submenu only after resting for SUBMENU_OPEN_DELAY_MS', () => {
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 80, clientY: 60 }))
    expect(h.getActiveId()).toBeNull()

    h.advance(SUBMENU_OPEN_DELAY_MS - 10)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 82, clientY: 60 }))
    expect(h.getActiveId()).toBeNull()

    h.advance(10)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 84, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')
    h.aim.clear()
  })

  it('cancels pending open when passing onto a leaf row before the delay elapses', () => {
    vi.useFakeTimers()
    const addSpy = vi.spyOn(window, 'addEventListener')
    const removeSpy = vi.spyOn(window, 'removeEventListener')
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 80, clientY: 60 }))
    expect(addSpy).toHaveBeenCalledTimes(2)
    h.advance(50)
    h.aim.onRowHover(LEAF, new MouseEvent('mouseenter', { clientX: 80, clientY: 90 }))
    expect(addSpy).toHaveBeenCalledTimes(2)
    expect(removeSpy).toHaveBeenCalledWith('mousemove', expect.any(Function))
    expect(removeSpy).toHaveBeenCalledWith('pointermove', expect.any(Function))

    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.openedIds).toEqual([])
    h.aim.clear()
    addSpy.mockRestore()
    removeSpy.mockRestore()
  })

  it('opens immediately on touch pointer events and waits on mouse pointer events', () => {
    vi.useFakeTimers()
    const h = createAimHarness()
    const mousePtr = new PointerEvent('pointermove', {
      pointerType: 'mouse',
      clientX: 80,
      clientY: 60,
    })
    h.aim.onRowHover(PARENT_A, mousePtr)
    expect(h.getActiveId()).toBeNull()

    // Touching PARENT_B cancels the pending mouse open on PARENT_A
    const touchB = new PointerEvent('pointermove', {
      pointerType: 'touch',
      clientX: 80,
      clientY: 75,
    })
    h.aim.onRowHover(PARENT_B, touchB)
    expect(h.getActiveId()).toBe('share')
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBe('share')

    // Hovering a leaf on touch closes the open submenu; hovering again when closed does not call onClose
    const touchLeaf = new PointerEvent('pointermove', {
      pointerType: 'touch',
      clientX: 80,
      clientY: 90,
    })
    h.aim.onRowHover(LEAF, touchLeaf)
    expect(h.getCloseCount()).toBe(1)
    h.aim.onRowHover(LEAF, touchLeaf)
    expect(h.getCloseCount()).toBe(1)

    const touch = new PointerEvent('pointermove', {
      pointerType: 'touch',
      clientX: 80,
      clientY: 60,
    })
    h.aim.onRowHover(PARENT_A, touch)
    expect(h.getActiveId()).toBe('move')

    // Touching the already-open item when disabled does not close it
    h.aim.onRowHover({ ...PARENT_A, isDisabled: true }, touch)
    expect(h.getCloseCount()).toBe(1)
    h.aim.clear()
  })

  it('fires the scheduled open via setTimeout and does not reset delay on repeated hover of same item', () => {
    vi.useFakeTimers()
    const clearSpy = vi.spyOn(globalThis, 'clearTimeout')
    const addSpy = vi.spyOn(window, 'addEventListener')
    const removeSpy = vi.spyOn(window, 'removeEventListener')
    const h = createAimHarness()
    // Calling clear when no timer is active does not invoke clearTimeout or removeEventListener
    h.aim.clear()
    expect(clearSpy).not.toHaveBeenCalled()
    expect(removeSpy).not.toHaveBeenCalled()

    // Opening submenu via touch and calling onSubmenuOpened twice binds mousemove and pointermove once
    h.aim.onRowHover(PARENT_A, new PointerEvent('pointermove', { pointerType: 'touch' }))
    h.aim.onSubmenuOpened()
    expect(addSpy).toHaveBeenCalledWith('mousemove', expect.any(Function))
    expect(addSpy).toHaveBeenCalledWith('pointermove', expect.any(Function))
    expect(addSpy).toHaveBeenCalledTimes(2)
    h.aim.onSubmenuOpened()
    expect(addSpy).toHaveBeenCalledTimes(2)
    h.aim.clear()
    expect(removeSpy).toHaveBeenCalledWith('mousemove', expect.any(Function))
    expect(removeSpy).toHaveBeenCalledWith('pointermove', expect.any(Function))
    h.aim.onRowHover(LEAF, new PointerEvent('pointermove', { pointerType: 'touch' }))

    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 80, clientY: 60 }))
    h.advance(100)
    vi.advanceTimersByTime(100)
    // Hovering the same item again before dueAt preserves the original timer
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 82, clientY: 60 }))
    expect(clearSpy).not.toHaveBeenCalled()

    // Let setTimeout fire for PARENT_A (timerHandle is cleared before runPending so clearTimeout is not called)
    h.advance(50)
    vi.advanceTimersByTime(50)
    expect(h.getActiveId()).toBe('move')
    expect(clearSpy).not.toHaveBeenCalled()

    // Timer already fired, so clear() does not call clearTimeout
    h.aim.clear()
    expect(clearSpy).not.toHaveBeenCalled()
    h.aim.onRowHover(LEAF, new PointerEvent('pointermove', { pointerType: 'touch' }))

    // Switching from PARENT_A to PARENT_B while closed cancels PARENT_A's timer via schedule -> clearPending
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 80, clientY: 60 }))
    h.aim.onRowHover(PARENT_B, new MouseEvent('mouseenter', { clientX: 80, clientY: 75 }))
    expect(clearSpy).toHaveBeenCalledTimes(1)

    // Completing PARENT_B open synchronously via onRowHover cancels its timer via runPending -> clearPending
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_B, new MouseEvent('mousemove', { clientX: 84, clientY: 75 }))
    expect(clearSpy).toHaveBeenCalledTimes(2)
    expect(h.getActiveId()).toBe('share')
    h.aim.onRowHover(LEAF, new PointerEvent('pointermove', { pointerType: 'touch' }))

    // Scheduling a timer and calling clear() before it fires calls clearTimeout
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 80, clientY: 60 }))
    h.aim.clear()
    expect(clearSpy).toHaveBeenCalledTimes(3)

    // Items with empty children cannot open a submenu
    h.aim.onRowHover({ id: 'empty', text: 'Empty', children: [] })
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBeNull()
    clearSpy.mockRestore()
    addSpy.mockRestore()
    removeSpy.mockRestore()
  })
})

describe('createSubmenuAim safe triangle and close grace', () => {
  it('keeps submenu open on diagonal move inside the safe triangle and switches on rest', () => {
    vi.useFakeTimers()
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 250, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    // Completing pending open at x=120 updates the triangle current to (120, 60)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 120, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')
    expect(h.openedIds).toEqual(['move'])

    // Moving immediately into triangle onto LEAF schedules switch (relies on stepTriangle from line above)
    h.aim.onRowHover(LEAF, new MouseEvent('mouseenter', { clientX: 150, clientY: 95 }))
    expect(h.getActiveId()).toBe('move')

    // Returning to active row cancels the pending switch without re-calling onOpen
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 122, clientY: 60 }))
    expect(h.openedIds).toEqual(['move'])
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBe('move')

    // Moving into triangle onto LEAF again and resting closes the submenu
    h.aim.onRowHover(LEAF, new MouseEvent('mouseenter', { clientX: 150, clientY: 95 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(LEAF, new MouseEvent('mousemove', { clientX: 150, clientY: 95 }))
    expect(h.getActiveId()).toBeNull()

    // Hovering another row without an event when origin is non-null returns false from stepTriangle and switches immediately
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 120, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBe('move')
    h.aim.onRowHover(PARENT_B)
    expect(h.getActiveId()).toBe('share')
    h.aim.clear()
  })

  it('switches immediately when moving vertically onto another row outside the triangle', () => {
    vi.useFakeTimers()
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 120, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')

    // Schedule a switch to LEAF inside triangle, then move vertically outside triangle to PARENT_B
    h.aim.onRowHover(LEAF, new MouseEvent('mouseenter', { clientX: 150, clientY: 95 }))
    h.aim.onRowHover(PARENT_B, new MouseEvent('mouseenter', { clientX: 100, clientY: 10 }))
    expect(h.getActiveId()).toBe('share')

    // Advancing past the old LEAF switch delay does not close PARENT_B
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBe('share')
    h.aim.clear()
  })

  it('handles zero coordinates, menu leave, window listeners, and close grace', () => {
    vi.useFakeTimers()
    const h = createAimHarness()

    // When activeId is null, onMenuLeave cancels a pending open without scheduling a close
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 10, clientY: 10 }))
    h.advance(SUBMENU_CLOSE_GRACE_MS)
    vi.advanceTimersByTime(SUBMENU_CLOSE_GRACE_MS)
    expect(h.getActiveId()).toBeNull()
    expect(h.getCloseCount()).toBe(0)

    // When activeId is null and window listener is active during pending open, pointermove outside menu does not overwrite pending open with close
    h.setInsideMenu(false)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 10, clientY: 10 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBe('move')
    expect(h.getCloseCount()).toBe(0)
    h.setInsideMenu(true)
    h.aim.onRowHover(LEAF, new PointerEvent('pointermove', { pointerType: 'touch' }))
    expect(h.getCloseCount()).toBe(1)

    // Open submenu and establish pointer track (100, 60) -> (120, 60)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 120, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')

    // Schedule switch to PARENT_B, then call onSubmenuOpened to verify it clears pending switch
    h.aim.onRowHover(PARENT_B, new MouseEvent('mousemove', { clientX: 150, clientY: 95 }))
    h.aim.onSubmenuOpened()
    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.getActiveId()).toBe('move')

    // Leaving menu into submenu box (while isInsideSubmenu is false) cancels pending and does not schedule close
    h.setInsideMenu(false)
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 250, clientY: 100 }))
    h.advance(SUBMENU_CLOSE_GRACE_MS)
    vi.advanceTimersByTime(SUBMENU_CLOSE_GRACE_MS)
    expect(h.getActiveId()).toBe('move')

    // Leaving menu outside submenu schedules close; re-entering onto active row cancels pending close cleanly
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 50, clientY: 50 }))
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 120, clientY: 60 }))
    h.advance(SUBMENU_CLOSE_GRACE_MS)
    vi.advanceTimersByTime(SUBMENU_CLOSE_GRACE_MS)
    expect(h.getActiveId()).toBe('move')

    // Leaving menu outside submenu schedules close; a second onMenuLeave outside triangle before dueAt does NOT reset timer
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 50, clientY: 50 }))
    h.advance(200)
    vi.advanceTimersByTime(200)
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 40, clientY: 50 }))
    h.advance(100)
    vi.advanceTimersByTime(100)
    expect(h.getActiveId()).toBeNull()

    // Reopen, establish track (100, 60) -> (120, 60), leave menu, and refresh close grace inside triangle via onMenuLeave
    h.setInsideMenu(true)
    h.aim.onRowHover(PARENT_B, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_B, new MouseEvent('mousemove', { clientX: 120, clientY: 60 }))
    expect(h.getActiveId()).toBe('share')
    h.setInsideMenu(false)
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 140, clientY: 95 }))
    h.advance(200)
    vi.advanceTimersByTime(200)
    // Second onMenuLeave inside triangle while pending.kind === 'close' refreshes the 300ms grace timer
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 160, clientY: 95 }))
    h.advance(200)
    vi.advanceTimersByTime(200)
    expect(h.getActiveId()).toBe('share')
    // Once clock >= dueAt, onMenuLeave executes the pending close immediately
    h.advance(100)
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 30, clientY: 50 }))
    expect(h.getActiveId()).toBeNull()

    // Reopen and verify window pointermove outside triangle does NOT reset close grace
    h.setInsideMenu(true)
    h.aim.onRowHover(PARENT_A, new PointerEvent('pointermove', { pointerType: 'touch' }))
    expect(h.getActiveId()).toBe('move')
    h.aim.onSubmenuOpened()
    h.setInsideMenu(false)
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 50, clientY: 50 }))
    h.advance(200)
    vi.advanceTimersByTime(200)
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 40, clientY: 50 }))
    h.advance(100)
    vi.advanceTimersByTime(100)
    expect(h.getActiveId()).toBeNull()

    // Reopen, establish track (100, 60) -> (120, 60), and verify window pointermove inside triangle refreshes close grace
    h.setInsideMenu(true)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 120, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')
    h.aim.onSubmenuOpened()
    h.setInsideMenu(false)
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 140, clientY: 95 }))
    h.advance(200)
    vi.advanceTimersByTime(200)
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 160, clientY: 95 }))
    h.advance(200)
    vi.advanceTimersByTime(200)
    expect(h.getActiveId()).toBe('move')
    // Moving into submenu box via window event cancels close
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 250, clientY: 100 }))
    h.advance(SUBMENU_CLOSE_GRACE_MS)
    vi.advanceTimersByTime(SUBMENU_CLOSE_GRACE_MS)
    expect(h.getActiveId()).toBe('move')

    // clear() resets pointer track so a single subsequent move has no triangle origin
    h.aim.clear()
    h.aim.onRowHover(PARENT_B, new MouseEvent('mousemove', { clientX: 150, clientY: 95 }))
    expect(h.getActiveId()).toBe('share')

    // Works safely when window is undefined
    vi.stubGlobal('window', undefined)
    expect(() => h.aim.onSubmenuOpened()).not.toThrow()
    expect(() => h.aim.clear()).not.toThrow()
    vi.unstubAllGlobals()
  })
})
