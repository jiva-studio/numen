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
import type { Directive } from 'vue'
import { safe } from './safe'

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

const isMarkupSpacing = (node: Node): boolean =>
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
      if (isMarkupSpacing(child)) child.remove()
      else walk(child)
    }
  }

  walk(readItem.body)
  return readItem.body.innerHTML
}

/** Text as the HTML a card draws, with nothing in it that a card may not. */
export const renderCardHtml = (html: string): string => tightenHtml(safe(html))

/** Writes markup that was checked before it reached a card component. */
export const vMarkup: Directive<HTMLElement, string> = {
  created: (element, binding) => {
    element.innerHTML = binding.value
  },
  beforeUpdate: (element, binding) => {
    if (binding.value !== binding.oldValue) element.innerHTML = binding.value
  },
}

/** Finds the href of the enclosing link for a click event, if any. */
export const findLinkHref = (press: MouseEvent): string | null => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  return link?.getAttribute('href') ?? null
}
