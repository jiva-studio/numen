/**
 * The action a keystroke requests on each screen where it may be pressed.
 *
 * Kept outside the template because the meaning of a key is a rule, and a rule
 * inside a component can only be tested by pressing a key on a rendered screen.
 */
import { grades } from '../card/grades'
import type { Grade } from '../card/grades'

/** The fields of a key press that these rules read; a keyboard event has them all. */
export interface KeyPress {
  readonly key: string
  readonly repeat: boolean
  readonly altKey: boolean
  readonly ctrlKey: boolean
  readonly metaKey: boolean
  readonly shiftKey: boolean
  readonly target: object | null
}

/** Whether the key was pressed into something being written in. */
const isTyping = (press: KeyPress): boolean => {
  const { target } = press
  if (!target) return false
  return (
    ('tagName' in target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) ||
    ('isContentEditable' in target && target.isContentEditable === true)
  )
}

/** The action a keystroke requests in a session, or null when it requests none. */
export type SessionKeyIntent =
  | { does: 'show' }
  | { does: 'takeBack' }
  | { does: 'leave' }
  | { does: 'answer'; how: Grade }
  | { does: 'ask' }
  | { does: 'read' }
  | { does: 'scroll'; back: boolean }
  | { does: 'shut' }

export interface ScreenState {
  isShown: boolean
  isAsking?: boolean
  isNotesOpen?: boolean
}

/** The letter that opens the panel for asking about the card. */
export const ASK_KEY = 'a'

/** The letter the panel the deck's notes are read in is brought in with. */
export const READ_KEY = 'r'

/**
 * The keys a whole session is done with: space turns a card over, the four
 * numbers say how it went, `u` takes the last answer back and escape goes back
 * to the decks. A key held down, a key pressed with a modifier, a key typed
 * into a question and space or enter on a button the focus has moved to answer
 * nothing.
 */
export function getSessionKeyIntent(
  press: KeyPress,
  screen: ScreenState,
): SessionKeyIntent | null {
  // A field also receives the overlay key: control and A selects the text it
  // contains.
  if (isTyping(press)) return press.key === 'Escape' ? { does: 'shut' } : null
  // The panels are held with the overlay key, because the letters on their own
  // are what a card is answered by.
  if (hasOverlayKey(press)) return getPanelKeyIntent(press)
  if (hasModifierOrRepeat(press)) return null
  return getCardKeyIntent(press, screen)
}

const getPanelKeyIntent = (press: KeyPress): SessionKeyIntent | null => {
  const letter = press.key.toLowerCase()
  if (letter === ASK_KEY) return { does: 'ask' }
  if (letter === READ_KEY) return { does: 'read' }
  return null
}

const getCardKeyIntent = (press: KeyPress, screen: ScreenState): SessionKeyIntent | null => {
  if (press.key === 'Escape') {
    return screen.isAsking || screen.isNotesOpen ? { does: 'shut' } : { does: 'leave' }
  }
  if (press.key === 'u' || press.key === 'U') return { does: 'takeBack' }
  if (press.key === ' ') return getSpaceKeyIntent(press, screen)
  return getGradeKeyIntent(press)
}

const getSpaceKeyIntent = (press: KeyPress, screen: ScreenState): SessionKeyIntent | null => {
  if (screen.isNotesOpen) return { does: 'scroll', back: press.shiftKey }
  return screen.isShown ? null : { does: 'show' }
}

const getGradeKeyIntent = (press: KeyPress): SessionKeyIntent | null => {
  const how = grades[Number(press.key) - 1]
  return how ? { does: 'answer', how } : null
}

/**
 * Whether the window consumes the key and does not pass it to the page. The
 * letter that opens a panel is consumed: the field it opens takes focus, and
 * the letter would be the first character typed into it. Space over
 * the notes is consumed because the page would scroll.
 */
export const shouldConsumeKey = (intent: SessionKeyIntent | null): boolean =>
  intent?.does === 'show' ||
  intent?.does === 'ask' ||
  intent?.does === 'read' ||
  intent?.does === 'scroll'

/** The action a keystroke requests while a deck is being chosen for a session. */
export type PickerKeyIntent = { does: 'all' } | { does: 'deck'; at: number } | { does: 'back' }

/** The letters the decks are picked by, in the order they are listed. */
const LETTERS = 'abcdefghijklmnopqrstuvwxyz'

/**
 * The letter that picks the deck at an index of the list, or an empty string
 * past the end of the alphabet: only the first twenty-six decks have a letter.
 */
export const getLetter = (index: number): string => LETTERS[index] ?? ''

/**
 * Keys of the deck picker: enter or space picks the whole vault, a letter picks the deck at its
 * index, and escape returns to the vault list. The whole vault is the daily action, so it has
 * the most accessible keys.
 */
export function getPickerKeyIntent(press: KeyPress, decks: number): PickerKeyIntent | null {
  if (hasModifierOrRepeat(press)) return null
  if (press.key === 'Escape') return { does: 'back' }
  if (press.key === 'Enter' || press.key === ' ') return { does: 'all' }

  if (press.key.length !== 1) return null
  const choiceIndex = LETTERS.indexOf(press.key.toLowerCase())
  if (choiceIndex >= 0 && choiceIndex < decks) return { does: 'deck', at: choiceIndex }
  return null
}

const hasModifierOrRepeat = (press: KeyPress): boolean =>
  press.repeat || press.altKey || press.ctrlKey || press.metaKey

const hasOverlayKey = (press: KeyPress): boolean =>
  !press.repeat && !press.altKey && (press.ctrlKey || press.metaKey)
