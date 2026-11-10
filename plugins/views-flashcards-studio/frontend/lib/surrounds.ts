/**
 * The prose around the part a card tab edits, preserved unchanged. A note
 * belongs to its user, and the text above the first card and below the last is
 * written back as it was read.
 */
export interface Surrounds {
  readonly preamble: string
  readonly tail: string
}
