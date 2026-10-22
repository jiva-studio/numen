/**
 * Typing a name to land the keyboard on it.
 *
 * Letters struck close together are one word, and one letter struck again and
 * again walks the items beginning with it. When a letter arrived is handed in,
 * so nothing here reads a clock.
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
  /** The item to stand on, and nothing where no item begins with the word. */
  readonly at: number | null
}

/** A key that stands for a letter a person meant to type. */
export const isLetter = (press: {
  key: string
  ctrlKey: boolean
  metaKey: boolean
  altKey: boolean
}): boolean =>
  press.key.length === 1 && press.key !== ' ' && !press.ctrlKey && !press.metaKey && !press.altKey

/**
 * Where a letter struck at `now` lands the keyboard, counting on from where it
 * stands and passing over the items that cannot be chosen.
 *
 * A letter that lengthens the word looks for the whole word from where the
 * keyboard already is, so a word being typed stands still while it grows. A
 * letter that begins a word, and the same letter struck again, look on from the
 * item after it.
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
    const at = (start + step) % items.length
    const item = items[at]
    if (!item || item.isDisabled) continue
    if (item.text.toLowerCase().startsWith(prefix)) return { typed: nextTypeahead, at }
  }
  return { typed: nextTypeahead, at: null }
}
