/**
 * Submenu hover intent, safe-triangle geometry, and close-grace timing.
 */
import type { Position } from '@/shared/lib/math/geometry'
import type { MenuItem } from '../items/item'

/** Hover rest duration required before opening or switching a submenu. */
export const SUBMENU_OPEN_DELAY_MS = 150

/** Grace duration before closing an open submenu once the pointer leaves the safe triangle. */
export const SUBMENU_CLOSE_GRACE_MS = 300

/** Minimum Manhattan distance in pixels before updating the safe-triangle origin. */
const MIN_ORIGIN_STEP_PX = 2

export interface MenuBox {
  readonly left: number
  readonly right: number
  readonly top: number
  readonly bottom: number
}

export interface SubmenuAimOptions {
  readonly clock: () => number
  readonly getActiveId: () => string | null
  readonly getSubmenuBox: () => MenuBox | null
  readonly isInsideSubmenu: (event: Event) => boolean
  readonly isInsideMenu: (event: Event) => boolean
  readonly onOpen: (item: MenuItem) => void
  readonly onClose: () => void
}

export interface SubmenuAim {
  readonly onRowHover: (item: MenuItem, event?: MouseEvent) => void
  readonly onMenuLeave: (event?: MouseEvent) => void
  readonly onPointerMove: (event: MouseEvent) => void
  readonly onSubmenuOpened: () => void
  readonly clear: () => void
}

export interface PointerTrack {
  readonly origin: Position | null
  readonly current: Position | null
}

interface PendingAimAction {
  readonly kind: 'open' | 'switch' | 'close'
  readonly item?: MenuItem | undefined
  readonly dueAt: number
}

/** Returns true when `point` is within or on the edges of `box` with positive area. */
export function isPointInBox(point: Position, box: MenuBox | null | undefined): boolean {
  if (!box || box.right <= box.left || box.bottom <= box.top) return false
  return point.x >= box.left && point.x <= box.right && point.y >= box.top && point.y <= box.bottom
}

/**
 * Returns true when `point` is inside the triangle from `origin` to the near
 * vertical edge of `submenu` and advances horizontally toward that edge.
 */
export function isPointInSafeTriangle(
  point: Position,
  origin: Position | null,
  submenu: MenuBox | null | undefined,
): boolean {
  if (!origin || !submenu || submenu.right <= submenu.left || submenu.bottom <= submenu.top) {
    return false
  }
  const centerX = (submenu.left + submenu.right) / 2
  const nearX = origin.x <= centerX ? submenu.left : submenu.right
  const spanX = nearX - origin.x
  // Stryker disable next-line ConditionalExpression: division by zero yields Infinity or NaN which fail the range checks below
  if (spanX === 0) return false
  const progress = (point.x - origin.x) / spanX
  if (progress <= 0 || progress > 1) return false
  const minY = origin.y + progress * (submenu.top - origin.y)
  const maxY = origin.y + progress * (submenu.bottom - origin.y)
  return point.y >= minY && point.y <= maxY
}

export function readEventPoint(event?: MouseEvent): Position | null {
  if (!event || (event.clientX === 0 && event.clientY === 0)) return null
  return { x: event.clientX, y: event.clientY }
}

export function advancePointerTrack(track: PointerTrack, point: Position | null): PointerTrack {
  if (!point) return track
  if (!track.current) return { origin: null, current: point }
  const step = Math.abs(point.x - track.current.x) + Math.abs(point.y - track.current.y)
  if (step < MIN_ORIGIN_STEP_PX) return track
  return { origin: track.current, current: point }
}

function canOpenSubmenu(item: MenuItem): boolean {
  return !item.isDisabled && Boolean(item.children && item.children.length > 0)
}

function applyRowTarget(item: MenuItem, options: SubmenuAimOptions): void {
  if (canOpenSubmenu(item)) {
    options.onOpen(item)
    return
  }
  const activeId = options.getActiveId()
  if (activeId !== null && activeId !== item.id) {
    options.onClose()
  }
}

function isReachingSubmenu(
  event: MouseEvent | undefined,
  point: Position | null,
  options: SubmenuAimOptions,
): boolean {
  return Boolean(
    (event && options.isInsideSubmenu(event)) ||
    (point && isPointInBox(point, options.getSubmenuBox())),
  )
}

interface AimContext {
  readonly options: SubmenuAimOptions
  track: PointerTrack
  pending: PendingAimAction | null
  timerHandle: ReturnType<typeof setTimeout> | undefined
  readonly syncWindow: (shouldBind: boolean) => void
  readonly clearPending: () => void
  readonly runPendingAction: (action: PendingAimAction) => void
  readonly schedule: (kind: PendingAimAction['kind'], delayMs: number, item?: MenuItem) => void
  readonly stepTriangle: (event?: MouseEvent) => boolean
}

function createWindowSync(listener: (event: MouseEvent) => void): (shouldBind: boolean) => void {
  let isBound = false
  return (shouldBind: boolean): void => {
    if (shouldBind === isBound || typeof window === 'undefined') return
    isBound = shouldBind
    if (shouldBind) {
      window.addEventListener('mousemove', listener)
      window.addEventListener('pointermove', listener)
    } else {
      window.removeEventListener('mousemove', listener)
      window.removeEventListener('pointermove', listener)
    }
  }
}

