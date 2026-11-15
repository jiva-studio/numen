import type { PageNavigationAction } from '@numen/ui'

/** Minimum horizontal distance in CSS pixels for a swipe gesture. */
export const SWIPE = 40

/** Determine page turn direction from swipe delta. */
export function swipePage(delta: number): PageNavigationAction | undefined {
  if (delta <= -SWIPE) return 'next'
  if (delta >= SWIPE) return 'back'
  return undefined
}

/** Edge margin ratio of the reading area that triggers a page turn tap. */
export const EDGE = 0.15

/** Which way a press across the reading area turns the page. */
export function pressPage(x: number, width: number): PageNavigationAction | undefined {
  if (width <= 0) return undefined
  if (x <= width * EDGE) return 'back'
  if (x >= width * (1 - EDGE)) return 'next'
  return undefined
}
