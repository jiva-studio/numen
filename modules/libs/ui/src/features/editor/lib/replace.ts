/**
 * Putting a document in over the one that is there. The change covers the
 * stretch that differs and nothing else, and is annotated out of the undo
 * history.
 *
 * A position is held as a line and a column: where the line is shorter than the
 * column it is the end of the line, and where the text has fewer lines it is
 * the last of them.
 */
import {
  EditorSelection,
  Transaction,
  type ChangeSpec,
  type EditorState,
  type Text,
  type TransactionSpec,
} from '@codemirror/state'
import { isLeading, isTrailing } from '@/shared/lib/surrogate'

/** Where a position sits in the text a person sees. */
interface Place {
  readonly line: number
  readonly column: number
}

const getLineColumn = (doc: Text, at: number): Place => {
  const line = doc.lineAt(at)
  return { line: line.number, column: at - line.from }
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
  // A pair standing for one character is not cut between its halves.
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

/** The text of `fresh`, put in over what the state holds. */
export const replace = (state: EditorState, fresh: string): TransactionSpec => {
  const now = state.toText(fresh)
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
