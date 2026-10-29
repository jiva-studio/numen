/**
 * Pointer drag and wheel handling for a row of pages.
 *
 * Kept separate from the component, like `strip.ts`, so the mapping of wheel
 * input to row offset for one-axis and two-axis layouts is testable without a
 * browser.
 */

/** How far something is to be moved, in CSS pixels. */
export interface Offset {
  readonly x: number
  readonly y: number
}

/** Wheel deltas in pixels. */
export interface Wheel {
  readonly x: number
  readonly y: number
}

export const DRAG_THRESHOLD = 3

/**
 * How far a wheel moves the row.
 *
 * When nothing lies above or below the row, vertical wheel movement is added to
 * horizontal movement, so scrolling down advances to the next page. When the
 * viewport has vertical room (`hasBelow`), vertical movement stays vertical.
 */
export function getWheelOffset(wheel: Wheel, hasBelow: boolean): Offset {
  if (hasBelow) return { x: wheel.x, y: wheel.y }
  return { x: wheel.x + wheel.y, y: 0 }
}

/**
 * A pointer press on the row: where it started, the row offset at that point,
 * and whether it has moved far enough to count as a drag.
 *
 * A press that never moves is a press. Treating it as a drag from the first
 * pixel takes the click off whatever was under it.
 */
export class Hand {
  private grip: { readonly from: Offset; readonly status: Offset } | undefined
  private hasMoved = false

  /** The pointer pressed at a point while the row was at this offset. */
  take(point: Offset, status: Offset) {
    this.grip = { from: point, status: status }
    this.hasMoved = false
  }

  /** Whether the pointer is holding the row. */
  get holding(): boolean {
    return this.grip !== undefined
  }

  /** Whether it has moved far enough that this is a drag. */
  get dragging(): boolean {
    return this.hasMoved
  }

  /**
   * The row offset for the pointer's current position, or undefined while the
   * pointer is not holding the row.
   *
   * Dragging left brings the pages after this one into view.
   */
  to(point: Offset): Offset | undefined {
    if (!this.grip) return undefined
    const { from, status } = this.grip
    const delta = { x: point.x - from.x, y: point.y - from.y }
    if (!this.hasMoved && Math.hypot(delta.x, delta.y) < DRAG_THRESHOLD) return undefined
    this.hasMoved = true
    return { x: status.x - delta.x, y: status.y - delta.y }
  }

  /** The pointer was released. */
  release() {
    this.grip = undefined
  }
}
