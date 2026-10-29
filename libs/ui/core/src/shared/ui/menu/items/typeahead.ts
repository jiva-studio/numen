/**
 * Typeahead: typing a name moves the keyboard focus to the matching item.
 *
 * Letters typed in quick succession form one word, and one letter typed
 * repeatedly cycles through the items beginning with it. The time of each
 * keystroke is a parameter, so nothing here reads a clock.
 */
import type { MenuItem } from './item'

/** How long a run of letters stays one word, in milliseconds. */
const TYPING = 1000

/** The word typed so far, and when its last letter arrived. */
export interface Typeahead {
  readonly word: string
  readonly struck: number
}

/** Nothing typed yet, which is where a menu opens. */
export const NOTHING_TYPED: Typeahead = { word: '', struck: 0 }

export interface Jump {
  readonly typed: Typeahead
  /** The item to focus, or null when no item begins with the word. */
  readonly at: number | null
}

/** A key press that types a single printable character, without a modifier. */
export const isLetter = (press: {
  key: string
  ctrlKey: boolean
  metaKey: boolean
  altKey: boolean
}): boolean =>
  press.key.length === 1 && press.key !== ' ' && !press.ctrlKey && !press.metaKey && !press.altKey

/**
 * The item a letter typed at `now` moves the keyboard to, counting on from the
 * current item and skipping the items that cannot be chosen.
 *
 * A letter that lengthens the word searches for the whole word from the current
 * item, so the focus stays in place while the word grows. A letter that begins
 * a word, and the same letter typed again, search from the next item.
 */
export interface JumpStrike {
  readonly letter: string
  readonly from: number
  readonly now: number
}

export function jumpTo(items: readonly MenuItem[], was: Typeahead, strike: JumpStrike): Jump {
  const { letter, from, now } = strike
  const word = now - was.struck > TYPING ? letter : was.word + letter
  const nextTypeahead: Typeahead = { word, struck: now }

  const first = word[0]!
  const drumming = [...word].every((each) => each === first)
  const prefix = (drumming ? first : word).toLowerCase()
  const start = from < 0 ? 0 : from + (drumming ? 1 : 0)

  for (const step of items.keys()) {
    const pos = (start + step) % items.length
    const item = items[pos]
    if (!item || item.isDisabled) continue
    if (item.text.toLowerCase().startsWith(prefix)) return { typed: nextTypeahead, at: pos }
  }
  return { typed: nextTypeahead, at: null }
}
