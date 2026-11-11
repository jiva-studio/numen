/**
 * A card's HTML, measured against what a card may be drawn with.
 *
 * A deck may come from another person and the window is a webview, so a tag,
 * an attribute or a scheme that is not named here does not survive.
 */

import { ANY, OWN, SCHEMES, STYLED } from './tags'

/** An image standing in the text itself. */
const INLINE_IMAGE = /^data:image\/(png|jpeg|jpg|gif|webp|avif);base64,/i

const stripControls = (url: string): string =>
  Array.from(url)
    .filter((character) => character.charCodeAt(0) > 0x20)
    .join('')

/** The scheme a URL names, and nothing where it names none. */
export const scheme = (url: string): string | null =>
  /^[a-zA-Z][a-zA-Z0-9+.-]*:/.exec(stripControls(url))?.[0].toLowerCase() ?? null

const canLinkTo = (url: string): boolean => {
  const named = scheme(url)
  return named === null || SCHEMES.has(named)
}

const canShow = (url: string): boolean => {
  const cleaned = stripControls(url)
  if (scheme(cleaned) === 'data:') return INLINE_IMAGE.test(cleaned)
  // An address starting with two slashes names a host and inherits the window's
  // scheme, so it has no scheme of its own and still addresses a remote resource.
  return scheme(cleaned) === null && !/^[\\/]{2}/.test(cleaned)
}

export const filterStyles = (value: string): string =>
  value
    .split(';')
    .map((each) => each.trim())
    .filter((each) => {
      const pos = each.indexOf(':')
      if (pos === -1) return false
      const property = each.slice(0, pos).trim().toLowerCase()
      if (!STYLED.has(property)) return false
      const declaration = each.toLowerCase()
      return (
        !declaration.includes('url(') &&
        !declaration.includes('expression') &&
        !declaration.includes('\\')
      )
    })
    .join('; ')

export const canCarry = (tag: string, name: string): boolean =>
  // Stryker disable next-line ArrayDeclaration: the mutated value is ["Stryker was here"], a name with spaces that no parsed attribute has.
  ANY.has(name) || (OWN[tag] ?? []).includes(name)

export const canPointAt = (name: string, value: string): boolean => {
  if (name === 'href') return canLinkTo(value)
  if (name === 'src') return canShow(value)
  return true
}
