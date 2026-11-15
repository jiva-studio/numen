import type { Position } from '@/shared/lib/math/geometry'

/** Minimum Manhattan distance in pixels before updating the safe-triangle origin. */
const MIN_ORIGIN_STEP_PX = 2

export interface MenuBox {
  readonly left: number
  readonly right: number
  readonly top: number
  readonly bottom: number
}

export interface PointerTrack {
  readonly origin: Position | null
  readonly current: Position | null
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
