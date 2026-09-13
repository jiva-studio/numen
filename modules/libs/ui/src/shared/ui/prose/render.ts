/**
 * Markdown as Vue's own nodes, read by markdown-it. Vue patches the tree it
 * drew before, so prose arriving a piece at a time keeps what is on the screen,
 * and a word that has just arrived can be shown arriving.
 *
 * HTML in the text is text.
 */
import MarkdownIt, { type Token } from 'markdown-it'
import { h, type VNode } from 'vue'
import { wikilinks } from '@/shared/lib/marks'

const marks = new MarkdownIt({ html: false, linkify: true }).use(wikilinks)

/** The class every word is drawn in. */
export const WORD = 'prose__word'

/** The attribute a link that reaches nothing is drawn with. */
const REACHES = 'data-reaches'

/** The addresses a text points at that reach nothing. */
export type BrokenAddresses = ReadonlySet<string>

const NONE: BrokenAddresses = new Set()

export const render = (text: string, deadAddresses: BrokenAddresses = NONE): VNode[] =>
  nodes(marks.parse(text, {}), deadAddresses)

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
  const root: Frame = { tag: '', attrs: {}, children: [] }
  const stack: Frame[] = [root]
  const top = () => stack[stack.length - 1]!
  let placed = 0

  for (const token of tokens) {
    // A hidden token is one the marks imply and the reader never sees: the
    // paragraph markdown puts inside a tight list item.
    if (token.hidden) continue

    if (token.nesting === 1) {
      stack.push({ tag: token.tag, attrs: attrs(token, deadAddresses), children: [] })
      continue
    }
    if (token.nesting === -1) {
      const done = stack.pop()
      if (!done) continue
      top().children.push(h(done.tag, done.attrs, done.children))
      continue
    }

    top().children.push(...block(token, deadAddresses, () => placed++))
  }

  // A stream stops mid-sentence, so elements are left open. They are closed
  // here in the order they were opened.
  while (stack.length > 1) {
    const done = stack.pop()!
    top().children.push(h(done.tag, done.attrs, done.children))
  }
  return root.children.filter((child): child is VNode => typeof child !== 'string')
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
    case 'fence':
    case 'code_block':
      return [h('pre', attrs(token), [h('code', token.content)])]
    case 'hr':
      return [h('hr')]
    case 'html_block':
      return [token.content]
    default:
      return token.content ? [token.content] : []
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
  const root: Frame = { tag: '', attrs: {}, children: [] }
  const stack: Frame[] = [root]
  const top = () => stack[stack.length - 1]!

  for (const token of tokens) {
    if (token.nesting === 1) {
      stack.push({ tag: token.tag, attrs: attrs(token, deadAddresses), children: [] })
      continue
    }
    if (token.nesting === -1) {
      const done = stack.pop()
      if (!done) continue
      top().children.push(h(done.tag, done.attrs, done.children))
      continue
    }

    top().children.push(...mark(token, next))
  }

  while (stack.length > 1) {
    const done = stack.pop()!
    top().children.push(h(done.tag, done.attrs, done.children))
  }
  return root.children
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
    case 'image':
      return [h('img', attrs(token))]
    case 'html_inline':
      return [token.content]
    default:
      return token.content ? [token.content] : []
  }
}

/** Text, cut into words with the spaces between them kept. */
const words = (text: string, next: () => number): (VNode | string)[] =>
  text
    .split(/(\s+)/)
    .filter((piece) => piece !== '')
    .map((piece) =>
      /^\s+$/.test(piece) ? piece : h('span', { key: next(), class: WORD }, piece),
    )

const attrs = (token: Token, deadAddresses: BrokenAddresses = NONE): Record<string, string> => {
  const written: Record<string, string> = Object.fromEntries(
    (token.attrs ?? []).map(([name, value]) => [name, String(value)]),
  )
  const href = written.href
  if (href !== undefined && deadAddresses.has(href)) written[REACHES] = 'nothing'
  return written
}