function isTouchHover(event?: MouseEvent): boolean {
  return event instanceof PointerEvent && event.pointerType === 'touch'
}

function handleClosedHover(ctx: AimContext, item: MenuItem): void {
  if (!canOpenSubmenu(item)) {
    ctx.clearPending()
    return
  }
  // Stryker disable next-line OptionalChaining,StringLiteral: pending actions while closed always carry an item and runPendingAction branches on close vs item
  if (ctx.pending?.item?.id !== item.id) ctx.schedule('open', SUBMENU_OPEN_DELAY_MS, item)
}

function handleRowHover(ctx: AimContext, item: MenuItem, event?: MouseEvent): void {
  const { options } = ctx
  if (isTouchHover(event)) {
    ctx.clearPending()
    applyRowTarget(item, options)
    return
  }
  if (ctx.pending?.item?.id === item.id && options.clock() >= ctx.pending.dueAt) {
    ctx.stepTriangle(event)
    ctx.runPendingAction(ctx.pending)
    return
  }
  const isInTriangle = ctx.stepTriangle(event)
  const activeId = options.getActiveId()
  if (activeId === null) {
    handleClosedHover(ctx, item)
    return
  }
  if (activeId === item.id) {
    ctx.clearPending()
    return
  }
  if (isInTriangle) {
    // Stryker disable next-line StringLiteral: runPendingAction branches on close vs item
    ctx.schedule('switch', SUBMENU_OPEN_DELAY_MS, item)
    return
  }
  ctx.clearPending()
  applyRowTarget(item, options)
}

function handleMenuLeave(ctx: AimContext, event?: MouseEvent): void {
  const { options } = ctx
  if (options.getActiveId() === null || isReachingSubmenu(event, readEventPoint(event), options)) {
    ctx.clearPending()
  } else if (ctx.pending?.kind === 'close' && options.clock() >= ctx.pending.dueAt) {
    ctx.runPendingAction(ctx.pending)
  } else if (ctx.stepTriangle(event) || ctx.pending?.kind !== 'close') {
    ctx.schedule('close', SUBMENU_CLOSE_GRACE_MS)
  }
}

function handlePointerMove(ctx: AimContext, event: MouseEvent): void {
  const { options } = ctx
  if (isReachingSubmenu(event, readEventPoint(event), options)) ctx.clearPending()
  else if (ctx.pending && options.clock() >= ctx.pending.dueAt) ctx.runPendingAction(ctx.pending)
  else if (!options.isInsideMenu(event) && options.getActiveId() !== null) {
    if (ctx.stepTriangle(event) || ctx.pending?.kind !== 'close') {
      ctx.schedule('close', SUBMENU_CLOSE_GRACE_MS)
    }
  }
}

export function createSubmenuAim(options: SubmenuAimOptions): SubmenuAim {
  const onPointerMove = (event: MouseEvent): void => handlePointerMove(ctx, event)
  const syncWindow = createWindowSync(onPointerMove)

  const clearPending = (): void => {
    if (ctx.timerHandle !== undefined) {
      clearTimeout(ctx.timerHandle)
      ctx.timerHandle = undefined
    }
    ctx.pending = null
    if (options.getActiveId() === null) syncWindow(false)
  }

  const runPendingAction = (action: PendingAimAction): void => {
    clearPending()
    if (action.kind === 'close') {
      options.onClose()
      return
    }
    // Stryker disable next-line ConditionalExpression: non-close actions always carry an item
    if (action.item) applyRowTarget(action.item, options)
  }

  const schedule = (kind: PendingAimAction['kind'], delayMs: number, item?: MenuItem): void => {
    clearPending()
    const action: PendingAimAction = { kind, item, dueAt: options.clock() + delayMs }
    ctx.pending = action
    syncWindow(true)
    ctx.timerHandle = setTimeout(() => {
      ctx.timerHandle = undefined
      // Stryker disable next-line ConditionalExpression,EqualityOperator: clearPending cancels the timeout whenever pending changes
      if (ctx.pending === action) runPendingAction(action)
    }, delayMs)
  }

  const stepTriangle = (event?: MouseEvent): boolean => {
    const point = readEventPoint(event)
    if (!point) return false
    ctx.track = advancePointerTrack(ctx.track, point)
    return isPointInSafeTriangle(point, ctx.track.origin, options.getSubmenuBox())
  }

  const ctx: AimContext = {
    options,
    // Stryker disable next-line ObjectLiteral: null and undefined fields on PointerTrack are both falsy
    track: { origin: null, current: null },
    pending: null,
    timerHandle: undefined,
    syncWindow,
    clearPending,
    runPendingAction,
    schedule,
    stepTriangle,
  }

  return {
    onRowHover: (item, event) => handleRowHover(ctx, item, event),
    onMenuLeave: (event) => handleMenuLeave(ctx, event),
    onPointerMove,
    onSubmenuOpened: () => {
      clearPending()
      syncWindow(true)
    },
    clear: () => {
      clearPending()
      syncWindow(false)
      // Stryker disable next-line ObjectLiteral: null and undefined fields on PointerTrack are both falsy
      ctx.track = { origin: null, current: null }
    },
  }
}
