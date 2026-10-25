/**
 * Resolves destination addresses for links and wikilinks in editor text.
 */
import { syntaxTree } from '@codemirror/language'
import type { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'
import { findWikilink } from '@/shared/lib/text/address'
import { addressOpener } from '../outside'

/** Finds the first child node matching the specified syntax type. */
export const findChild = (node: SyntaxNode, name: string): SyntaxNode | null => {
  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.name === name) return child
  }
  return null
}

/** Whether the syntax node is enclosed within code formatting. */
const isCode = (node: SyntaxNode): boolean => {
  for (let one: SyntaxNode | null = node; one; one = one.parent) {
    if (one.name === 'FencedCode' || one.name === 'CodeBlock' || one.name === 'InlineCode') {
      return true
    }
  }
  return false
}

/**
 * Returns destination address of the markdown link or wikilink at the specified document offset.
 */
export const getLinkAddressAt = (state: EditorState, pos: number): string | null => {
  const innermost = syntaxTree(state).resolveInner(pos, 1)
  if (!isCode(innermost)) {
    const line = state.doc.lineAt(pos)
    const wiki = findWikilink(line.text, pos - line.from)
    if (wiki) return wiki.address
  }

  for (let node: SyntaxNode | null = innermost; node; node = node.parent) {
    if (node.name !== 'Link' && node.name !== 'Autolink') continue
    const address = findChild(node, 'URL')
    return address
      ? state.doc.sliceString(address.from, address.to)
      : state.doc.sliceString(node.from + 1, node.to - 1)
  }
  return null
}

/** Handles ctrl/cmd click on links to open their destination addresses. */
export const follow = EditorView.domEventHandlers({
  mousedown(event, view) {
    if (!event.metaKey && !event.ctrlKey) return false
    const pos = view.posAtCoords({ x: event.clientX, y: event.clientY })
    if (pos === null) return false

    const targetAddress = getLinkAddressAt(view.state, pos)
    if (targetAddress === null) return false
    view.state.facet(addressOpener)(targetAddress)
    return true
  },
})
