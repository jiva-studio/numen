/**
 * What turns a book from one spread to the next by hand, and where a turn
 * lands. Which key turns which way is settled for every reader in
 * `shared/lib/turn.ts`.
 *
 * lifted again means is a decision, and a test asks it without a browser.
 */
import type { Span } from '@/shared/lib/span'
import type { PageNavigationAction } from '@/shared/lib/pageNavigation'

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

export interface HandTurnOptions {
  readonly wide: number
  readonly hasSelection: boolean
}

export function handTurn(from: number, to: number, options: HandTurnOptions): PageNavigationAction | undefined {
  if (options.hasSelection) return undefined
  return swipePage(to - from) ?? pressTurn(to, options.wide)
}

/** Which way a press across the reading area turns the page. */
export function pressTurn(x: number, width: number): PageNavigationAction | undefined {
  if (width <= 0) return undefined
  if (x <= width * EDGE) return 'back'
  if (x >= width * (1 - EDGE)) return 'next'
  return undefined
}

/** Where a turn lands: on a spread of this document, or past either end of it. */
export interface Destination {
  /** The spread the turn lands on, absent past the ends of the document. */
  readonly spread?: number
  /**
   * The offset asked for past an end of the document, which the holder of the
   * book answers by opening the document next to it.
   */
  readonly offset?: number
}

export interface TurnContext {
  readonly spread: number
  readonly count: number
  readonly span: Span
  readonly book: Span
}

/**
 * Where a turn lands. A turn inside the document lands on the spread it asks
 * for; past either end stands the document beside this one, which the holder
 * of the book opens at the offset the turn names.
 */
export function goToPage(way: PageNavigationAction, context: TurnContext): Destination {
  if (way === 'first') return { spread: 0 }
  if (way === 'last') return { spread: context.count - 1 }

  const to = context.spread + (way === 'next' ? 1 : -1)
  if (to >= 0 && to < context.count) return { spread: to }

  const offset = way === 'next' ? context.span.to : context.span.from - 1
  if (offset >= context.book.from && offset < context.book.to) return { offset }
  return {}
}
