/**
 * Markdown as Vue's own nodes.
 *
 * The marks are read by markdown-it and turned into elements here. Two things
 * follow, and both matter for prose that arrives a piece at a time:
 *
 * - Vue patches the tree it drew before, so what is on the screen stays.
 * - a word that has just arrived is a node that has just arrived, and can be
 *   shown arriving.
 *
 * HTML in the text is text.
 */
import MarkdownIt, { type Token } from 'markdown-it'
import { h, type VNode } from 'vue'

const marks = new MarkdownIt({ html: false, linkify: true })

/** The class every word is drawn in. */
export const WORD = 'prose__word'

export const render = (text: string): VNode[] => nodes(marks.parse(text, {}))

/**
 * Tokens arrive flat, with the nesting written on them. A frame is one element
 * being filled; the tokens that close it hand it to the frame beneath.
 */
interface Frame {
  tag: string
  attrs: Record<string, string>
  children: (VNode | string)[]
}

const nodes = (tokens: readonly Token[]): VNode[] => {
  const root: Frame = { tag: '', attrs: {}, children: [] }
  const stack: Frame[] = [root]
  const top = () => stack[stack.length - 1]!
  let placed = 0

  for (const token of tokens) {
    // A hidden token is one the marks imply and the reader never sees: the
    // paragraph markdown puts inside a tight list item.
    if (token.hidden) continue

    if (token.nesting === 1) {
      stack.push({ tag: token.tag, attrs: attrs(token), children: [] })
      continue
    }
    if (token.nesting === -1) {
      const done = stack.pop()
      if (!done) continue
      top().children.push(h(done.tag, done.attrs, done.children))
      continue
    }

    switch (token.type) {
      case 'inline':
        top().children.push(...inline(token.children ?? [], () => placed++))
        break
      case 'fence':
      case 'code_block':
        top().children.push(h('pre', attrs(token), [h('code', token.content)]))
        break
      case 'hr':
        top().children.push(h('hr'))
        break
      case 'html_block':
        top().children.push(token.content)
        break
      default:
        if (token.content) top().children.push(token.content)
    }
  }

  // A stream stops mid-sentence, so elements are left open. They are closed
  // here in the order they were opened.
  while (stack.length > 1) {
    const done = stack.pop()!
    top().children.push(h(done.tag, done.attrs, done.children))
  }
  return root.children.filter((child): child is VNode => typeof child !== 'string')
}

/**
 * The inside of a paragraph: words, and the marks that dress them.
 *
 * Every word is a node of its own, keyed by where it falls in the answer.
 */
const inline = (tokens: readonly Token[], next: () => number): (VNode | string)[] => {
  const root: Frame = { tag: '', attrs: {}, children: [] }
  const stack: Frame[] = [root]
  const top = () => stack[stack.length - 1]!

  for (const token of tokens) {
    if (token.nesting === 1) {
      stack.push({ tag: token.tag, attrs: attrs(token), children: [] })
      continue
    }
    if (token.nesting === -1) {
      const done = stack.pop()
      if (!done) continue
      top().children.push(h(done.tag, done.attrs, done.children))
      continue
    }

    switch (token.type) {
      case 'text':
        top().children.push(...words(token.content, next))
        break
      case 'code_inline':
        top().children.push(h('code', { key: next() }, token.content))
        break
      case 'softbreak':
        top().children.push(' ')
        break
      case 'hardbreak':
        top().children.push(h('br'))
        break
      case 'image':
        top().children.push(h('img', attrs(token)))
        break
      case 'html_inline':
        top().children.push(token.content)
        break
      default:
        if (token.content) top().children.push(token.content)
    }
  }

  while (stack.length > 1) {
    const done = stack.pop()!
    top().children.push(h(done.tag, done.attrs, done.children))
  }
  return root.children
}

/** Text, cut into words with the spaces between them kept. */
const words = (text: string, next: () => number): (VNode | string)[] =>
  text
    .split(/(\s+)/)
    .filter((piece) => piece !== '')
    .map((piece) =>
      /^\s+$/.test(piece) ? piece : h('span', { key: next(), class: WORD }, piece),
    )

const attrs = (token: Token): Record<string, string> =>
  Object.fromEntries((token.attrs ?? []).map(([name, value]) => [name, String(value)]))
