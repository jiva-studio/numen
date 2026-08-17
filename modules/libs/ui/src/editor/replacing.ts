/**
 * Putting a document in over the one that is there.
 *
 * The change covers the lines that differ and nothing else, so the lines
 * around it keep their positions and the view keeps its scroll offset.
 *
 * A position is held as a line and a column, and comes back at that line and
 * column. Where the line is shorter than the column, the position is the end
 * of the line; where the text has fewer lines, it is the last of them. The
 * change is annotated out of the undo history.
 */
import {
  EditorSelection,
  Transaction,
  type ChangeSpec,
  type EditorState,
  type Text,
  type TransactionSpec,
} from '@codemirror/state'

/** Where a position sits in the text a person sees. */
interface Place {
  readonly line: number
  readonly column: number
}

const placeOf = (doc: Text, at: number): Place => {
  const line = doc.lineAt(at)
  return { line: line.number, column: at - line.from }
}

const positionOf = (doc: Text, place: Place): number => {
  const line = doc.line(Math.min(place.line, doc.lines))
  return Math.min(line.from + place.column, line.to)
}

/** The stretch between the lines the two texts share, as one change. */
const difference = (was: Text, now: Text): ChangeSpec => {
  const most = Math.min(was.lines, now.lines)

  let head = 0
  while (head < most && was.line(head + 1).text === now.line(head + 1).text) head += 1

  let tail = 0
  while (tail < most - head && was.line(was.lines - tail).text === now.line(now.lines - tail).text) {
    tail += 1
  }

  // The stretch runs from the end of the last line the two share to the break
  // before the first of the lines they share again.
  const opens = (doc: Text) => (head === 0 ? 0 : doc.line(head).to)
  const closes = (doc: Text) => (tail === 0 ? doc.length : doc.line(doc.lines - tail + 1).from - 1)

  return { from: opens(was), to: closes(was), insert: now.sliceString(opens(now), closes(now)) }
}

/** The text of `fresh`, put in over what the state holds. */
export const replacing = (state: EditorState, fresh: string): TransactionSpec => {
  const now = state.toText(fresh)
  const ranges = state.selection.ranges.map((range) =>
    EditorSelection.range(
      positionOf(now, placeOf(state.doc, range.anchor)),
      positionOf(now, placeOf(state.doc, range.head)),
    ),
  )

  return {
    changes: difference(state.doc, now),
    selection: EditorSelection.create(ranges, state.selection.mainIndex),
    annotations: Transaction.addToHistory.of(false),
  }
}
