import type { Rect } from '../lib/rect'

/** What an element takes up on screen, in the window's own coordinates. */
const RECT_METHOD = 'getBoundingClientRect'

export const getRect = (element: Element): Rect => {
  const box = element[RECT_METHOD]()
  return { x: box.left, y: box.top, width: box.width, height: box.height }
}
