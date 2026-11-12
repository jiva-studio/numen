/**
 * What a move of a tab to another window says to the user, and where a tab opens in a new window.
 */
import type { TabMoveError } from '@/entities/tab'

export type Point = { readonly x: number; readonly y: number }

/** The box of a pane in viewport coordinates. */
export type PaneRect = Point & { readonly width: number; readonly height: number }

export const MOVE_ERRORS: Record<TabMoveError, string> = {
  'window-closed': 'the target window closed',
  timeout: 'the target window did not accept the tab in time',
  'not-owner': 'another window has this tab',
  cancelled: 'a newer move of the tab or a cancel ended this move',
}

export const ASKING_MESSAGE = 'The tab is waiting for an answer and was not moved'

export const STILL_OPEN_MESSAGE = 'The tab opened in the other window and did not close here'

/** How many times a window sends AcceptTabMove when the request throws. */
export const ACCEPT_ATTEMPTS = 3

/** How long a window waits before it sends a thrown AcceptTabMove again. */
export const ACCEPT_RETRY_MS = 1000

/** Returns the place of a new window at a screen point, with the size of a pane. */
export const getPlaceAt = (point: Point, rect: PaneRect | null): PaneRect => ({
  ...point,
  width: rect?.width ?? 0,
  height: rect?.height ?? 0,
})

/**
 * Returns the place of a new window over a pane: its screen position and size. A 0 position and
 * size leave the window where and as large as the system makes it.
 */
export const getPlaceOver = (rect: PaneRect | null, screen: Point): PaneRect =>
  rect
    ? { x: screen.x + rect.x, y: screen.y + rect.y, width: rect.width, height: rect.height }
    : { x: 0, y: 0, width: 0, height: 0 }
