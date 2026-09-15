/**
 * A link to a note, as it is written wherever it is written.
 *
 * `[[name]]` is the written form and `note://<identifier>` the auxiliary one,
 * written inside the brackets where a name picks nobody. An address is a
 * scheme and a value, so reading one is a split at `://`; a title carrying a
 * colon of its own begins no scheme.
 */

/** What a link points at. */
export interface Address {
  readonly scheme: string
  readonly value: string
}

/** The scheme a bare name carries. It is what the brackets become, never what they hold. */
export const NAME = 'name'

/** The scheme an identifier carries. It names one note in the world. */
export const NOTE = 'note'

/** A scheme is letters, digits, `+`, `-` and `.`, written before the `://`. */
const SCHEME = /^[A-Za-z0-9+\-.]+$/

/**
 * Reads what stands between the brackets.
 *
 * An alias after `|` is how the link is read in the sentence and a fragment
 * after `#` names a place inside the note. Neither is part of the address.
 */
export const addressOf = (raw: string): Address => {
  let target = raw.trim()
  if (target.startsWith('[[')) target = target.slice(2)
  if (target.endsWith(']]')) target = target.slice(0, -2)

  const alias = target.indexOf('|')
  if (alias >= 0) target = target.slice(0, alias)
  const fragment = target.indexOf('#')
  if (fragment >= 0) target = target.slice(0, fragment)
  target = target.trim()

  const split = target.indexOf('://')
  if (split > 0) {
    const scheme = target.slice(0, split)
    if (SCHEME.test(scheme)) return { scheme, value: target.slice(split + 3) }
  }
  return { scheme: NAME, value: target }
}

/** An address as one string, which is how it is handed on and read back. */
export const writeAddress = (address: Address): string => `${address.scheme}://${address.value}`

/** Whether an address, written as one string, points at a note. */
export const isNoteAddress = (address: string): boolean =>
  address.startsWith(`${NAME}://`) || address.startsWith(`${NOTE}://`)

/** One wikilink as it stands in a text. */
export interface Wikilink {
  /** Where the brackets begin, and where they end. */
  readonly at: number
  readonly to: number
  /** What it points at, as one string. */
  readonly address: string
  /** How it is read in the sentence: the alias where one was written. */
  readonly text: string
}

const WIKILINK = /\[\[([^[\]]+)\]\]/g

/** Every wikilink in a text, in the order they are written. */
export const wikilinksIn = (text: string): readonly Wikilink[] => {
  const found: Wikilink[] = []
  for (const match of text.matchAll(WIKILINK)) {
    const inside = match[1] ?? ''
    const address = addressOf(inside)
    if (address.value === '') continue
    const alias = inside.indexOf('|')
    found.push({
      at: match.index,
      to: match.index + match[0].length,
      address: writeAddress(address),
      text: (alias >= 0 ? inside.slice(alias + 1) : inside).trim(),
    })
  }
  return found
}

/** The wikilink a position in a text falls inside, and nothing where it falls in none. */
export const wikilinkAt = (text: string, at: number): Wikilink | null =>
  wikilinksIn(text).find((one) => at >= one.at && at <= one.to) ?? null
