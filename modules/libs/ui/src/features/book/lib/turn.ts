/**
 * What turns a book from one spread to the next: the key it is read with, the
 * hand it is read by, and what each asks for at the ends of a document.
 *
 * Apart from the component the way `spread.ts` is: which key turns which way
 * and what a hand put down and lifted again means are decisions, and a test
 * asks them without a browser.
 */

/** A turn of the page, and the two ends of the document. */
export type PageTurn = 'back' | 'next' | 'first' | 'last'

/** Which way a key turns the page, and nothing for a key that turns none. */
export function keyTurn(key: string): PageTurn | undefined {
  switch (key) {
    case 'ArrowRight':
    case 'ArrowDown':
    case 'PageDown':
    case ' ':
      return 'next'
    case 'ArrowLeft':
    case 'ArrowUp':
    case 'PageUp':
      return 'back'
    case 'Home':
      return 'first'
    case 'End':
      return 'last'
    default:
      return undefined
  }
}

/** How far a hand travels sideways before it is a swipe, in CSS pixels. */
export const SWIPE = 40

/** Which way a swipe turns the page: the page follows the hand. */
export function swipeTurn(by: number): PageTurn | undefined {
  if (by <= -SWIPE) return 'next'
  if (by >= SWIPE) return 'back'
  return undefined
}

/** How much of either edge of the reading area is a press that turns, as a share of it. */
export const EDGE = 0.15

/**
 * What a hand put down and lifted again does: a swipe turns the way it went, a
 * press near either edge turns that way, and a hand that took words up did
 * neither. Taking words up and swiping are one gesture until the hand lifts,
 * and a run worth quoting is wider than a swipe.
 */
export function handTurn(
  from: number,
  to: number,
  wide: number,
  hasSelection: boolean,
): PageTurn | undefined {
  if (hasSelection) return undefined
  return swipeTurn(to - from) ?? pressTurn(to, wide)
}

/** Which way a press across the reading area turns the page. */
export function pressTurn(x: number, wide: number): PageTurn | undefined {
  if (wide <= 0) return undefined
  if (x <= wide * EDGE) return 'back'
  if (x >= wide * (1 - EDGE)) return 'next'
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

/**
 * Where a turn lands. A turn inside the document lands on the spread it asks
 * for; past either end stands the document beside this one, which the holder
 * of the book opens at the offset the turn names.
 */
export function turnTo(
  way: PageTurn,
  spread: number,
  count: number,
  span: { begins: number; ends: number },
  book: { begins: number; ends: number },
): Destination {
  if (way === 'first') return { spread: 0 }
  if (way === 'last') return { spread: count - 1 }

  const to = spread + (way === 'next' ? 1 : -1)
  if (to >= 0 && to < count) return { spread: to }

  const offset = way === 'next' ? span.ends : span.begins - 1
  if (offset >= book.begins && offset < book.ends) return { offset }
  return {}
}
