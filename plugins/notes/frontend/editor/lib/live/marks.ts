/**
 * What each mark of the markdown is drawn as.
 *
 * is shown as it is written. Standing in it means a selection touches it, so
 * the getMarks come back under the caret and nowhere else. Nothing here changes
 * the text: the file is what was typed, mark for mark.
 */
import { syntaxTree } from '@codemirror/language'
import type { EditorState, Range, Text } from '@codemirror/state'
import { Decoration, type DecorationSet, type WidgetType } from '@codemirror/view'
import type { SyntaxNode, SyntaxNodeRef } from '@lezer/common'
import { findChild } from '../extensions/address'
import { getGrid } from '../table/grid'
import { BULLET, HEADING, HIDDEN, INLINE, line, mark, SHY } from './styles'
import { Box, Bullet, Picture, Rule } from './widgets'

/** The block nodes that hold other blocks; a Table or a HorizontalRule is allowed only in these. */
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
 * A widget replacing whole lines must come from the state; everything else is drawn
 * for the lines on screen. `pass` says which of the two this pass is collecting.
 */
interface MarkSpan {
  readonly from: number
  readonly to: number
}

class MarkBuilder {
  readonly found: Range<Decoration>[] = []
  private readonly doc: Text
  private readonly blocks: boolean

  constructor(
    private readonly state: EditorState,
    pass: 'blocks' | 'getMarks',
  ) {
    this.doc = state.doc
    this.blocks = pass === 'blocks'
  }

  private isSelected(start: number, end: number): boolean {
    return this.state.selection.ranges.some((range) => range.from <= end && range.to >= start)
  }

  private isOutsideSelection(node: SyntaxNodeRef): boolean {
    return !this.isSelected(node.from, node.to)
  }

  private isParentSelected(node: SyntaxNodeRef): boolean {
    const parent = node.node.parent as SyntaxNode
    return this.isSelected(parent.from, parent.to)
  }

  private hide(start: number, end: number): void {
    const after = this.doc.sliceString(end, end + 1) === ' ' ? end + 1 : end
    this.found.push(HIDDEN.range(start, after))
  }

  private eachLine(
    start: number,
    end: number,
    add: (pos: number, first: boolean, isLast: boolean) => void,
  ): void {
    for (let offset = start, first = true; ; first = false) {
      const row = this.doc.lineAt(offset)
      const isLast = row.to >= end
      add(row.from, first, isLast)
      if (isLast) break
      offset = row.to + 1
    }
  }

  private block(node: SyntaxNodeRef, widget: WidgetType): Range<Decoration> {
    return Decoration.replace({ widget, block: true }).range(
      this.doc.lineAt(node.from).from,
      this.doc.lineAt(node.to).to,
    )
  }

  takeBlock(node: SyntaxNodeRef): boolean | null {
    switch (node.name) {
      case 'HorizontalRule':
        if (this.blocks && this.isOutsideSelection(node))
          this.found.push(this.block(node, new Rule()))
        return false

      case 'Table':
        if (!this.isOutsideSelection(node)) return !this.blocks
        if (this.blocks) this.found.push(this.block(node, getGrid(this.state, node.node)))
        return false
    }
    return null
  }

  takeLine(node: SyntaxNodeRef): boolean | null {
    const heading = HEADING.exec(node.name)
    if (heading) {
      this.found.push(
        line(`cm-heading cm-heading-${heading[1]}`).range(this.doc.lineAt(node.from).from),
      )
      return true
    }

    if (node.name === 'Blockquote') {
      this.eachLine(node.from, node.to, (pos) => this.found.push(line('cm-quote').range(pos)))
      return true
    }

    if (node.name === 'FencedCode') {
      this.eachLine(node.from, node.to, (pos, first, isLast) => {
        const edge = `${first ? ' cm-code-first' : ''}${isLast ? ' cm-code-last' : ''}`
        this.found.push(line(`cm-code${edge}`).range(pos))
      })
      return true
    }

    return null
  }

  takeWidget(node: SyntaxNodeRef): boolean | null {
    switch (node.name) {
      case 'ListMark': {
        if (BULLET.test(this.doc.sliceString(node.from, node.to)))
          this.found.push(Decoration.replace({ widget: new Bullet() }).range(node.from, node.to))
        return false
      }

      case 'TaskMarker': {
        const isDone = this.doc.sliceString(node.from, node.to).toLowerCase() === '[x]'
        const box = new Box(isDone, !this.state.readOnly)
        this.found.push(Decoration.replace({ widget: box }).range(node.from, node.to))
        return false
      }
    }
    return null
  }

  takeImage(node: SyntaxNodeRef): boolean | null {
    if (node.name !== 'Image') return null

    const address = findChild(node.node, 'URL')
    if (!address || !this.isOutsideSelection(node)) return true
    this.found.push(
      Decoration.replace({
        widget: new Picture(this.doc.sliceString(address.from, address.to)),
      }).range(node.from, node.to),
    )
    return false
  }

  takeShy(node: SyntaxNodeRef): boolean | null {
    if (SHY.has(node.name)) {
      if (!this.isParentSelected(node)) this.found.push(HIDDEN.range(node.from, node.to))
      return false
    }

    if (node.name === 'CodeMark') {
      if (node.node.parent?.name === 'InlineCode' && !this.isParentSelected(node))
        this.found.push(HIDDEN.range(node.from, node.to))
      return false
    }

    return null
  }

  isSpacedMark(node: SyntaxNodeRef): boolean | null {
    if (node.name === 'HeaderMark') {
      if ((node.node.parent as SyntaxNode).name.startsWith('Setext')) return false
      if (!this.isParentSelected(node)) this.hide(node.from, node.to)
      return false
    }

    if (node.name === 'QuoteMark') {
      if (!this.isParentSelected(node)) this.hide(node.from, node.to)
      return false
    }

    return null
  }

  takeInline(node: SyntaxNodeRef): boolean | null {
    const name = INLINE.get(node.name)
    if (!name) return null

    this.found.push(mark(name).range(node.from, node.to))
    return true
  }

  enter(node: SyntaxNodeRef): boolean {
    const asBlock = this.takeBlock(node)
    if (asBlock !== null) return asBlock
    if (this.blocks) return BLOCK_CONTAINERS.has(node.name)

    return (
      this.takeLine(node) ??
      this.takeWidget(node) ??
      this.takeImage(node) ??
      this.takeShy(node) ??
      this.isSpacedMark(node) ??
      this.takeInline(node) ??
      true
    )
  }
}

const build = (state: EditorState, span: MarkSpan, pass: 'blocks' | 'getMarks'): DecorationSet => {
  const builder = new MarkBuilder(state, pass)
  syntaxTree(state).iterate({
    from: span.from,
    to: span.to,
    enter: (node) => builder.enter(node),
  })

  return Decoration.set(builder.found, true)
}

/** What is drawn inside the lines: the marks, and the widgets that replace them. */
export const getMarks = (state: EditorState, from: number, to: number) =>
  build(state, { from, to }, 'getMarks')

/** What is drawn in place of whole lines: a rule, a table. */
export const blockMarks = (state: EditorState) =>
  build(state, { from: 0, to: state.doc.length }, 'blocks')

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
