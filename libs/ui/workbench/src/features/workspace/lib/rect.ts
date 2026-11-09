/** Boxes and places, in whatever coordinates the caller measures in. */
import type { Position } from '@numen/ui'

export type { Position }

export interface Rect {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

export const getPlaceInBox = (point: Position, box: Rect): Position => ({
  x: point.x - box.x,
  y: point.y - box.y,
})
