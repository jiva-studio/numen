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

/** How far the hand travels before it is dragging and not pressing. */
export const DRAG_THRESHOLD = 3

/**
 * How far a wheel moves the row.
 *
 * A row at rest has one axis: a whole page stands in the room, so nothing is
 * above or below it and a wheel turned down means the next page. Drawn closer
 * the room has both, and then a wheel turned down means down — the way it does
 * everywhere else — and sideways is what a wheel says sideways.
 */
export function wheeled(wheel: Wheel, hasBelow: boolean): Offset {
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
  private from: Offset | undefined
  private stood: Offset = { x: 0, y: 0 }
  private moved = false

  /** The hand took hold, at a point, with the row standing here. */
  take(at: Offset, stood: Offset) {
    this.from = at
    this.stood = stood
    this.moved = false
  }

  /** Whether the hand is holding the row. */
  get holding(): boolean {
    return this.from !== undefined
  }

  /** Whether it has moved far enough that this is a drag. */
  get dragging(): boolean {
    return this.moved
  }

  /**
   * Where the row stands now that the hand is here, and nothing while the hand
   * is not holding it.
   *
   * The row follows the hand, so it moves against the way the hand went: a hand
   * pulled left brings the pages after this one into the room.
   */
  to(at: Offset): Offset | undefined {
    if (!this.from) return undefined
    const by = { x: at.x - this.from.x, y: at.y - this.from.y }
    if (!this.moved && Math.hypot(by.x, by.y) < DRAG_THRESHOLD) return undefined
    this.moved = true
    return { x: this.stood.x - by.x, y: this.stood.y - by.y }
  }

  /** The hand let go. */
  release() {
    this.from = undefined
  }
}
