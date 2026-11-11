/**
 * A card's HTML, measured against what a card may be drawn with.
 *
 * A deck may come from another person and the window is a webview, so a tag,
 * an attribute or a scheme that is not named here does not survive.
 */

import { canCarry, canPointAt, filterStyles } from '../lib/cards/links'
import { DRAWN, STRUCK } from '../lib/cards/tags'

const stripComments = (root: ParentNode & Node): void => {
  const walk = (node: Node): void => {
    for (const child of [...node.childNodes]) {
      if (child.nodeType === 8) child.remove()
      else walk(child)
    }
  }
  walk(root)
}

const strip = (element: Element): void => {
  const tag = element.tagName.toLowerCase()
  for (const attribute of [...element.attributes]) {
    const name = attribute.name.toLowerCase()

    if (name === 'style') {
      const filtered = filterStyles(attribute.value)
      if (filtered === '') element.removeAttribute(attribute.name)
      else element.setAttribute('style', filtered)
      continue
    }

    if (!canCarry(tag, name) || !canPointAt(name, attribute.value)) {
      element.removeAttribute(attribute.name)
    }
  }
}

const pass = (html: string): string => {
  const readItem = new DOMParser().parseFromString(html, 'text/html')
  stripComments(readItem.body)

  for (const element of [...readItem.body.querySelectorAll('*')]) {
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

  return readItem.body.innerHTML
}

/**
 * Text a card may be drawn with. It is read twice, because what one reading
 * writes out is what the next parser is handed.
 */
export const safe = (html: string): string => pass(pass(html))
