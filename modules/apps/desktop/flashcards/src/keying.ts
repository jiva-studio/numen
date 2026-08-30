/**
 * What a keystroke asks for, on each screen it may be pressed at.
 *
 * Apart from the template because what a key means is a rule, and a rule inside
 * a component can only be exercised by pressing a key at a screen.
 */
import { said } from './core'
import type { Said } from './core'

/** What a keystroke asks for in a sitting, and nothing when it asks nothing. */
export type Asks =
  | { does: 'show' }
  | { does: 'takeBack' }
  | { does: 'leave' }
  | { does: 'answer'; how: Said }
  | { does: 'ask' }
  | { does: 'shut' }

/** What the window is showing when the key is pressed. */
export interface Showing {
  /** Whether the answer is already showing. */
  shown: boolean
  /** Whether the panel a card is asked about in is up. */
  asking?: boolean
}

/** The letter the panel a card is asked about in is brought in with. */
export const ASKS = 'a'

/**
 * Whether the key was pressed into something being written in. A letter is
 * text there, and a sitting reads none of its own keys out of a field.
 */
export const typing = (press: KeyboardEvent): boolean => {
  const at = press.target as HTMLElement | null
  if (!at) return false
  return at.tagName === 'INPUT' || at.tagName === 'TEXTAREA' || at.isContentEditable === true
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
 *
 * While a question is being written none of these are pressed: the letters are
 * the question. Escape there sends the panel away and leaves the sitting where
 * it is.
 */
export function asks(press: KeyboardEvent, showing: Showing): Asks | null {
  if (spoken(press)) return null
  if (typing(press)) return press.key === 'Escape' ? { does: 'shut' } : null
  if (press.key === 'Escape') return showing.asking ? { does: 'shut' } : { does: 'leave' }
  if (press.key === 'u' || press.key === 'U') return { does: 'takeBack' }
  if (press.key === ' ' && !showing.shown) return { does: 'show' }
  // A card is asked about once its answer is showing: one that can be asked
  // about before it is turned is a way not to recall it.
  if ((press.key === ASKS || press.key === ASKS.toUpperCase()) && showing.shown) {
    return { does: 'ask' }
  }

  const which = Number(press.key)
  if (Number.isInteger(which) && which >= 1 && which <= said.length) {
    const how = said[which - 1]
    if (how) return { does: 'answer', how }
  }
  return null
}

/** Whether the window swallows the key, rather than leaving it to the page. */
export const swallows = (asked: Asks | null): boolean => asked?.does === 'show'

/** What a keystroke asks for while a person is choosing what to sit down to. */
export type Picks =
  | { does: 'all' }
  | { does: 'deck'; at: number }
  | { does: 'back' }

/** The letters the decks are picked by, in the order they are listed. */
export const LETTERS = 'abcdefghijklmnopqrstuvwxyz'

/**
 * The letter one deck of a list is picked by, and nothing past the alphabet: a
 * vault of thirty decks is a vault where the last four are picked with the
 * hand.
 */
export const letterOf = (at: number): string => LETTERS[at] ?? ''

/**
 * The keys the decks are chosen with: enter sits down to the whole vault, a
 * letter to the deck standing at it, and escape goes back to the vaults.
 *
 * The whole vault is the daily act, so it is the key under the hand. A deck is
 * a letter because a person reads down the list and presses what they see.
 */
export function picks(press: KeyboardEvent, decks: number): Picks | null {
  if (spoken(press)) return null
  if (press.key === 'Escape') return { does: 'back' }
  if (press.key === 'Enter' || press.key === ' ') return { does: 'all' }

  if (press.key.length !== 1) return null
  const at = LETTERS.indexOf(press.key.toLowerCase())
  if (at >= 0 && at < decks) return { does: 'deck', at }
  return null
}

/**
 * A key held down repeats, and a key pressed with a modifier is the machine's
 * own shortcut. Neither is a person asking for anything here.
 */
const spoken = (press: KeyboardEvent): boolean =>
  press.repeat || press.altKey || press.ctrlKey || press.metaKey
