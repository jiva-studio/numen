/**
 * What letting go of a dragged tab evaluates to, as geometry alone.
 *
 * A point and a box in, a side out. Which pane that box belongs to and what
 * the side then does to the tree are settled elsewhere; `getRect` is the one
 * thing here that measures anything.
 */
import type { NodeId, Side } from './node'
import { getPlaceInBox, type Position, type Rect } from './rect'

/**
 * The place a tab is dropped at if released now, and the part of the screen that
 * marks it.
 */
export type TabDropTarget = { readonly box: Rect } & (
  | { readonly kind: 'edge'; readonly side: Side }
  | { readonly kind: 'pane'; readonly pane: NodeId; readonly side: Side }
  | { readonly kind: 'strip'; readonly pane: NodeId; readonly slot: number }
)

export interface DropOptions {
  /** How much of a box each edge zone takes, along its own axis. */
  readonly share: number
  /** The most an edge zone takes, in the units the box is measured in. */
  readonly limit: number
}

export const DEFAULT_DROP: DropOptions = { share: 0.2, limit: 96 }

/**
 * The side that a place requests.
 *
 * An edge wins by how far into its own zone the place has come, so a corner
 * goes to whichever edge it is deeper inside. A place in no zone lands in the
 * middle, and the tab joins the stack.
 */
export function getSide(point: Position, box: Rect, options: Partial<DropOptions> = {}): Side {
  const { share, limit } = { ...DEFAULT_DROP, ...options }
  const { x, y } = getPlaceInBox(point, box)

  const acrossZone = Math.min(box.width * share, limit)
  const downZone = Math.min(box.height * share, limit)

  const reaches: readonly (readonly [Side, number])[] = [
    ['left', acrossZone > 0 ? x / acrossZone : Infinity],
    ['right', acrossZone > 0 ? (box.width - x) / acrossZone : Infinity],
    ['top', downZone > 0 ? y / downZone : Infinity],
    ['bottom', downZone > 0 ? (box.height - y) / downZone : Infinity],
  ]

  let selected: Side = 'center'
  let deepest = 1
  for (const [side, reach] of reaches) {
    if (reach < deepest) {
      selected = side
      deepest = reach
    }
  }
  return selected
}

/** The part of a box a side would take, for the overlay that shows it. */
export function getOverlay(side: Side, box: Rect): Rect {
  const half = { width: box.width / 2, height: box.height / 2 }
  switch (side) {
    case 'left':
      return { x: box.x, y: box.y, width: half.width, height: box.height }
    case 'right':
      return { x: box.x + half.width, y: box.y, width: half.width, height: box.height }
    case 'top':
      return { x: box.x, y: box.y, width: box.width, height: half.height }
    case 'bottom':
      return { x: box.x, y: box.y + half.height, width: box.width, height: half.height }
    case 'center':
      return box
  }
}

/**
 * The place in a strip of tabs that a point requests, counted in gaps: zero is
 * before the first tab, and the length of the strip is after the last. A tab
 * is passed once the point is beyond its middle.
 */
export function getSlot(x: number, tabs: readonly Rect[]): number {
  for (const [i, tab] of tabs.entries()) {
    if (x < tab.x + tab.width / 2) return i
  }
  return tabs.length
}

/**
 * The gap a tab would take in a strip, as a box of no width. A strip holding
 * no tabs takes the whole of itself.
 */
export function getCaretRect(slot: number, tabs: readonly Rect[], strip: Rect): Rect {
  const adjacentElement = tabs[slot] ?? tabs[tabs.length - 1]
  if (!adjacentElement) return strip

  const caretX = slot < tabs.length ? adjacentElement.x : adjacentElement.x + adjacentElement.width
  return { x: caretX, y: adjacentElement.y, width: 0, height: adjacentElement.height }
}

/** The outer edge a place is within reach of, and nothing away from every edge. */
export function findEdgeSide(point: Position, box: Rect, reach: number): Side | null {
  const nearSides: readonly (readonly [Side, number])[] = [
    ['left', point.x - box.x],
    ['right', box.x + box.width - point.x],
    ['top', point.y - box.y],
    ['bottom', box.y + box.height - point.y],
  ]

  let side: Side | null = null
  let nearest = reach
  for (const [each, distance] of nearSides) {
    if (distance < nearest) {
      side = each
      nearest = distance
    }
  }
  return side
}

/** What an element takes up on screen, in the window's own coordinates. */
export const getRect = (element: Element): Rect => {
  const box = element.getBoundingClientRect()
  return { x: box.left, y: box.top, width: box.width, height: box.height }
}
