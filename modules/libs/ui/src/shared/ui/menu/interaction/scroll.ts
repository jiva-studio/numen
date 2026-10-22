/**
 * Whether a scroll is the person's.
 *
 * Opening a menu moves the page on its own: the trigger is scrolled into view
 * and focus moves into the list. Those scrolls arrive before the person has
 * touched anything, so a scroll counts as theirs once an input event has
 * arrived since the menu opened.
 */

/** The events that mean the person is driving the page. */
export const MENU_INPUT_EVENTS = ['wheel', 'touchstart', 'pointerdown', 'keydown'] as const

export interface ScrollGate {
  /** Forgets the input seen: a new opening starts with none. */
  readonly reset: () => void
  /** Records that the person gave input. */
  readonly noteInput: () => void
  /** Whether a scroll arriving now is the person's. */
  readonly isUserScroll: () => boolean
}

export const createScrollGate = (): ScrollGate => {
  let observed = false
  return {
    reset: () => {
      observed = false
    },
    noteInput: () => {
      observed = true
    },
    isUserScroll: () => observed,
  }
}
