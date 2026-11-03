import { afterEach, describe, expect, it, vi } from 'vitest'
import type { MenuItem } from '../items/item'
import {
  createSubmenuAim,
  isPointInBox,
  isPointInSafeTriangle,
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
    expect(isPointInBox({ x: 199, y: 100 }, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInBox({ x: 250, y: 201 }, SUBMENU_RIGHT)).toBe(false)
  })

  it('rejects missing or zero-area boxes', () => {
    expect(isPointInBox({ x: 0, y: 0 }, null)).toBe(false)
    expect(isPointInBox({ x: 10, y: 10 }, { left: 10, right: 10, top: 0, bottom: 20 })).toBe(false)
    expect(isPointInBox({ x: 10, y: 10 }, { left: 0, right: 20, top: 10, bottom: 10 })).toBe(false)
  })
})

describe('isPointInSafeTriangle', () => {
  it('accepts diagonal moves toward a right-side submenu', () => {
    const origin = { x: 100, y: 60 }
    expect(isPointInSafeTriangle({ x: 150, y: 95 }, origin, SUBMENU_RIGHT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 150, y: 50 }, origin, SUBMENU_RIGHT)).toBe(true)
  })

  it('rejects vertical or backward moves when submenu is on the right', () => {
    const origin = { x: 100, y: 60 }
    expect(isPointInSafeTriangle({ x: 100, y: 95 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 90, y: 60 }, origin, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 150, y: 180 }, origin, SUBMENU_RIGHT)).toBe(false)
  })

  it('accepts diagonal moves toward a left-flipped submenu', () => {
    const origin = { x: 260, y: 60 }
    expect(isPointInSafeTriangle({ x: 220, y: 100 }, origin, SUBMENU_LEFT)).toBe(true)
    expect(isPointInSafeTriangle({ x: 270, y: 60 }, origin, SUBMENU_LEFT)).toBe(false)
  })

  it('rejects missing origin, missing submenu, or origin on the submenu edge', () => {
    expect(isPointInSafeTriangle({ x: 150, y: 95 }, null, SUBMENU_RIGHT)).toBe(false)
    expect(isPointInSafeTriangle({ x: 150, y: 95 }, { x: 100, y: 60 }, null)).toBe(false)
    expect(isPointInSafeTriangle({ x: 200, y: 80 }, { x: 200, y: 60 }, SUBMENU_RIGHT)).toBe(false)
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
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 80, clientY: 60 }))
    h.advance(50)
    h.aim.onRowHover(LEAF, new MouseEvent('mouseenter', { clientX: 80, clientY: 90 }))

    h.advance(SUBMENU_OPEN_DELAY_MS)
    vi.advanceTimersByTime(SUBMENU_OPEN_DELAY_MS)
    expect(h.openedIds).toEqual([])
    h.aim.clear()
  })

  it('opens immediately on touch pointer events', () => {
    const h = createAimHarness()
    const touch = new PointerEvent('pointermove', {
      pointerType: 'touch',
      clientX: 80,
      clientY: 60,
    })
    h.aim.onRowHover(PARENT_A, touch)
    expect(h.getActiveId()).toBe('move')
    h.aim.clear()
  })
})

describe('createSubmenuAim safe triangle and close grace', () => {
  it('keeps submenu open on diagonal move inside the safe triangle and switches on rest', () => {
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 100, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')

    h.aim.onRowHover(LEAF, new MouseEvent('mouseenter', { clientX: 150, clientY: 95 }))
    expect(h.getActiveId()).toBe('move')

    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(LEAF, new MouseEvent('mousemove', { clientX: 150, clientY: 95 }))
    expect(h.getActiveId()).toBeNull()
    h.aim.clear()
  })

  it('switches immediately when moving vertically onto another row outside the triangle', () => {
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 100, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')

    h.aim.onRowHover(PARENT_B, new MouseEvent('mouseenter', { clientX: 100, clientY: 110 }))
    expect(h.getActiveId()).toBe('share')
    h.aim.clear()
  })

  it('waits SUBMENU_CLOSE_GRACE_MS after leaving the menu and cancels close on reaching submenu', () => {
    const h = createAimHarness()
    h.aim.onRowHover(PARENT_A, new MouseEvent('mouseenter', { clientX: 100, clientY: 60 }))
    h.advance(SUBMENU_OPEN_DELAY_MS)
    h.aim.onRowHover(PARENT_A, new MouseEvent('mousemove', { clientX: 100, clientY: 60 }))
    h.aim.onSubmenuOpened()

    h.setInsideMenu(false)
    h.aim.onMenuLeave(new MouseEvent('mouseleave', { clientX: 60, clientY: 60 }))
    expect(h.getActiveId()).toBe('move')

    h.advance(100)
    h.setInsideSubmenu(true)
    h.aim.onPointerMove(new MouseEvent('mousemove', { clientX: 220, clientY: 80 }))
    h.advance(SUBMENU_CLOSE_GRACE_MS)
    expect(h.getActiveId()).toBe('move')

    h.setInsideSubmenu(false)
    h.aim.onPointerMove(new MouseEvent('mousemove', { clientX: 40, clientY: 40 }))
    h.advance(SUBMENU_CLOSE_GRACE_MS)
    h.aim.onPointerMove(new MouseEvent('mousemove', { clientX: 35, clientY: 40 }))
    expect(h.getActiveId()).toBeNull()
    expect(h.getCloseCount()).toBe(1)
    h.aim.clear()
  })
})
