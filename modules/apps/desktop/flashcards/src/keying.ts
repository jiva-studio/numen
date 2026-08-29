/**
 * What a keystroke asks for while a person is answering cards.
 *
 * Apart from the template because what a key means is a rule, and a rule inside
 * a component can only be exercised by pressing a key at a screen.
 */
import { said } from './core'
import type { Said } from './core'

/** What a keystroke asks for, and nothing when it asks for nothing. */
export type Asks = { does: 'show' } | { does: 'takeBack' } | { does: 'leave' } | { does: 'answer'; how: Said }

/** What the window is showing when the key is pressed. */
export interface Showing {
  /** Whether the answer is already showing. */
  shown: boolean
}

/**
 * The keys a whole sitting is done with: the space bar turns a card over, the
 * four numbers say how it went, `u` takes the last answer back and escape goes
 * back to the decks.
 *
 * A key held down repeats, and a card is answered once. A key pressed with a
 * modifier is the machine's own shortcut and is not an answer. Once the card is
 * over, space and enter are left to whatever the person has moved focus to, so
 * a button reached with the keyboard is pressed with the keyboard.
 */
export function asks(press: KeyboardEvent, showing: Showing): Asks | null {
  if (press.repeat || press.altKey || press.ctrlKey || press.metaKey) return null
  if (press.key === 'Escape') return { does: 'leave' }
  if (press.key === 'u' || press.key === 'U') return { does: 'takeBack' }
  if (press.key === ' ' && !showing.shown) return { does: 'show' }

  const which = Number(press.key)
  if (Number.isInteger(which) && which >= 1 && which <= said.length) {
    const how = said[which - 1]
    if (how) return { does: 'answer', how }
  }
  return null
}

/** Whether the window swallows the key, rather than leaving it to the page. */
export const swallows = (asked: Asks | null): boolean => asked?.does === 'show'
