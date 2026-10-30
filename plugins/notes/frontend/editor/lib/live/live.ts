/**
 * When the marks are drawn again, and over which lines.
 *
 * A widget standing for whole lines is held in the state; everything else is
 * drawn for the lines on screen with a margin either side of them.
 */
import { syntaxTree } from '@codemirror/language'
import {
  type EditorSelection,
  type EditorState,
  type Range,
  StateField,
  type Transaction,
} from '@codemirror/state'
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
  type WidgetType,
} from '@codemirror/view'
import { moveBlock } from '../table/grid'
import { type BlockNode, blockMarks, findBlocks, getMarks } from './marks'

/**
 * How far beyond the visible lines the marks are drawn, in characters. Resizing
 * a pane moves the viewport a line at a time, and marks already drawn cover the
 * lines the viewport moves into.
 */
const margin = 2000

/** The lines the viewport shows, whole. */
const onScreen = (view: EditorView) => {
  const ranges = view.visibleRanges
  const first = ranges[0]
  const last = ranges[ranges.length - 1]
  if (!first || !last) return null
  const doc = view.state.doc
  return { from: doc.lineAt(first.from).from, to: doc.lineAt(last.to).to }
}

/** Those lines with a margin of them either side, which is what is drawn for. */
const getDrawnRange = (view: EditorView) => {
  const observed = onScreen(view)
  if (!observed) return null
  const doc = view.state.doc
  return {
    from: doc.lineAt(Math.max(observed.from - margin, 0)).from,
    to: doc.lineAt(Math.min(observed.to + margin, doc.length)).to,
  }
}

/** The marks of the whole lines, and the tables and rules they stand for. */
interface Blocks {
  readonly marks: DecorationSet
  readonly nodes: readonly BlockNode[]
}

const createBlocks = (state: EditorState): Blocks => ({
  marks: blockMarks(state),
  nodes: findBlocks(state),
})

const isSelected = (selection: EditorSelection, node: BlockNode) =>
  selection.ranges.some((range) => range.from <= node.to && range.to >= node.from)

/** Whether a changed range overlaps the node. `start` is the offset in `ranges` where the range bounds begin. */
const isTouched = (node: BlockNode, ranges: readonly number[], start: 0 | 2) => {
  // Stryker disable next-line EqualityOperator: a read past the last range is undefined, and no comparison with it holds
  for (let i = 0; i < ranges.length; i += 4) {
    const from = ranges[i + start] as number
    const to = ranges[i + start + 1] as number
    if (from <= node.to && to >= node.from) return true
  }
  return false
}

/**
 * Whether the marks of `was` are still valid after the transaction: the same
 * tables and rules are written in the same places, no change overlaps any of
 * them, and the selection is in the same ones.
 */
const isUnchanged = (was: Blocks, nodes: readonly BlockNode[], tr: Transaction) => {
  if (was.nodes.length !== nodes.length) return false

  const ranges: number[] = []
  tr.changes.iterChangedRanges((...bounds) => {
    ranges.push(...bounds)
  })

  return nodes.every((node, index) => {
    const before = was.nodes[index] as BlockNode
    return (
      before.name === node.name &&
      tr.changes.mapPos(before.from, 1) === node.from &&
      // Stryker disable next-line UnaryOperator: a position that moves with the assoc has a change at it, and isTouched answers for that
      tr.changes.mapPos(before.to, -1) === node.to &&
      !isTouched(before, ranges, 0) &&
      !isTouched(node, ranges, 2) &&
      isSelected(tr.startState.selection, before) === isSelected(tr.state.selection, node)
    )
  })
}

/** The marks of `was`, standing where the changes have carried them. */
const mapBlocks = (was: Blocks, nodes: readonly BlockNode[], tr: Transaction): Blocks => {
  if (tr.changes.empty) return nodes === was.nodes ? was : { marks: was.marks, nodes }

  const moved: Range<Decoration>[] = []
  was.marks.map(tr.changes).between(0, tr.state.doc.length, (from, to, mark) => {
    const spec = mark.spec as { widget: WidgetType }
    const widget = moveBlock(spec.widget, tr.changes)
    moved.push(
      widget === spec.widget
        ? mark.range(from, to)
        : Decoration.replace({ widget, block: true }).range(from, to),
    )
  })
  // Stryker disable next-line BooleanLiteral: the marks are collected in document order, which is sorted
  return { marks: Decoration.set(moved, true), nodes }
}

/** A widget standing for whole lines is the state's to hold, not a plugin's. */
export const wholeLines = StateField.define<Blocks>({
  create: createBlocks,
  update: (was, tr) => {
    const isSameTree = syntaxTree(tr.startState) === syntaxTree(tr.state)
    if (!tr.docChanged && !tr.selection && isSameTree) return was

    const nodes = !tr.docChanged && isSameTree ? was.nodes : findBlocks(tr.state)
    return isUnchanged(was, nodes, tr) ? mapBlocks(was, nodes, tr) : createBlocks(tr.state)
  },
  provide: (field) => [
    EditorView.decorations.from(field, (blocks) => blocks.marks),
    EditorView.atomicRanges.of((view) => view.state.field(field).marks),
  ],
})

export const live = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    /** The span the current marks were drawn for. */
    span: { from: number; to: number } | null

    constructor(view: EditorView) {
      this.span = getDrawnRange(view)
      this.decorations = this.span
        ? getMarks(view.state, this.span.from, this.span.to)
        : Decoration.none
    }

    update(update: ViewUpdate) {
      // A parse completes in chunks, and the transaction announcing a chunk
      // changes neither the document nor the selection, so the syntax tree
      // comparison below detects it.
      const afresh =
        update.docChanged ||
        update.selectionSet ||
        syntaxTree(update.startState) !== syntaxTree(update.state)

      // The visible lines, compared with the wider span the current marks were
      // drawn for. A viewport inside that span is already drawn.
      const observed = onScreen(update.view)
      if (
        !afresh &&
        this.span &&
        observed &&
        observed.from >= this.span.from &&
        observed.to <= this.span.to
      ) {
        return
      }

      const span = getDrawnRange(update.view)
      this.span = span
      this.decorations = span ? getMarks(update.view.state, span.from, span.to) : Decoration.none
    }
  },
  {
    decorations: (plugin) => plugin.decorations,
    // A mark that is not drawn is not a place the caret can be put.
    provide: (plugin) =>
      EditorView.atomicRanges.of((view) => view.plugin(plugin)?.decorations ?? Decoration.none),
  },
)
