/** Boxes and points, in whatever coordinates the caller measures in. */

export interface Point {
  readonly x: number
  readonly y: number
}

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

export const holds = (box: Rect, point: Point): boolean =>
  point.x >= box.x &&
  point.x <= box.x + box.width &&
  point.y >= box.y &&
  point.y <= box.y + box.height
