import type { PageNavigationAction } from '@numen/ui'
import { EDGE, SWIPE } from './pageGestureSizes'

/** Determine page turn direction from swipe delta. */
export function swipePage(delta: number): PageNavigationAction | undefined {
  if (delta <= -SWIPE) return 'next'
  if (delta >= SWIPE) return 'back'
  return undefined
}

/** Which way a press across the reading area turns the page. */
export function pressPage(x: number, width: number): PageNavigationAction | undefined {
  if (width <= 0) return undefined
  if (x <= width * EDGE) return 'back'
  if (x >= width * (1 - EDGE)) return 'next'
  return undefined
}
