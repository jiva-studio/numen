/**
 * What each mark of the markdown is drawn as.
 *
 * A construct is shown as it reads until the person stands in it, and then it
 * is shown as it is written. Standing in it means a selection touches it, so
 * the getMarks come back under the caret and nowhere else. Nothing here changes
 * the text: the file is what was typed, mark for mark.
 */
import { syntaxTree } from '@codemirror/language'
import type { EditorState, Range } from '@codemirror/state'
import { Decoration, type DecorationSet, type WidgetType } from '@codemirror/view'
import type { SyntaxNode, SyntaxNodeRef } from '@lezer/common'
import { findChild } from './address'
import { getGrid } from './grid'
import { BULLET, HEADING, hidden, INLINE, line, mark, SHY } from './styles'
import { Box, Bullet, Picture, Rule } from './widgets'

/** The block nodes that hold other blocks; a Table or a HorizontalRule stands only in these. */
const BLOCK_CONTAINERS = new Set([
  'Document',
  'Blockquote',
  'BulletList',
  'OrderedList',
  'ListItem',
])

/** A node drawn as whole lines. */
export interface BlockNode {
  readonly name: 'Table' | 'HorizontalRule'
  readonly from: number
  readonly to: number
}

/**
 * What is drawn between `from` and `to`.
 *
 * A widget standing for whole lines has to come from the state; everything
 * else is drawn for the lines on screen. `pass` says which of the two this
 * pass is collecting.
 */
