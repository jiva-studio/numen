/** Boxes and places, in whatever coordinates the caller measures in. */
import type { Position } from '@/shared/lib/geometry'

export type { Position }

export interface Rect {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** Where a place sits inside a box, with the box's corner as the origin. */
export const getPlaceInBox = (at: Position, box: Rect): Position => ({
  x: at.x - box.x,
  y: at.y - box.y,
})
