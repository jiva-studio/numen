/**
 * Manual page turns of a book from one spread to the next, and where a turn
 * lands. The key-to-direction mapping is shared by every reader in
 * `shared/lib/turn.ts`.
 *
 * What a turn means is a pure decision, which a test checks without a browser.
 */
import type { TextRange } from '@numen/ui'
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

export interface DragPageOptions {
  readonly wide: number
  readonly hasSelection: boolean
}

export function dragPage(
  from: number,
  to: number,
  options: DragPageOptions,
): PageNavigationAction | undefined {
  if (options.hasSelection) return undefined
  return swipePage(to - from) ?? pressPage(to, options.wide)
}

/** Which way a press across the reading area turns the page. */
export function pressPage(x: number, width: number): PageNavigationAction | undefined {
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
   * The offset requested past an end of the document, which the owner of the
   * book resolves by opening the adjacent document.
   */
  readonly offset?: number
}

export interface TurnContext {
  readonly spread: number
  readonly count: number
  readonly span: TextRange
  readonly book: TextRange
}

/**
 * Where a turn lands. A turn inside the document lands on the spread it
 * requests; a turn past either end targets the adjacent document, which the
 * owner of the book opens at the offset the turn names.
 */
export function goToPage(way: PageNavigationAction, context: TurnContext): Destination {
  if (way === 'first') return { spread: 0 }
  if (way === 'last') return { spread: context.count - 1 }

  const to = context.spread + (way === 'next' ? 1 : -1)
  if (to >= 0 && to < context.count) return { spread: to }

  const offset = way === 'next' ? context.span.end : context.span.start - 1
  if (offset >= context.book.start && offset < context.book.end) return { offset }
  return {}
}