const build = (
  state: EditorState,
  from: number,
  to: number,
  pass: 'blocks' | 'getMarks',
): DecorationSet => {
  const found: Range<Decoration>[] = []
  const doc = state.doc
  const blocks = pass === 'blocks'

  const isSelected = (start: number, end: number) =>
    state.selection.ranges.some((range) => range.from <= end && range.to >= start)

  const isOutsideSelection = (node: SyntaxNodeRef) => !isSelected(node.from, node.to)

  const isParentSelected = (node: SyntaxNodeRef) => {
    const parent = node.node.parent as SyntaxNode
    return isSelected(parent.from, parent.to)
  }

  /** A mark and the space it is separated from its content by. */
  const hide = (start: number, end: number) => {
    const after = doc.sliceString(end, end + 1) === ' ' ? end + 1 : end
    found.push(hidden.range(start, after))
  }

  const eachLine = (
    start: number,
    end: number,
    add: (at: number, first: boolean, isLast: boolean) => void,
  ) => {
    for (let at = start, first = true; ; first = false) {
      const row = doc.lineAt(at)
      const isLast = row.to >= end
      add(row.from, first, isLast)
      if (isLast) break
      at = row.to + 1
    }
  }

  /** A widget standing for whole lines. */
  const block = (node: SyntaxNodeRef, widget: WidgetType) =>
    Decoration.replace({ widget, block: true }).range(
      doc.lineAt(node.from).from,
      doc.lineAt(node.to).to,
    )

  /** A widget standing for whole lines, and null where the node stands for none. */
  const takeBlock = (node: SyntaxNodeRef): boolean | null => {
    switch (node.name) {
      case 'HorizontalRule':
        if (blocks && isOutsideSelection(node)) found.push(block(node, new Rule()))
        // Stryker disable next-line BooleanLiteral: true descends into a HorizontalRule, which has no children
        return false

      case 'Table':
        if (!isOutsideSelection(node)) return !blocks
        if (blocks) found.push(block(node, getGrid(state, node.node)))
        return false
    }
    return null
  }

  /** The lines a construct colours: a heading, a quote, a fenced block. */
  const takeLine = (node: SyntaxNodeRef): boolean | null => {
    const heading = HEADING.exec(node.name)
    if (heading) {
      found.push(line(`cm-heading cm-heading-${heading[1]}`).range(doc.lineAt(node.from).from))
      return true
    }

    if (node.name === 'Blockquote') {
      eachLine(node.from, node.to, (at) => found.push(line('cm-quote').range(at)))
      return true
    }

    if (node.name === 'FencedCode') {
      eachLine(node.from, node.to, (at, first, isLast) => {
        const edge = `${first ? ' cm-code-first' : ''}${isLast ? ' cm-code-last' : ''}`
        found.push(line(`cm-code${edge}`).range(at))
      })
      // Stryker disable next-line BooleanLiteral: true descends into a FencedCode, which has no child that is drawn
      return true
    }

    return null
  }

  /** What stands in place of a mark: a bullet, a checkbox. */
  const takeWidget = (node: SyntaxNodeRef): boolean | null => {
    switch (node.name) {
      case 'ListMark': {
        if (BULLET.test(doc.sliceString(node.from, node.to)))
          found.push(Decoration.replace({ widget: new Bullet() }).range(node.from, node.to))
        // Stryker disable next-line BooleanLiteral: true descends into a ListMark, which has no children
        return false
      }

      case 'TaskMarker': {
        const done = doc.sliceString(node.from, node.to).toLowerCase() === '[x]'
        const box = new Box(done, !state.readOnly)
        found.push(Decoration.replace({ widget: box }).range(node.from, node.to))
        // Stryker disable next-line BooleanLiteral: true descends into a TaskMarker, which has no children
        return false
      }
    }
    return null
  }

  /** The picture an image is shown as while the selection is outside it. */
  const takeImage = (node: SyntaxNodeRef): boolean | null => {
    if (node.name !== 'Image') return null

    const address = findChild(node.node, 'URL')
    if (!address || !isOutsideSelection(node)) return true
    found.push(
      Decoration.replace({
        widget: new Picture(doc.sliceString(address.from, address.to)),
      }).range(node.from, node.to),
    )
    return false
  }

  /** A mark taken out of the drawing while the selection is outside what it belongs to. */
  const takeShy = (node: SyntaxNodeRef): boolean | null => {
    if (SHY.has(node.name)) {
      if (!isParentSelected(node)) found.push(hidden.range(node.from, node.to))
      // Stryker disable next-line BooleanLiteral: true descends into a mark of the shy set, which has no children
      return false
    }

    if (node.name === 'CodeMark') {
      if (node.node.parent?.name === 'InlineCode' && !isParentSelected(node))
        found.push(hidden.range(node.from, node.to))
      // Stryker disable next-line BooleanLiteral: true descends into a CodeMark, which has no children
      return false
    }

    return null
  }

  /** A mark taken out with the space that separates it from its content. */
  const isSpacedMark = (node: SyntaxNodeRef): boolean | null => {
    if (node.name === 'HeaderMark') {
      // Stryker disable next-line BooleanLiteral: true descends into a HeaderMark, which has no children
      if ((node.node.parent as SyntaxNode).name.startsWith('Setext')) return false
      if (!isParentSelected(node)) hide(node.from, node.to)
      // Stryker disable next-line BooleanLiteral: true descends into a HeaderMark, which has no children
      return false
    }

    if (node.name === 'QuoteMark') {
      if (!isParentSelected(node)) hide(node.from, node.to)
      // Stryker disable next-line BooleanLiteral: true descends into a QuoteMark, which has no children
      return false
    }

    return null
  }

  /** The class a stretch of text is drawn in: emphasis, code, a link. */
  const takeInline = (node: SyntaxNodeRef): boolean | null => {
    const name = INLINE.get(node.name)
    if (!name) return null

    found.push(mark(name).range(node.from, node.to))
    return true
  }

  syntaxTree(state).iterate({
    from,
    to,
    enter: (node) => {
      const asBlock = takeBlock(node)
      if (asBlock !== null) return asBlock
      if (blocks) return BLOCK_CONTAINERS.has(node.name)

      return (
        takeLine(node) ??
        takeWidget(node) ??
        takeImage(node) ??
        takeShy(node) ??
        isSpacedMark(node) ??
        takeInline(node) ??
        true
      )
    },
  })

  return Decoration.set(found, true)
}

/** What is drawn inside the lines: the getMarks, and what stands for them. */
export const getMarks = (state: EditorState, from: number, to: number) =>
  build(state, from, to, 'getMarks')

/** What is drawn in place of whole lines: a rule, a table. */
export const blockMarks = (state: EditorState) => build(state, 0, state.doc.length, 'blocks')

/** The tables and rules of the document, in order, with the stretch each one is written in. */
export const findBlocks = (state: EditorState): BlockNode[] => {
  const found: BlockNode[] = []
  syntaxTree(state).iterate({
    enter: (node) => {
      if (node.name === 'Table' || node.name === 'HorizontalRule') {
        found.push({ name: node.name, from: node.from, to: node.to })
        // Stryker disable next-line BooleanLiteral: true descends into a Table or HorizontalRule, which has no Table or HorizontalRule below it
        return false
      }
      return BLOCK_CONTAINERS.has(node.name)
    },
  })
  return found
}
