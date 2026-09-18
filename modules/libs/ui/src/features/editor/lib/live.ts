/**
 * When the marks are drawn again, and over which lines.
 *
 * A widget standing for whole lines is held in the state; everything else is
 * drawn for the lines on screen with a margin either side of them.
 */
import { syntaxTree } from '@codemirror/language'
import { StateField } from '@codemirror/state'
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from '@codemirror/view'
import { blockMarks, getMarks } from './marks'

/**
 * How far past the lines on screen the marks are drawn, in characters. A pane
 * being resized moves the viewport a line at a time, and marks already drawn
 * answer for the lines it reaches.
 */
const margin = 2000

/** The lines the viewport shows, whole. */
const onScreen = (view: EditorView) => {
  const ranges = view.visibleRanges
  if (!ranges.length) return null
  const first = ranges[0]
  const last = ranges[ranges.length - 1]
  if (!first || !last) return null
  const doc = view.state.doc
  return { from: doc.lineAt(first.from).from, to: doc.lineAt(last.to).to }
}

/** Those lines with a margin of them either side, which is what is drawn for. */
const getDrawnRange = (view: EditorView) => {
  const seen = onScreen(view)
  if (!seen) return null
  const doc = view.state.doc
  return {
    from: doc.lineAt(Math.max(seen.from - margin, 0)).from,
    to: doc.lineAt(Math.min(seen.to + margin, doc.length)).to,
  }
}

/** A widget standing for whole lines is the state's to hold, not a plugin's. */
export const wholeLines = StateField.define<DecorationSet>({
  create: (state) => blockMarks(state),
  update: (was, transaction) =>
    transaction.docChanged ||
    transaction.selection ||
    syntaxTree(transaction.startState) != syntaxTree(transaction.state)
      ? blockMarks(transaction.state)
      : was,
  provide: (field) => [
    EditorView.decorations.from(field),
    EditorView.atomicRanges.of((view) => view.state.field(field)),
  ],
})

export const live = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    /** The span the marks in hand were drawn for. */
    span: { from: number; to: number } | null

    constructor(view: EditorView) {
      this.span = getDrawnRange(view)
      this.decorations = this.span
        ? getMarks(view.state, this.span.from, this.span.to)
        : Decoration.none
    }

    update(update: ViewUpdate) {
      // A parse finishes in chunks, and the transaction that announces one
      // changes neither the document nor the selection. Past the first chunk
      // the document is drawn as it is written until this is asked.
      const afresh =
        update.docChanged ||
        update.selectionSet ||
        syntaxTree(update.startState) != syntaxTree(update.state)

      // The lines on screen, against the wider span the marks in hand were
      // drawn for. A viewport still inside them is already drawn.
      const seen = onScreen(update.view)
      if (!afresh && this.span && seen && seen.from >= this.span.from && seen.to <= this.span.to) {
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
