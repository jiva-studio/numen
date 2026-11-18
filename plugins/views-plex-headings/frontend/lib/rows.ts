import type { NoteHeading } from '@numen/plugin-sdk'
import type { HeadingRow } from '../types'

/** How many levels in a heading is ever set. */
const DEEPEST = 3

/**
 * The rows of the headings of a note, in the order they were given. A heading is set in by its rank among
 * the levels the note uses, so a note that starts at level 2 is not set in.
 */
export function getRows(headings: readonly NoteHeading[], indentStep: number): HeadingRow[] {
  const levels = [...new Set(headings.map((heading) => heading.level))].sort((a, b) => a - b)
  const indents = new Map(
    levels.map((level, rank) => [level, Math.min(rank, DEEPEST) * indentStep]),
  )
  return headings.map((heading) => ({
    line: heading.line,
    text: heading.text,
    indent: indents.get(heading.level) ?? 0,
  }))
}
