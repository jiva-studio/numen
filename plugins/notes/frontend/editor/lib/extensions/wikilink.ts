/**
 * The wikilinks of a line, as the editor follows them.
 *
 * `[[name]]` is the written form and `note://<identifier>` the auxiliary one. An alias after `|`
 * and a fragment after `#` are not part of the address. The core reads the same brackets from the
 * corpus in `libs/protocol/testdata`.
 */

/** A scheme is letters, digits, `+`, `-` and `.`, written before the `://`. */
const SCHEMED = /^([A-Za-z0-9+\-.]+):\/\/([\s\S]*)$/

const WIKILINK = /\[\[([^[\]]+)\]\]/g

/** One wikilink of a text: where its brackets begin and end, and its address as one string. */
export interface Wikilink {
  readonly at: number
  readonly to: number
  readonly address: string
}

/** The address between brackets as `scheme://value`; empty for no target. */
export const getAddress = (inside: string): string => {
  let target = inside
  const alias = target.indexOf('|')
  if (alias >= 0) target = target.slice(0, alias)
  const fragment = target.indexOf('#')
  if (fragment >= 0) target = target.slice(0, fragment)
  target = target.trim()
  const schemed = SCHEMED.exec(target)
  const [scheme, value] = schemed ? [schemed[1], schemed[2]] : ['name', target]
  return value ? `${scheme}://${value}` : ''
}

/** Every wikilink of a text, in the order they are written. */
export const getWikilinks = (text: string): readonly Wikilink[] => {
  const found: Wikilink[] = []
  for (const match of text.matchAll(WIKILINK)) {
    const address = getAddress(match[0].slice(2, -2))
    if (address) found.push({ at: match.index, to: match.index + match[0].length, address })
  }
  return found
}

/** The wikilink a position of a text falls inside, and null where it falls in none. */
export const findWikilink = (text: string, pos: number): Wikilink | null =>
  getWikilinks(text).find((one) => pos >= one.at && pos <= one.to) ?? null
