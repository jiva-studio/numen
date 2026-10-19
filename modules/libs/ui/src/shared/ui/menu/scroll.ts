/**
 * Whether a scroll is the person's.
 *
 * Opening a menu moves the page on its own: the trigger is scrolled into view
 * and focus moves into the list. Those scrolls arrive before the person has
 * touched anything, so a scroll counts as theirs once an input event has
 * followed the gate being armed.
 */

/** The events that mean the person is driving the page. */
export const MENU_INPUT_EVENTS = ['wheel', 'touchstart', 'pointerdown', 'keydown'] as const

export interface ScrollGate {
  /** Starts a new opening: no input has been seen yet. */
  readonly arm: () => void
  /** Records that the person gave input. */
  readonly noteInput: () => void
  /** Whether a scroll arriving now is the person's. */
  readonly isUserScroll: () => boolean
}

export function createScrollGate(): ScrollGate {
  let seen = false
  return {
    arm: () => {
      seen = false
    },
    noteInput: () => {
      seen = true
    },
    isUserScroll: () => seen,
  }
}
