/**
 * How a hand moves a row of pages: dragged, and turned with a wheel.
 *
 * Apart from the component the way `strip.ts` is. Which way a wheel moves a row
 * that has one axis and which way it moves one that has two is a decision, and
 * a test asks it without a browser.
 */

/** How far something is to be moved, in CSS pixels. */
export interface Offset {
  readonly x: number
  readonly y: number
}

/** What a wheel said, in the pixels it said it in. */
export interface Wheel {
  readonly x: number
  readonly y: number
}

export const DRAG_THRESHOLD = 3

/**
 * How far a wheel moves the row.
 *
 * above or below it and a wheel turned down means the next page. Drawn closer
 * the room has both, and then a wheel turned down means down — the way it does
 * everywhere else — and sideways is what a wheel says sideways.
 */
export function getWheelOffset(wheel: Wheel, hasBelow: boolean): Offset {
  if (hasBelow) return { x: wheel.x, y: wheel.y }
  return { x: wheel.x + wheel.y, y: 0 }
}

/**
 * A hand on the row: where it took hold, where the row stood, and whether it
 * has moved far enough to be dragging.
 *
 * A press that never moves is a press. Treating it as a drag from the first
 * pixel takes the click off whatever was under it.
 */
export class Hand {
  private grip: { readonly from: Offset; readonly status: Offset } | undefined
  private hasMoved = false

  /** The hand took hold, at a point, with the row standing here. */
  take(point: Offset, status: Offset) {
    this.grip = { from: point, status: status }
    this.hasMoved = false
  }

  /** Whether the hand is holding the row. */
  get holding(): boolean {
    return this.grip !== undefined
  }

  /** Whether it has moved far enough that this is a drag. */
  get dragging(): boolean {
    return this.hasMoved
  }

  /**
   * Where the row stands now that the hand is here, and nothing while the hand
   * is not holding it.
   *
   * pulled left brings the pages after this one into the room.
   */
  to(point: Offset): Offset | undefined {
    if (!this.grip) return undefined
    const { from, status } = this.grip
    const delta = { x: point.x - from.x, y: point.y - from.y }
    if (!this.hasMoved && Math.hypot(delta.x, delta.y) < DRAG_THRESHOLD) return undefined
    this.hasMoved = true
    return { x: status.x - delta.x, y: status.y - delta.y }
  }

  /** The hand let go. */
  release() {
    this.grip = undefined
  }
}
