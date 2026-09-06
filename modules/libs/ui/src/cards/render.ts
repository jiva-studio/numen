/**
 * What a card is written with, drawn.
 *
 * A face is HTML, so what a person wrote is drawn as the markup it is, and a
 * deck may come from another person, so that markup is measured against what a
 * card may be drawn with first.
 *
 * Nothing wraps a line written with no tag around it, so the card keeps the
 * breaks it was written with and each such line reads as the line it is. A
 * break standing between two tags is whitespace of the markup, and is taken out.
 */
import { safe } from './safe'

/** The tags whose own whitespace is the markup's, and no line of a card. */
const BLOCKS = new Set([
  'p', 'div', 'hr',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'blockquote', 'pre',
  'ul', 'ol', 'li', 'dl', 'dt', 'dd',
  'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'caption',
  'colgroup', 'col',
  'figure', 'figcaption',
])

/** Whether the whitespace beside this node stands between tags. The end of what holds it counts. */
const blocking = (node: Node | null): boolean =>
  node === null ||
  (node.nodeType === 1 && BLOCKS.has((node as Element).tagName.toLowerCase()))

const spacing = (node: Node): boolean =>
  node.nodeType === 3 &&
  (node.textContent ?? '').trim() === '' &&
  blocking(node.previousSibling) &&
  blocking(node.nextSibling)

/** The text with the whitespace that stands between tags taken out of it. */
const tightened = (html: string): string => {
  const read = new DOMParser().parseFromString(html, 'text/html')

  const walk = (node: Node): void => {
    for (const child of [...node.childNodes]) {
      // Preformatted text keeps every character it was written with.
      if (child.nodeType === 1 && (child as Element).tagName.toLowerCase() === 'pre') continue
      if (spacing(child)) child.parentNode?.removeChild(child)
      else walk(child)
    }
  }

  walk(read.body)
  return read.body.innerHTML
}

/** Text as the HTML a card draws, with nothing in it that a card may not. */
export const rendered = (html: string): string => tightened(safe(html))
