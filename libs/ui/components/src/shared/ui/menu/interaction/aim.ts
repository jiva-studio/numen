/**
 * Submenu hover intent.
 */
import type { Position } from '@/shared/lib/math/geometry'
import type { MenuItem } from '../items/item'
import {
  advancePointerTrack,
  isPointInBox,
  isPointInSafeTriangle,
  readEventPoint,
  type MenuBox,
  type PointerTrack,
} from './aimGeometry'
import { SUBMENU_CLOSE_GRACE_MS, SUBMENU_OPEN_DELAY_MS } from './aimTiming'

export interface SubmenuAimOptions {
  readonly getTime: () => number
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

interface PendingAimAction {
  readonly kind: 'open' | 'switch' | 'close'
  readonly item?: MenuItem | undefined
  readonly dueAt: number
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
  if (ctx.pending?.item?.id === item.id && options.getTime() >= ctx.pending.dueAt) {
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
  } else if (ctx.pending?.kind === 'close' && options.getTime() >= ctx.pending.dueAt) {
    ctx.runPendingAction(ctx.pending)
  } else if (ctx.stepTriangle(event) || ctx.pending?.kind !== 'close') {
    ctx.schedule('close', SUBMENU_CLOSE_GRACE_MS)
  }
}

function handlePointerMove(ctx: AimContext, event: MouseEvent): void {
  const { options } = ctx
  if (isReachingSubmenu(event, readEventPoint(event), options)) ctx.clearPending()
  else if (ctx.pending && options.getTime() >= ctx.pending.dueAt) ctx.runPendingAction(ctx.pending)
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
    const action: PendingAimAction = { kind, item, dueAt: options.getTime() + delayMs }
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
