/**
 * Which key turns a page, for everything read a page at a time.
 *
 * A book and a document are two ways of drawing the same act, and a reader who
 * has opened one of them does not know which. So the keys are settled once,
 * here, and each reader says what a turn does to what it draws.
 */

/** A turn of the page, and the two ends of what is read. */
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
