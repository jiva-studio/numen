/**
 * Whether a scroll was made by the user.
 *
 * Opening a menu scrolls the page programmatically: the trigger is scrolled into
 * view and focus moves into the list. Those scrolls occur before any user input,
 * so a scroll counts as the user's once an input event has occurred since the
 * menu opened.
 */

/** The events that indicate the user is driving the page. */
export const MENU_INPUT_EVENTS = ['wheel', 'touchstart', 'pointerdown', 'keydown'] as const

export interface ScrollGate {
  /** Forgets the input seen: a new opening starts with none. */
  readonly reset: () => void
  /** Records that the user gave input. */
  readonly noteInput: () => void
  /** Whether a scroll occurring now is the user's. */
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
