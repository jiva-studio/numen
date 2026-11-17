/**
 * The wikilinks of a line, as the editor follows them.
 *
 * `[[name]]` is the written form and `note://<identifier>` the auxiliary one. An alias after `|`
 * and a fragment after `#` are not part of the address. The core reads the same brackets from the
 * corpus in `libs/shared/protocol/testdata`.
 */
import { getAddress } from './wikilinkAddress'

const WIKILINK = /\[\[([^[\]]+)\]\]/g

/**
 * One wikilink of a text: where its brackets begin and end, its address as one string, and the
 * stretch a reader sees when the brackets are concealed: the alias if there is one, else the target.
 */
export interface Wikilink {
  readonly at: number
  readonly to: number
  readonly address: string
  readonly shownFrom: number
  readonly shownTo: number
}

/** Every wikilink of a text, in the order they are written. */
export const getWikilinks = (text: string): readonly Wikilink[] => {
  const found: Wikilink[] = []
  for (const match of text.matchAll(WIKILINK)) {
    const inside = match[0].slice(2, -2)
    const address = getAddress(inside)
    if (!address) continue
    const alias = inside.indexOf('|')
    found.push({
      at: match.index,
      to: match.index + match[0].length,
      address,
      shownFrom: match.index + 2 + (alias >= 0 ? alias + 1 : 0),
      shownTo: match.index + match[0].length - 2,
    })
  }
  return found
}

/** The wikilink a position of a text falls inside, and null where it falls in none. */
export const findWikilink = (text: string, pos: number): Wikilink | null =>
  getWikilinks(text).find((one) => pos >= one.at && pos <= one.to) ?? null
