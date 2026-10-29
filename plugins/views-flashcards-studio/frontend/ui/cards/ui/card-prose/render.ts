/**
 * What a card is written with, rendered.
 *
 * A template is HTML, so the user's text is rendered as the markup it is, and a
 * deck may come from another author, so the markup is first checked against
 * what a card may render.
 *
 * Nothing wraps a line with no tag around it, so the card keeps the breaks it
 * was written with and each such line renders as its own line. A break between
 * two tags is markup whitespace and is removed.
 */
import { safe } from '../../lib/safe'

/** The tags whose own whitespace is the markup's, and no line of a card. */
const BLOCKS = new Set([
  'p',
  'div',
  'hr',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'blockquote',
  'pre',
  'ul',
  'ol',
  'li',
  'dl',
  'dt',
  'dd',
  'table',
  'thead',
  'tbody',
  'tfoot',
  'tr',
  'th',
  'td',
  'caption',
  'colgroup',
  'col',
  'figure',
  'figcaption',
])

const isBlock = (node: Node | null): boolean =>
  node === null || BLOCKS.has((node as Element).tagName.toLowerCase())

const spacing = (node: Node): boolean =>
  node instanceof Text &&
  node.data.trim() === '' &&
  isBlock(node.previousSibling) &&
  isBlock(node.nextSibling)

const tightenHtml = (html: string): string => {
  const readItem = new DOMParser().parseFromString(html, 'text/html')

  const walk = (node: Node): void => {
    for (const child of [...node.childNodes]) {
      // Preformatted text keeps every character it was written with.
      if (child.nodeType === 1 && (child as Element).tagName.toLowerCase() === 'pre') continue
      if (spacing(child)) child.remove()
      else walk(child)
    }
  }

  walk(readItem.body)
  return readItem.body.innerHTML
}

/** Text as the HTML a card draws, with nothing in it that a card may not. */
export const renderCardHtml = (html: string): string => tightenHtml(safe(html))
