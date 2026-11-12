/**
 * Conversions of the plain fields several answers of the core share, into the window's own words.
 */

/** What a note is, where the schema leaves it empty for a plain note. */
export const mapNoteType = (of: string): string => of || 'note'

/** Copies a wire range into the plain pair the window carries. */
export const mapRange = (range: { start: number; end: number }) => ({
  start: range.start,
  end: range.end,
})

/** What a file did when it moved, in the shape the window carries it. */
export const mapMoveResult = (answer: {
  readonly from: string
  readonly to: string
  readonly repaired: readonly string[]
}) => ({
  from: answer.from,
  to: answer.to,
  repaired: answer.repaired,
})
