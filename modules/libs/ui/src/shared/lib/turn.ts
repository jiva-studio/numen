/**
 * Which key turns a page, for everything read a page at a time.
 *
 * A book and a document are two ways of drawing the same act, and a reader who
 * has opened one of them does not know which. So the keys are settled once,
 * here, and each reader says what a turn does to what it draws.
 */

/** A turn of the page, and the two ends of what is read. */
export type PageTurn = 'back' | 'next' | 'first' | 'last'

const KEY_TURNS = new Map<string, PageTurn>([
  ['ArrowRight', 'next'],
  ['ArrowDown', 'next'],
  ['PageDown', 'next'],
  [' ', 'next'],
  ['ArrowLeft', 'back'],
  ['ArrowUp', 'back'],
  ['PageUp', 'back'],
  ['Home', 'first'],
  ['End', 'last'],
])

/** Which way a key turns the page, and nothing for a key that turns none. */
export const keyTurn = (key: string): PageTurn | undefined => KEY_TURNS.get(key)
