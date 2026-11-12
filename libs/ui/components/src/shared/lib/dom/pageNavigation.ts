/**
 * Which key triggers page navigation, for everything read a page at a time.
 *
 * A book and a document are two ways of rendering the same reading experience,
 * so key actions are settled once here.
 */

/** An action of page navigation, including ends of document. */
export type PageNavigationAction = 'back' | 'next' | 'first' | 'last'

const KEY_ACTIONS = new Map<string, PageNavigationAction>([
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

/** Returns navigation action for a key, or undefined if key is unrelated. */
export const getPageNavigationAction = (key: string): PageNavigationAction | undefined =>
  KEY_ACTIONS.get(key)
