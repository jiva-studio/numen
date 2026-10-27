/**
 * Markdown as Vue's own nodes, read by markdown-it. Vue patches the tree it
 * drew before, so prose arriving a piece at a time keeps what is on the screen,
 * and a word that has just arrived can be shown arriving.
 *
 * HTML in the text is text.
 */
import MarkdownIt, { type Token } from 'markdown-it'
import { h, type VNode } from 'vue'
import { linkPieces, type LinkPieces, type LinkPiecesEnv } from '@/shared/lib/text/marks'

const marks = new MarkdownIt({ html: false, linkify: true }).use(linkPieces)

/** The class every word is drawn in. */
export const WORD = 'prose__word'

/** The attribute a link that reaches nothing is drawn with. */
const REACHES = 'data-reaches'

/** The addresses a text points at that reach nothing. */
export type BrokenAddresses = ReadonlySet<string>

const NONE: BrokenAddresses = new Set()

const NO_PIECES: LinkPieces = new Map()

export const render = (
  text: string,
  deadAddresses: BrokenAddresses = NONE,
  pieces: LinkPieces = NO_PIECES,
): VNode[] => {
  const env: LinkPiecesEnv = { pieces }
  return nodes(marks.parse(text, env), deadAddresses)
}

/**
 * Tokens arrive flat, with the nesting written on them. A frame is one element
 * being filled; the tokens that close it hand it to the frame beneath.
 */
interface Frame {
  tag: string
  attrs: Record<string, string>
  children: (VNode | string)[]
}

const nodes = (tokens: readonly Token[], deadAddresses: BrokenAddresses): VNode[] => {
  const root: VNode[] = []
  // Stryker disable next-line ArrayDeclaration: a first entry of the stack is never read, as every closing token has its opener.
  const stack: Frame[] = []
  const top = (): (VNode | string)[] => stack.at(-1)?.children ?? root
  let placedCount = 0

  for (const token of tokens) {
    // A hidden token is one the marks imply and the reader never sees: the
    // paragraph markdown puts inside a tight list item.
    if (token.hidden) continue

    if (token.nesting === 1) {
      stack.push({ tag: token.tag, attrs: attrs(token, deadAddresses), children: [] })
      continue
    }
    if (token.nesting === -1) {
      const isDone = stack.pop()!
      top().push(h(isDone.tag, isDone.attrs, isDone.children))
      continue
    }

    top().push(...block(token, deadAddresses, () => placedCount++))
  }
  return root
}

/** What a block token stands for, once it has opened and closed nothing. */
const block = (
  token: Token,
  deadAddresses: BrokenAddresses,
  next: () => number,
): (VNode | string)[] => {
  switch (token.type) {
    case 'inline':
      return inline(token.children ?? [], deadAddresses, next)
    case 'hr':
      return [h('hr')]
    default:
      // A fence or a code block.
      return [h('pre', attrs(token), [h('code', token.content)])]
  }
}

/**
 * The inside of a paragraph: words, and the marks that dress them.
 *
 * Every word is a node of its own, keyed by where it falls in the answer.
 */
const inline = (
  tokens: readonly Token[],
  deadAddresses: BrokenAddresses,
  next: () => number,
): (VNode | string)[] => {
  const root: (VNode | string)[] = []
  // Stryker disable next-line ArrayDeclaration: a first entry of the stack is never read, as every closing token has its opener.
  const stack: Frame[] = []
  const top = (): (VNode | string)[] => stack.at(-1)?.children ?? root

  for (const token of tokens) {
    if (token.nesting === 1) {
      stack.push({ tag: token.tag, attrs: attrs(token, deadAddresses), children: [] })
      continue
    }
    if (token.nesting === -1) {
      const isDone = stack.pop()!
      top().push(h(isDone.tag, isDone.attrs, isDone.children))
      continue
    }

    top().push(...mark(token, next))
  }
  return root
}

/** What an inline token stands for, once it has opened and closed nothing. */
const mark = (token: Token, next: () => number): (VNode | string)[] => {
  switch (token.type) {
    case 'text':
      return words(token.content, next)
    case 'code_inline':
      return [h('code', { key: next() }, token.content)]
    case 'softbreak':
      return [' ']
    case 'hardbreak':
      return [h('br')]
    default:
      // An image.
      return [h('img', { ...attrs(token), alt: getPlainText(token.children ?? []) })]
  }
}

/** The text of inline tokens with their marks removed. */
const getPlainText = (tokens: readonly Token[]): string =>
  tokens
    .map((token) => {
      if (token.type === 'image') return getPlainText(token.children ?? [])
      if (token.type === 'softbreak' || token.type === 'hardbreak') return ' '
      return token.content
    })
    .join('')

/** Text, cut into words with the spaces between them kept. */
const words = (text: string, next: () => number): (VNode | string)[] =>
  text
    .split(/(\s+)/)
    .filter((piece) => piece !== '')
    .map((piece) => (/^\s+$/.test(piece) ? piece : h('span', { key: next(), class: WORD }, piece)))

const attrs = (token: Token, deadAddresses: BrokenAddresses = NONE): Record<string, string> => {
  const content: Record<string, string> = Object.fromEntries(
    (token.attrs ?? []).map(([name, value]) => [name, String(value)]),
  )
  const href = content.href
  if (href !== undefined && deadAddresses.has(href)) content[REACHES] = 'nothing'
  return content
}
