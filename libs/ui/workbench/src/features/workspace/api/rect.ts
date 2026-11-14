import type { Rect } from '../lib/rect'

/** What an element takes up on screen, in the window's own coordinates. */
export const getRect = (element: Element): Rect => {
  const box = element.getBoundingClientRect()
  return { x: box.left, y: box.top, width: box.width, height: box.height }
}
