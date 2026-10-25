/**
 * Document text replacement preserving selection line and column positions.
 */
import {
  EditorSelection,
  Transaction,
  type ChangeSpec,
  type EditorState,
  type Text,
  type TransactionSpec,
} from '@codemirror/state'
import { isLeading, isTrailing } from '@/shared/lib/text/surrogate'

/** Line and column position in document text. */
interface Place {
  readonly line: number
  readonly column: number
}

const getLineColumn = (doc: Text, pos: number): Place => {
  const line = doc.lineAt(pos)
  return { line: line.number, column: pos - line.from }
}

const getOffset = (doc: Text, place: Place): number => {
  const line = doc.line(Math.min(place.line, doc.lines))
  return Math.min(line.from + place.column, line.to)
}

/**
 * The stretch the two texts differ over, as one change.
 *
 * Measured in characters from both ends, so what is replaced is the smallest
 * run that differs and every position outside it is where it was.
 */
const difference = (was: Text, now: Text): ChangeSpec => {
  const before = was.toString()
  const after = now.toString()
  const most = Math.min(before.length, after.length)

  let head = 0
  while (before.charCodeAt(head) === after.charCodeAt(head)) head += 1
  // Surrogate pairs representing a single character are not split.
  if (isLeading(before.charCodeAt(head - 1))) head -= 1

  let tail = 0
  while (
    tail < most - head &&
    before.charCodeAt(before.length - tail - 1) === after.charCodeAt(after.length - tail - 1)
  ) {
    tail += 1
  }
  if (isTrailing(before.charCodeAt(before.length - tail))) tail -= 1

  return {
    from: head,
    to: before.length - tail,
    insert: after.slice(head, after.length - tail),
  }
}

/** Replaces editor text while preserving selection line and column positions. */
export const replace = (state: EditorState, newText: string): TransactionSpec => {
  const now = state.toText(newText)
  const ranges = state.selection.ranges.map((range) =>
    EditorSelection.range(
      getOffset(now, getLineColumn(state.doc, range.anchor)),
      getOffset(now, getLineColumn(state.doc, range.head)),
    ),
  )

  return {
    changes: difference(state.doc, now),
    selection: EditorSelection.create(ranges, state.selection.mainIndex),
    annotations: Transaction.addToHistory.of(false),
  }
}
