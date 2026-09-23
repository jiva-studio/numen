/**
 * A card's HTML, measured against what a card may be drawn with.
 *
 * A deck may come from another person and the window is a webview, so a tag,
 * an attribute or a scheme that is not named here does not survive.
 */

import { ANY, DRAWN, OWN, SCHEMES, STRUCK, STYLED } from './tags'

/** An image standing in the text itself. */
const INLINE_IMAGE = /^data:image\/(png|jpeg|jpg|gif|webp|avif);base64,/i

/** A URL with the spaces and control characters a scheme may be hidden behind dropped. */
const bare = (url: string): string => url.replace(/[\u0000-\u0020]/g, '')

/** The scheme a URL names, and nothing where it names none. */
export const scheme = (url: string): string | null =>
  /^[a-zA-Z][a-zA-Z0-9+.-]*:/.exec(bare(url))?.[0]?.toLowerCase() ?? null

const canLinkTo = (url: string): boolean => {
  const said = scheme(url)
  return said === null || SCHEMES.has(said)
}

/**
 * A picture a card may draw: its own bytes, or a file this window serves.
 *
 * An address off the machine is a request the moment the card is drawn, which
 * tells whoever wrote the deck that it was read, and from where.
 */
const canShow = (url: string): boolean => {
  const said = bare(url)
  if (scheme(said) === 'data:') return INLINE_IMAGE.test(said)
  // An address opening with two slashes names a host and keeps the window's
  // own scheme, so it names no scheme and reaches off the machine all the same.
  return scheme(said) === null && !/^[\\/]{2}/.test(said)
}

/** A style with every declaration that is not drawn with dropped. */
const filterStyles = (value: string): string =>
  value
    .split(';')
    .map((each) => each.trim())
    .filter((each) => {
      const at = each.indexOf(':')
      if (at === -1) return false
      const property = each.slice(0, at).trim().toLowerCase()
      const said = each.slice(at + 1).toLowerCase()
      if (!STYLED.has(property)) return false
      return !said.includes('url(') && !said.includes('expression') && !said.includes('\\')
    })
    .join('; ')

/** Comments go: what a parser makes of one is not what the next parser makes. */
const stripComments = (root: ParentNode & Node): void => {
  const walk = (node: Node): void => {
    for (const child of [...node.childNodes]) {
      if (child.nodeType === 8) child.parentNode?.removeChild(child)
      else walk(child)
    }
  }
  walk(root)
}

/** Whether a tag may carry an attribute of that name. */
const canCarry = (tag: string, name: string): boolean =>
  ANY.has(name) || (OWN[tag] ?? []).includes(name)

/** Whether an address an attribute names is one a card may reach. */
const canPointAt = (name: string, value: string): boolean => {
  if (name === 'href') return canLinkTo(value)
  if (name === 'src') return canShow(value)
  return true
}

const strip = (element: Element): void => {
  const tag = element.tagName.toLowerCase()
  for (const attribute of [...element.attributes]) {
    const name = attribute.name.toLowerCase()

    if (name === 'style') {
      const said = filterStyles(attribute.value)
      if (said === '') element.removeAttribute(attribute.name)
      else element.setAttribute('style', said)
      continue
    }

    if (!canCarry(tag, name) || !canPointAt(name, attribute.value)) {
      element.removeAttribute(attribute.name)
    }
  }
}

/** One reading of the text, with everything not drawn with taken out of it. */
const pass = (html: string): string => {
  const read = new DOMParser().parseFromString(html, 'text/html')
  stripComments(read.body)

  for (const element of [...read.body.querySelectorAll('*')]) {
    const tag = element.tagName.toLowerCase()
    if (STRUCK.has(tag)) {
      element.remove()
      continue
    }
    if (!DRAWN.has(tag)) {
      element.replaceWith(...element.childNodes)
      continue
    }
    strip(element)
  }

  return read.body.innerHTML
}

/**
 * Text a card may be drawn with. It is read twice, because what one reading
 * writes out is what the next parser is handed.
 */
export const safe = (html: string): string => pass(pass(html))
