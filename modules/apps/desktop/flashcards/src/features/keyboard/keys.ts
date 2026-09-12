/**
 * What a keystroke asks for, on each screen it may be pressed at.
 *
 * Apart from the template because what a key means is a rule, and a rule inside
 * a component can only be exercised by pressing a key at a screen.
 */
import { isTyping } from '@numen/ui'
import { grades } from '@/entities/card'
import type { Grade } from '@/entities/card'

/** What a keystroke asks for in a session, and nothing when it asks nothing. */
export type SessionKeyIntent =
  | { does: 'show' }
  | { does: 'takeBack' }
  | { does: 'leave' }
  | { does: 'answer'; how: Grade }
  | { does: 'ask' }
  | { does: 'read' }
  | { does: 'scroll'; back: boolean }
  | { does: 'shut' }

/** What the window is showing when the key is pressed. */
export interface ScreenState {
  /** Whether the answer is already showing. */
  shown: boolean
  /** Whether the panel a card is asked about in is up. */
  asking?: boolean
  /** Whether the panel the deck's notes are read in is up. */
  reading?: boolean
}

/** The letter the panel a card is asked about in is brought in with. */
export const ASKS = 'a'

/** The letter the panel the deck's notes are read in is brought in with. */
export const READS = 'r'

/**
 * The keys a whole session is done with: space turns a card over, the four
 * numbers say how it went, `u` takes the last answer back and escape goes back
 * to the decks. A key held down, a key pressed with a modifier, a key typed
 * into a question and space or enter on a button the focus has moved to answer
 * nothing.
 */
export function getSessionKeyIntent(
  press: KeyboardEvent,
  showing: ScreenState,
): SessionKeyIntent | null {
  // A field takes the overlay key too: control and A is how a person selects
  // what they have written.
  if (isTyping(press)) return press.key === 'Escape' ? { does: 'shut' } : null
  // The panels are held with the overlay key, because the letters on their own
  // are what a card is answered by.
  if (hasOverlayKey(press)) {
    const letter = press.key.toLowerCase()
    if (letter === ASKS) return { does: 'ask' }
    if (letter === READS) return { does: 'read' }
    return null
  }
  if (hasModifierOrRepeat(press)) return null
  if (press.key === 'Escape') {
    return showing.asking || showing.reading ? { does: 'shut' } : { does: 'leave' }
  }
  if (press.key === 'u' || press.key === 'U') return { does: 'takeBack' }
  // The reading is read down, and space is the key the hand is already on. The
  // answer is still shown by the control standing under both panes.
  if (press.key === ' ' && showing.reading) return { does: 'scroll', back: press.shiftKey }
  if (press.key === ' ' && !showing.shown) return { does: 'show' }

  const which = Number(press.key)
  if (Number.isInteger(which) && which >= 1 && which <= grades.length) {
    const how = grades[which - 1]
    if (how) return { does: 'answer', how }
  }
  return null
}

/**
 * Whether the window swallows the key, rather than leaving it to the page. The
 * letter that brings the panel in is one of them: the field it opens takes the
 * keyboard, and the letter would be the first thing typed into it. So is space
 * over the reading, which the page would otherwise scroll instead.
 */
export const isSwallowed = (asked: SessionKeyIntent | null): boolean =>
  asked?.does === 'show' ||
  asked?.does === 'ask' ||
  asked?.does === 'read' ||
  asked?.does === 'scroll'

/** What a keystroke asks for while a person is choosing what to sit down to. */
export type PickerKeyIntent = { does: 'all' } | { does: 'deck'; at: number } | { does: 'back' }

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
export function getPickerKeyIntent(press: KeyboardEvent, decks: number): PickerKeyIntent | null {
  if (hasModifierOrRepeat(press)) return null
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
const hasModifierOrRepeat = (press: KeyboardEvent): boolean =>
  press.repeat || press.altKey || press.ctrlKey || press.metaKey

/** Whether the overlay key is held, which is control here and command on a Mac. */
const hasOverlayKey = (press: KeyboardEvent): boolean =>
  !press.repeat && !press.altKey && (press.ctrlKey || press.metaKey)
