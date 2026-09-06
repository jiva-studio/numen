/**
 * The address a position in the text stands in, and following it.
 *
 * A link written in brackets and a note in double brackets are two spellings
 * of the same thing, and inside code neither is one: what is written there is
 * an example of a link and not a link.
 */
import { syntaxTree } from '@codemirror/language'
import type { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'
import { wikilinkAt } from '../linking/address'
import { opening } from './outside'

/** The first child of a node under that name, and nothing where it has none. */
export const childOf = (node: SyntaxNode, name: string) => {
  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.name === name) return child
  }
  return null
}

/** Whether a position stands in code, where a link is an example of one. */
const coded = (node: SyntaxNode): boolean => {
  for (let one: SyntaxNode | null = node; one; one = one.parent) {
    if (one.name === 'FencedCode' || one.name === 'CodeBlock' || one.name === 'InlineCode') {
      return true
    }
  }
  return false
}

/**
 * The address a position in the text stands in, and nothing where it stands in
 * none. A note in brackets is read by the parser every link is read by.
 */
export const addressAt = (state: EditorState, at: number): string | null => {
  const innermost = syntaxTree(state).resolveInner(at, 1)
  if (!coded(innermost)) {
    const line = state.doc.lineAt(at)
    const wiki = wikilinkAt(line.text, at - line.from)
    if (wiki) return wiki.address
  }

  for (let node: SyntaxNode | null = innermost; node; node = node.parent) {
    if (node.name !== 'Link' && node.name !== 'Autolink') continue
    const address = childOf(node, 'URL')
    return address
      ? state.doc.sliceString(address.from, address.to)
      : state.doc.sliceString(node.from + 1, node.to - 1)
  }
  return null
}

/** A drawn link is followed with the platform's modifier held down. */
export const following = EditorView.domEventHandlers({
  mousedown(event, view) {
    if (!event.metaKey && !event.ctrlKey) return false
    const at = view.posAtCoords({ x: event.clientX, y: event.clientY })
    if (at === null) return false

    const address = addressAt(view.state, at)
    if (address === null) return false
    view.state.facet(opening)(address)
    event.preventDefault()
    return true
  },
})
