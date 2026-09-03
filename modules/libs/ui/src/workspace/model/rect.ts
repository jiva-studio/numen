/** Boxes and points, in whatever coordinates the caller measures in. */
import type { Point } from '../../lib/geometry'

export type { Point }

export interface Rect {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** Where a point sits inside a box, with the box's corner as the origin. */
export const within = (point: Point, box: Rect): Point => ({
  x: point.x - box.x,
  y: point.y - box.y,
})
